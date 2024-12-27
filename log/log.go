package log

import (
	"context"
	"github.com/docker/docker/api/types/filters"
	"github.com/zeromicro/go-zero/core/logc"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"watchlog/controller"
	"watchlog/pkg/ctx"
	"watchlog/pkg/provider"
)

// Run starts the log pilot.
func Run(tmplPath, logPrefix, baseDir string) error {
	tmpl, err := loadTemplate(tmplPath)
	if err != nil {
		return err
	}

	p := provider.NewFilebeatPointer(tmpl, baseDir)
	c := ctx.NewContext(baseDir, logPrefix, p)
	return startWorker(c)
}

// loadTemplate reads and parses the template file.
func loadTemplate(path string) (*template.Template, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return template.New("watchalert").Parse(string(data))
}

// startWorker initiates the worker process.
func startWorker(c *ctx.Context) error {
	if err := c.FilebeatPointer.CleanConfigs(); err != nil {
		return err
	}

	if err := c.FilebeatPointer.Start(); err != nil {
		return err
	}

	if err := processContainers(c); err != nil {
		return err
	}

	waitForShutdown()
	logc.Infof(context.Background(), "Program Stop Successful!!!")
	return nil
}

// processContainers handles container processing based on the runtime type.
func processContainers(c *ctx.Context) error {
	switch os.Getenv("RUNTIME_TYPE") {
	case "docker":
		logc.Infof(context.Background(), "Processing Docker runtime")
		filter := filters.NewArgs()
		filter.Add("type", "container")
		dockerController := controller.NewDockerInterface(c, filter)
		return dockerController.ProcessContainers()
	case "containerd":
		logc.Infof(context.Background(), "Processing container runtime")
		containerController := controller.NewContainerInterface(c)
		return containerController.ProcessContainers()
	default:
		return nil
	}
}

// waitForShutdown listens for OS signals to gracefully shut down the program.
func waitForShutdown() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
