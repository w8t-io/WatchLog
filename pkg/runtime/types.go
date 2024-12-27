package runtime

import "os"

const (
	KubernetesPodName            = "io.kubernetes.pod.name"
	KubernetesContainerName      = "io.kubernetes.container.name"
	KubernetesContainerNamespace = "io.kubernetes.pod.namespace"
)

func putIfNotEmpty(store map[string]string, key, value string) {
	if key == "" || value == "" {
		return
	}
	store[key] = value
}

func BuildContainerLabels(labels map[string]string) map[string]string {
	c := make(map[string]string)
	putIfNotEmpty(c, "k8s_pod", labels[KubernetesPodName])
	putIfNotEmpty(c, "k8s_pod_namespace", labels[KubernetesContainerNamespace])
	putIfNotEmpty(c, "k8s_container_name", labels[KubernetesContainerName])
	putIfNotEmpty(c, "k8s_node_name", os.Getenv("NODE_NAME"))
	return c
}
