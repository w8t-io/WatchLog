package worker

import (
	"github.com/docker/docker/api/types/filters"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
	"watchlog/pkg/controller"
	"watchlog/pkg/ctx"
)

type Worker struct {
	ctx *ctx.Context
}

func NewWorker(c *ctx.Context) Worker {
	return Worker{
		ctx: c,
	}
}

func (w Worker) Run() error {
	// 清理旧配置
	if err := w.ctx.CleanConfigs(); err != nil {
		return err
	}

	// 启动 filebeat
	err := w.ctx.Piloter.Start()
	if err != nil {
		return err
	}

	w.ctx.LastReload = time.Now()
	//go w.ctx.DoReload()

	filter := filters.NewArgs()
	filter.Add("type", "container")

	switch os.Getenv("RUNTIME_TYPE") {
	case "docker":
		log.Info("Process docker runtime")
		c := controller.NewDockerInterface(w.ctx, filter)
		err := c.ProcessContainers(w.ctx)
		if err != nil {
			return err
		}
	case "containerd":
		log.Info("Process container runtime")
		c := controller.NewContainerInterface(w.ctx)
		err := c.ProcessContainers()
		if err != nil {
			return err
		}
	}

	<-w.ctx.StopChan
	close(w.ctx.ReloadChan)
	close(w.ctx.StopChan)
	return nil
}
