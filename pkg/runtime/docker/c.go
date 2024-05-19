package docker

import (
	"fmt"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"os"
	"strings"
	"watchlog/pkg/runtime"
)

func NewClient() *docker.Client {
	cli, err := docker.NewEnvClient()
	if err != nil {
		panic(fmt.Sprintf("Error: Create docker client failed, %s", err.Error()))
	}

	return cli
}

func Container(containerJSON *types.ContainerJSON) map[string]string {
	labels := containerJSON.Config.Labels
	c := make(map[string]string)
	putIfNotEmpty(c, "k8s_pod", labels[runtime.KubernetesPodName])
	putIfNotEmpty(c, "k8s_pod_namespace", labels[runtime.KubernetesContainerNamespace])
	putIfNotEmpty(c, "k8s_container_name", labels[runtime.KubernetesContainerName])
	putIfNotEmpty(c, "k8s_node_name", os.Getenv("NODE_NAME"))
	putIfNotEmpty(c, "docker_container", strings.TrimPrefix(containerJSON.Name, "/"))
	return c
}

func putIfNotEmpty(store map[string]string, key, value string) {
	if key == "" || value == "" {
		return
	}
	store[key] = value
}
