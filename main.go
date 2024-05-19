package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	log2 "watchlog/log"
)

func main() {
	// -template /pilot/filebeat.tpl -base /host -log-level debug
	template := flag.String("template", "", "Template filepath for fluentd or filebeat.")
	base := flag.String("base", "", "Directory which mount host root.")
	level := flag.String("log-level", "INFO", "Log level")
	flag.Parse()

	if os.Getenv("DOCKER_API_VERSION") == "" {
		err := os.Setenv("DOCKER_API_VERSION", "1.23")
		if err != nil {
			log.Errorf(err.Error())
			return
		}
	}

	if os.Getenv("RUNTIME_TYPE") == "" {
		log.Errorf("Please set service type, (docker|containerd)")
	}

	baseDir, err := filepath.Abs(*base)
	if err != nil {
		panic(err)
	}

	if baseDir == "/" {
		baseDir = ""
	}

	if *template == "" {
		panic("template file can not be empty")
	}

	log.SetOutput(os.Stdout)
	logLevel, err := log.ParseLevel(*level)
	if err != nil {
		log.Errorf(err.Error())
		panic(err)
	}
	log.SetLevel(logLevel)

	log.Fatal(log2.Run(*template, baseDir))
}
