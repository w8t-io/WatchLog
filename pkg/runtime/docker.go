package runtime

import (
	"fmt"
	docker "github.com/docker/docker/client"
)

func NewDockerClient() *docker.Client {
	cli, err := docker.NewEnvClient()
	if err != nil {
		panic(fmt.Sprintf("Error: Create docker client failed, %s", err.Error()))
	}

	return cli
}
