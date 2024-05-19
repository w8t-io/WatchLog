package controller

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	logtypes "watchlog/log/config"
	"watchlog/pkg/ctx"
	"watchlog/pkg/runtime/docker"
)

const (
	EnvServiceLogsTmpl = "%s_"
)

type RuntimeDocker struct {
	ctx *ctx.Context
	f   filters.Args
}

type DockerInterface interface {
	ProcessContainers(ctx *ctx.Context) error
	NewContainer(containerJSON *types.ContainerJSON) error
}

func NewDockerInterface(ctx *ctx.Context, f filters.Args) DockerInterface {
	return &RuntimeDocker{
		ctx: ctx,
		f:   f,
	}
}

func (d RuntimeDocker) ProcessContainers(ctx *ctx.Context) error {
	log.Debug("process all container log config")
	ctx.Mutex.Lock()
	defer ctx.Mutex.Unlock()

	d.ProcessEvent(d.f)

	opts := types.ContainerListOptions{}
	containers, err := ctx.Client.ContainerList(ctx.Context, opts)
	if err != nil {
		log.Errorf("fail to list container: %v", err)
		return err
	}

	for _, c := range containers {
		if c.State == "removing" {
			continue
		}

		if Exists(ctx, c.ID) {
			log.Debugf("%s is already exists", c.ID)
			continue
		}

		containerJSON, err := ctx.Client.ContainerInspect(ctx.Context, c.ID)
		if err != nil {
			log.Errorf("fail to inspect container %s: %v", c.ID, err)
			continue
		}

		if err = d.NewContainer(&containerJSON); err != nil {
			log.Errorf("fail to process container %s: %v", containerJSON.Name, err)
		}
	}

	return nil
}

func (d RuntimeDocker) NewContainer(containerJSON *types.ContainerJSON) error {
	id := containerJSON.ID
	env := containerJSON.Config.Env
	labels := containerJSON.Config.Labels
	jsonLogPath := containerJSON.LogPath
	ct := docker.Container(containerJSON)

	for _, e := range env {
		serviceLogs := fmt.Sprintf(EnvServiceLogsTmpl, d.ctx.LogPrefix)
		if !strings.HasPrefix(e, serviceLogs) {
			continue
		}

		// log_test 转换为 log.test
		envLabel := strings.SplitN(e, "=", 2)
		if len(envLabel) == 2 {
			labelKey := strings.Replace(envLabel[0], "_", ".", -1)
			labels[labelKey] = envLabel[1]
		}
	}

	logPath := filepath.Join(d.ctx.BaseDir, jsonLogPath)
	logConfigs, err := logtypes.GetLogConfigs(d.ctx.LogPrefix, logPath, labels)
	if err != nil {
		return err
	}

	if len(logConfigs) == 0 {
		log.Debugf("%s has not log config, skip", id)
		return nil
	}

	//生成 filebeat 采集配置
	logConfig, err := d.ctx.Piloter.RenderLogConfig(id, ct, logConfigs)
	if err != nil {
		return err
	}
	//TODO validate config before save
	if err = ioutil.WriteFile(d.ctx.Piloter.GetConfPath(id), []byte(logConfig), os.FileMode(0644)); err != nil {
		return err
	}

	//c.TryReload()
	return nil
}

func (d RuntimeDocker) ProcessEvent(filter filters.Args) {
	options := types.EventsOptions{
		Filters: filter,
	}

	msgs, errs := d.ctx.Client.Events(d.ctx.Context, options)

	go func() {
		defer func() {
			log.Warn("finish to watch event")
			d.ctx.StopChan <- true
		}()

		log.Info("begin to watch event")

		for {
			select {
			case msg := <-msgs:
				if err := d.processEvent(d.ctx, msg); err != nil {
					log.Errorf("fail to process event: %v,  %v", msg, err)
				}
			case err := <-errs:
				log.Warnf("error: %v", err)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return
				}
				msgs, errs = d.ctx.Client.Events(d.ctx.Context, options)
			}
		}
	}()
}

func (d RuntimeDocker) processEvent(ctx *ctx.Context, msg events.Message) error {
	containerId := msg.Actor.ID

	switch msg.Action {
	case "start", "restart":
		log.Debugf("Process container start event: %s", containerId)

		if Exists(ctx, containerId) {
			log.Debugf("%s is already exists", containerId)
			return nil
		}

		containerJSON, err := ctx.Client.ContainerInspect(ctx.Context, containerId)
		if err != nil {
			return err
		}

		return d.NewContainer(&containerJSON)
	case "destroy", "die":
		log.Debugf("Process container destroy event: %s", containerId)

		err := DelContainerLogFile(ctx, containerId)
		if err != nil {
			log.Warnf("Process container destroy event error: %s, %s", containerId, err.Error())
		}
	}

	return nil
}
