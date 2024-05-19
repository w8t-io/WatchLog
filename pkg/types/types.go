package types

import (
	"context"
	"fmt"
	"github.com/containerd/containerd"
	docker "github.com/docker/docker/client"
	"os"
	"path/filepath"
	"sync"
	"time"
	"watchlog/pkg/client/filebeat"
)

// Pilot entry point
type Pilot struct {
	Context       context.Context
	Piloter       filebeat.InterFilebeatPointer
	Mutex         sync.Mutex
	Client        *docker.Client
	ContainerCli  *containerd.Client
	LastReload    time.Time
	ReloadChan    chan bool
	StopChan      chan bool
	BaseDir       string
	LogPrefix     string
	CreateSymlink bool
}

//func (p *Pilot) DoReload() {
//	log.Info("Reload goroutine is ready")
//	for {
//		<-p.ReloadChan
//		err := p.Reload()
//		if err != nil {
//			log.Errorf(err.Error())
//			return
//		}
//	}
//}

//func (p *Pilot) Reload() error {
//	p.Mutex.Lock()
//	defer p.Mutex.Unlock()
//
//	log.Infof("Reload %s", p.Piloter.Name())
//	interval := time.Now().Sub(p.LastReload)
//	time.Sleep(30*time.Second - interval)
//
//	log.Info("Start reloading")
//	err := p.Piloter.Reload()
//	p.LastReload = time.Now()
//	return err
//}

//func (p *Pilot) TryReload() {
//	select {
//	case p.ReloadChan <- true:
//	default:
//		log.Info("Another load is pending")
//	}
//}

func (p *Pilot) CleanConfigs() error {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	confDir := fmt.Sprintf(p.Piloter.GetConfHome())
	d, err := os.Open(confDir)
	if err != nil {
		return err
	}
	defer d.Close()

	// 获取目录下所有数据, 包括目录和文件
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		conf := filepath.Join(confDir, name)
		stat, err := os.Stat(filepath.Join(confDir, name))
		if err != nil {
			return err
		}
		// 是否为普通文件
		if stat.Mode().IsRegular() {
			if err := os.Remove(conf); err != nil {
				return err
			}
		}
	}
	return nil
}
