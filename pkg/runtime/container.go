package runtime

import (
	"fmt"
	"github.com/containerd/containerd"
)

const sock = "/run/containerd/containerd.sock"

func NewContainerClient() *containerd.Client {
	cli, err := containerd.New(sock)
	if err != nil {
		panic(fmt.Sprintf("Error: Create container client failed, %s", err.Error()))
	}

	return cli
}
