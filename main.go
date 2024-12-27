package main

import (
	"context"
	"flag"
	"github.com/zeromicro/go-zero/core/logc"
	"os"
	"watchlog/log"
)

func main() {
	// Command-line flags
	template := flag.String("template", "", "Template filepath for fluentd or filebeat.")
	flag.Parse()

	// Set default Docker API version if not set
	if err := setDefaultDockerAPIVersion(); err != nil {
		logc.Errorf(context.Background(), err.Error())
		return
	}

	// Validate runtime type
	if os.Getenv("RUNTIME_TYPE") == "" {
		panic("Please set service type, (docker|containerd)")
	}

	// Validate template
	if *template == "" {
		panic("template file cannot be empty")
	}

	// Run log processing
	if err := log.Run(*template, getLogPrefix(), getBaseDir()); err != nil {
		logc.Errorf(context.Background(), err.Error())
	}
}

// setDefaultDockerAPIVersion sets the default Docker API version if not already set.
func setDefaultDockerAPIVersion() error {
	if os.Getenv("DOCKER_API_VERSION") == "" {
		return os.Setenv("DOCKER_API_VERSION", "1.24")
	}
	return nil
}

// getLogPrefix retrieves the log prefix from the environment or defaults to "watchlog".
func getLogPrefix() string {
	if lp := os.Getenv("LOG_PREFIX"); len(lp) > 0 {
		return lp
	}
	return "watchlog"
}

// getBaseDir get log base store dir or defaults to "/host/var/log/pods"
func getBaseDir() string {
	if lbd := os.Getenv("LOG_BASE_DIR"); len(lbd) > 0 {
		return lbd
	}
	return "/host/var/log/pods"
}
