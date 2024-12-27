package controller

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/zeromicro/go-zero/core/logc"
	"io"
	"strings"
	"watchlog/pkg/ctx"
)

type Docker struct {
	ctx *ctx.Context
	f   filters.Args
}

// NewDockerInterface creates a new Docker interface.
func NewDockerInterface(ctx *ctx.Context, f filters.Args) InterRuntime {
	return &Docker{ctx: ctx, f: f}
}

// ProcessContainers processes the Docker containers and logs.
func (d *Docker) ProcessContainers() error {
	d.ctx.Lock()
	defer d.ctx.Unlock()

	d.watchEvent(d.f)
	containers, err := d.listContainers()
	if err != nil {
		return err
	}

	for _, c := range containers {
		if c.State == "removing" {
			continue
		}

		if Exists(d.ctx, c.ID) {
			continue
		}

		if err := d.processContainer(c.ID); err != nil {
			logc.Errorf(context.Background(), fmt.Sprintf("Error processing container %s: %v", c.ID, err))
		}
	}
	return nil
}

// listContainers retrieves the list of Docker containers.
func (d *Docker) listContainers() ([]types.Container, error) {
	opts := types.ContainerListOptions{}
	containers, err := d.ctx.DockerCli.ContainerList(d.ctx, opts)
	if err != nil {
		logc.Errorf(context.Background(), fmt.Sprintf("Failed to list containers: %s", err.Error()))
		return nil, err
	}
	return containers, nil
}

// processContainer inspects and processes an individual container.
func (d *Docker) processContainer(containerID string) error {
	containerJSON, err := d.ctx.DockerCli.ContainerInspect(d.ctx, containerID)
	if err != nil {
		logc.Errorf(context.Background(), fmt.Sprintf("Failed to inspect container %s: %v", containerID, err))
		return err
	}

	if !Collect(containerJSON.Config.Env, d.ctx.LogPrefix) {
		return nil
	}

	// 符合条件的 Env
	var logEnvs []string
	for _, envVar := range containerJSON.Config.Env {
		// LogPrefix: aliyun_logs_tencent-prod-hermione=stdout ,envVar: aliyun_logs
		if strings.HasPrefix(envVar, d.ctx.LogPrefix) {
			logEnvs = append(logEnvs, envVar)
		}
	}

	fields := CollectFields{
		Id:      containerJSON.ID,
		Env:     logEnvs,
		Labels:  containerJSON.Config.Labels,
		LogPath: containerJSON.LogPath,
	}
	return NewCollectFile(d.ctx, fields)
}

// watchEvent listens for Docker events and processes them.
func (d *Docker) watchEvent(filter filters.Args) {
	options := types.EventsOptions{Filters: filter}
	msgs, errs := d.ctx.DockerCli.Events(d.ctx.Context, options)

	go func() {
		logc.Infof(context.Background(), "Beginning to watch docker events")
		for {
			select {
			case msg := <-msgs:
				if err := d.processEvent(msg); err != nil {
					logc.Errorf(context.Background(), fmt.Sprintf("Error processing event: %v", err))
				}
			case err := <-errs:
				logc.Errorf(context.Background(), fmt.Sprintf("Error in event stream: %v", err))
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return
				}
			}
		}
	}()
}

// processEvent handles Docker events for containers.
func (d *Docker) processEvent(msg events.Message) error {
	containerID := msg.Actor.ID
	switch msg.Action {
	case "start", "restart":
		return d.handleStartRestartEvent(containerID)
	case "destroy", "die":
		return d.handleDestroyDieEvent(containerID)
	default:
		return nil
	}
}

// handleStartRestartEvent processes container start/restart events.
func (d *Docker) handleStartRestartEvent(containerID string) error {
	logc.Debugf(context.Background(), "Processing container start/restart event: %s", containerID)
	if Exists(d.ctx, containerID) {
		logc.Debugf(context.Background(), "Container %s already exists, skipping", containerID)
		return nil
	}
	return d.processContainer(containerID)
}

// handleDestroyDieEvent processes container destroy/die events.
func (d *Docker) handleDestroyDieEvent(containerID string) error {
	if !Exists(d.ctx, containerID) {
		return nil
	}

	logc.Debugf(context.Background(), "Processing container destroy event: %s", containerID)
	return DelContainerLogFile(d.ctx, containerID)
}
