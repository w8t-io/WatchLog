package ctx

import (
	"context"
	"github.com/containerd/containerd"
	"github.com/docker/docker/client"
	"os"
	"sync"
	"watchlog/pkg/provider"
	"watchlog/pkg/runtime"
)

type Context struct {
	context.Context
	// 采集器
	FilebeatPointer provider.FilebeatPointer
	// 日志前缀
	LogPrefix     string
	BaseDir       string
	DockerCli     *client.Client
	ContainerdCli *containerd.Client
	sync.Mutex
}

func NewContext(baseDir, logPrefix string, f provider.FilebeatPointer) *Context {
	dockerCli := new(client.Client)
	containerCli := new(containerd.Client)

	switch os.Getenv("RUNTIME_TYPE") {
	case "docker":
		dockerCli = runtime.NewDockerClient()
	case "containerd":
		containerCli = runtime.NewContainerClient()
	}

	return &Context{
		Context:         context.Background(),
		FilebeatPointer: f,
		LogPrefix:       logPrefix,
		BaseDir:         baseDir,
		DockerCli:       dockerCli,
		ContainerdCli:   containerCli,
	}
}
