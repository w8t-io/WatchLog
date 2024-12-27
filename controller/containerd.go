package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/events"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/zeromicro/go-zero/core/logc"
	"io"
	"regexp"
	"strings"
	"watchlog/pkg/ctx"
	"watchlog/pkg/runtime"
)

var (
	create = "containerd.events.ContainerCreate"
	delete = "containerd.events.ContainerDelete"
)

type Containerd struct {
	ctx *ctx.Context
}

func NewContainerInterface(ctx *ctx.Context) InterRuntime {
	return &Containerd{
		ctx: ctx,
	}
}

func (c Containerd) ProcessContainers() error {
	c.ctx.Lock()
	defer c.ctx.Unlock()

	containerCtx := namespaces.WithNamespace(c.ctx.Context, "k8s.io")
	c.watchEvent(c.ctx, containerCtx)

	containers, err := c.ctx.ContainerdCli.Containers(containerCtx)
	if err != nil {
		logc.Errorf(context.Background(), fmt.Sprintf("get containers failed, %s", err.Error()))
		return err
	}

	for _, container := range containers {
		if err := c.processContainer(containerCtx, container); err != nil {
			logc.Errorf(context.Background(), "process container failed: %v", err)
		}
	}

	return nil
}

func (c Containerd) processContainer(containerCtx context.Context, container containerd.Container) error {
	meta, err := container.Info(containerCtx)
	if err != nil {
		return fmt.Errorf("get container meta info failed: %s", err.Error())
	}

	spec, err := container.Spec(containerCtx)
	if err != nil {
		return fmt.Errorf("get container spec failed: %s", err.Error())
	}

	return processCollectFile(c.ctx, spec.Process.Env, meta)
}

func (c Containerd) watchEvent(ctx *ctx.Context, containerCtx context.Context) {
	msgs, errs := c.ctx.ContainerdCli.EventService().Subscribe(containerCtx, "")

	go func() {
		defer logc.Info(context.Background(), "finish to watch containerd event")
		logc.Infof(context.Background(), "begin to watch containerd event")

		for {
			select {
			case msg := <-msgs:
				if err := c.processEvent(ctx, containerCtx, msg); err != nil {
					logc.Errorf(context.Background(), "process event failed: %v", err)
				}
			case err := <-errs:
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return
				}
				logc.Errorf(context.Background(), "event subscription error: %v", err)
			}
		}
	}()
}

func (c Containerd) processEvent(ctx *ctx.Context, containerCtx context.Context, msg *events.Envelope) error {
	v := string(msg.Event.GetValue())
	s := strings.TrimPrefix(v, "\n@")
	containerId := removeSpecialChars(s)
	containerId = strings.Split(containerId, "-")[0]

	t := msg.Event.GetTypeUrl()
	switch t {
	case create:
		if Exists(ctx, containerId) {
			return nil
		}

		_, err := ctx.ContainerdCli.LoadContainer(containerCtx, containerId)
		if err != nil {
			if errdefs.IsNotFound(err) {
				_, err = ctx.ContainerdCli.LoadContainer(containerCtx, containerId)
			}

			return err
		}

		meta, err := ctx.ContainerdCli.ContainerService().Get(containerCtx, containerId)
		if err != nil {
			return err
		}

		var spec oci.Spec
		err = json.Unmarshal(meta.Spec.GetValue(), &spec)
		if err != nil {
			return err
		}

		err = processCollectFile(ctx, spec.Process.Env, meta)
		if err != nil {
			return err
		}

		return err

	case delete:
		logc.Infof(context.Background(), "Process container destroy event: %s", containerId)

		err := DelContainerLogFile(ctx, containerId)
		if err != nil {
			logc.Errorf(context.Background(), fmt.Sprintf("Process container destroy event error: %s, %s", containerId, err.Error()))
		}
	}
	return nil
}

func processCollectFile(c *ctx.Context, envs []string, meta containers.Container) error {
	// 符合条件的 Env
	var logEnvs []string
	for _, envVar := range envs {
		// LogPrefix: aliyun_logs_tencent-prod-hermione=stdout ,envVar: aliyun_logs
		if strings.HasPrefix(envVar, c.LogPrefix) {
			logEnvs = append(logEnvs, envVar)
		}
	}

	fields := CollectFields{
		Id:      meta.ID,
		Env:     logEnvs,
		Labels:  meta.Labels,
		LogPath: fmt.Sprintf("%s_%s_*/%s/*.log", meta.Labels[runtime.KubernetesContainerNamespace], meta.Labels[runtime.KubernetesPodName], meta.Labels[runtime.KubernetesContainerName]),
	}
	return NewCollectFile(c, fields)
}

func removeSpecialChars(str string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return re.ReplaceAllString(str, "-")
}
