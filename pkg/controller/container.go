package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/events"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"watchlog/log/config"
	"watchlog/pkg/ctx"
	"watchlog/pkg/runtime"
	"watchlog/pkg/runtime/container"
)

const (
	ContainerLogPath = "/var/log/pods"
)

var (
	create = "containerd.events.ContainerCreate"
	delete = "containerd.events.ContainerDelete"
)

type RuntimeContainer struct {
	ctx *ctx.Context
}

type ContainerInterface interface {
	ProcessContainers() error
	NewContainer(meta containers.Container, process *specs.Process) error
}

func NewContainerInterface(ctx *ctx.Context) ContainerInterface {
	return &RuntimeContainer{
		ctx: ctx,
	}
}

func (c RuntimeContainer) ProcessContainers() error {
	c.ctx.Mutex.Lock()
	defer c.ctx.Mutex.Unlock()

	containerCtx := namespaces.WithNamespace(c.ctx.Context, "k8s.io")
	c.ProcessEvent(c.ctx, containerCtx)

	containers, err := c.ctx.ContainerCli.Containers(containerCtx)
	if err != nil {
		log.Errorf("get containers failed, %s", err.Error())
		return err
	}

	for _, v := range containers {
		meta, err := v.Info(containerCtx)
		if err != nil {
			log.Errorf("get container meta info failed: %s", err.Error())
		}

		spec, err := v.Spec(containerCtx)
		if err != nil {
			log.Errorf("get container spec failed: %v", err)
			continue
		}

		var b bool
		for _, envVar := range spec.Process.Env {
			serviceLogs := fmt.Sprintf(c.ctx.LogPrefix)
			if !strings.HasPrefix(envVar, serviceLogs) {
				b = false
				continue
			}

			b = true
			break
		}

		if b {
			err = c.NewContainer(meta, spec.Process)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (c RuntimeContainer) NewContainer(meta containers.Container, process *specs.Process) error {
	id := meta.ID
	env := process.Env
	labels := meta.Labels
	logFile := meta.Labels[runtime.KubernetesContainerNamespace] + "_" + meta.Labels[runtime.KubernetesPodName] + "_*" + "/" + meta.Labels[runtime.KubernetesContainerName] + "/" + "*.log"
	ct := container.Container(meta)

	for _, envVar := range env {
		// log_test 转换为 log.test
		envLabel := strings.SplitN(envVar, "=", 2)
		if len(envLabel) == 2 {
			labelKey := strings.Replace(envLabel[0], "_", ".", -1)
			labels[labelKey] = envLabel[1]
		}
	}

	logPath := filepath.Join(c.ctx.BaseDir, ContainerLogPath, logFile)
	configs, err := config.GetLogConfigs(c.ctx.LogPrefix, logPath, labels)
	if err != nil {
		log.Errorf("%v", err.Error())
		return err
	}

	if len(configs) == 0 {
		log.Debugf("%s has not log config, skip", id)
		return nil
	}
	// 生成 filebeat 采集配置
	logConfig, err := c.ctx.Piloter.RenderLogConfig(id, ct, configs)
	if err != nil {
		log.Errorf("%v", err.Error())
		return err
	}

	if err = ioutil.WriteFile(c.ctx.Piloter.GetConfPath(id), []byte(logConfig), os.FileMode(0644)); err != nil {
		log.Errorf("%v", err.Error())
		return err
	}

	//c.TryReload()
	return nil

}

func (c RuntimeContainer) ProcessEvent(ctx *ctx.Context, containerCtx context.Context) {
	msgs, errs := ctx.ContainerCli.EventService().Subscribe(containerCtx, "")

	go func() {
		defer func() {
			log.Warn("finish to watch event")
			ctx.StopChan <- true
		}()

		log.Info("begin to watch event")

		for {
			select {
			case msg := <-msgs:
				if err := c.processEvent(ctx, containerCtx, msg); err != nil {
					log.Errorf("fail to process event: %v,  %v", msg, err)
				}
			case err := <-errs:
				log.Warnf("error: %v", err)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return
				}
				msgs, errs = ctx.ContainerCli.EventService().Subscribe(containerCtx, "")
			}
		}
	}()
}

func (c RuntimeContainer) processEvent(ctx *ctx.Context, containerCtx context.Context, msg *events.Envelope) error {
	v := string(msg.Event.GetValue())
	s := strings.TrimPrefix(v, "\n@")
	containerId := removeSpecialChars(s)
	containerId = strings.Split(containerId, "-")[0]

	t := msg.Event.GetTypeUrl()
	switch t {
	case create:
		if Exists(ctx, containerId) {
			log.Debugf("%s is already exists", containerId)
			return nil
		}

		_, err := ctx.ContainerCli.LoadContainer(containerCtx, containerId)
		if err != nil {
			if errdefs.IsNotFound(err) {
				_, err = ctx.ContainerCli.LoadContainer(containerCtx, containerId)
			}

			return err
		}

		get, err := ctx.ContainerCli.ContainerService().Get(containerCtx, containerId)
		if err != nil {
			return err
		}

		var spec oci.Spec
		err = json.Unmarshal(get.Spec.GetValue(), &spec)
		if err != nil {
			return err
		}

		return c.NewContainer(get, spec.Process)

	case delete:
		log.Debugf("Process container destroy event: %s", containerId)

		err := DelContainerLogFile(ctx, containerId)
		if err != nil {
			log.Errorf("Process container destroy event error: %s, %s", containerId, err.Error())
		}

	}

	return nil
}

func removeSpecialChars(str string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return re.ReplaceAllString(str, "-")
}
