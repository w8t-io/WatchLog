package container

import (
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"os"
	"watchlog/pkg/runtime"
)

const sock = "/run/containerd/containerd.sock"

func NewClient() *containerd.Client {
	cli, err := containerd.New(sock)
	if err != nil {
		panic(fmt.Sprintf("Error: Create container client failed, %s", err.Error()))
	}

	return cli
}

func Container(meta containers.Container) map[string]string {
	labels := meta.Labels
	c := make(map[string]string)
	putIfNotEmpty(c, "k8s_pod", labels[runtime.KubernetesPodName])
	putIfNotEmpty(c, "k8s_pod_namespace", labels[runtime.KubernetesContainerNamespace])
	putIfNotEmpty(c, "k8s_container_name", labels[runtime.KubernetesContainerName])
	putIfNotEmpty(c, "k8s_node_name", os.Getenv("NODE_NAME"))
	return c
}

func putIfNotEmpty(store map[string]string, key, value string) {
	if key == "" || value == "" {
		return
	}
	store[key] = value
}
