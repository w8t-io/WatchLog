package ctx

import (
	"context"
	"github.com/containerd/containerd"
	"github.com/docker/docker/client"
	"os"
	"watchlog/pkg/client/filebeat"
	"watchlog/pkg/runtime/container"
	"watchlog/pkg/runtime/docker"
	"watchlog/pkg/types"
)

type Context struct {
	types.Pilot
}

func NewContext(baseDir string, p filebeat.InterFilebeatPointer) *Context {
	dockerCli := new(client.Client)
	containerCli := new(containerd.Client)

	switch os.Getenv("RUNTIME_TYPE") {
	case "docker":
		dockerCli = docker.NewClient()
	case "containerd":
		containerCli = container.NewClient()
	}

	return &Context{
		types.Pilot{
			Context:      context.Background(),
			Client:       dockerCli,
			ContainerCli: containerCli,
			BaseDir:      baseDir,
			ReloadChan:   make(chan bool),
			StopChan:     make(chan bool),
			Piloter:      p,
			LogPrefix:    "watchlog",
		},
	}
}
