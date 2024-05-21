package filebeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/yaml"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
	logtypes "watchlog/log/config"
)

var filebeat *exec.Cmd

var configOpts = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

const (
	EnvLoggingOutput = "LOGGING_OUTPUT"
)

type InterFilebeatPointer interface {
	Name() string

	Start() error
	Reload() error
	Stop() error

	GetBaseConf() string
	// GetConfHome [base]/prospectors.d
	GetConfHome() string
	GetConfPath(container string) string

	OnDestroyEvent(container string) error

	RenderLogConfig(containerId string, container map[string]string, configList []*logtypes.LogConfig) (string, error)
}

// NewFilebeatPointer returns a FilebeatPointer instance
func NewFilebeatPointer(tmpl *template.Template, baseDir string) InterFilebeatPointer {
	return &FilebeatPointer{
		name:           "filebeat",
		Tmpl:           tmpl,
		baseDir:        baseDir,
		watchDone:      make(chan bool),
		watchContainer: make(map[string]string, 0),
		watchDuration:  60 * time.Second,
	}
}

// Start starting and watching filebeat process
func (p *FilebeatPointer) Start() error {
	if filebeat != nil {
		pid := filebeat.Process.Pid
		log.Infof("filebeat started, pid: %v", pid)
		return fmt.Errorf("filebeat process is exists")
	}

	log.Info("starting filebeat")
	filebeat = exec.Command(FilebeatExecCmd, "-c", FilebeatConfFile)
	filebeat.Stderr = os.Stderr
	filebeat.Stdout = os.Stdout
	err := filebeat.Start()
	if err != nil {
		log.Errorf("filebeat start fail: %v", err)
	}

	go func() {
		log.Infof("filebeat started pid: %v", filebeat.Process.Pid)
		err := filebeat.Wait()
		if err != nil {
			log.Errorf("filebeat exited: %v", err)
			if exitError, ok := err.(*exec.ExitError); ok {
				processState := exitError.ProcessState
				log.Errorf("filebeat exited pid: %v", processState.Pid())
			}
		}

		// try to restart filebeat
		log.Warningf("filebeat exited and try to restart")
		filebeat = nil
		err = p.Start()
		if err != nil {
			return
		}
	}()

	go func() {
		err := p.watch()
		if err != nil {
			panic(err)
		}
	}()
	return err
}

// Stop log collection
func (p *FilebeatPointer) Stop() error {
	p.watchDone <- true
	return nil
}

// Reload reload configuration file
func (p *FilebeatPointer) Reload() error {
	log.Debug("do not need to reload filebeat")
	return nil
}

// start filebeat watch process
func (p *FilebeatPointer) watch() error {
	log.Infof("%s watcher start", p.Name())
	for {
		select {
		case <-p.watchDone:
			log.Infof("%s watcher stop", p.Name())
			return nil
		case <-time.After(p.watchDuration):
			err := p.scan()
			if err != nil {
				log.Errorf("%s watcher scan error: %v", p.Name(), err)
			}
		}
	}
}

func (p *FilebeatPointer) scan() error {
	// 获取注册表状态
	states, err := p.getRegistryState()
	if err != nil {
		return fmt.Errorf("failed to get registry state: %v", err)
	}

	// 加载配置路径
	configPaths := p.loadConfigPaths()

	// 处理所有监控的容器
	for container := range p.watchContainer {
		if err := p.processContainer(container, states, configPaths); err != nil {
			log.Errorf("error processing container %s: %v", container, err)
		}
	}

	return nil
}

func (p *FilebeatPointer) processContainer(container string, states map[string]RegistryState, configPaths map[string]string) error {
	confPath := p.GetConfPath(container)
	if _, err := os.Stat(confPath); err != nil {
		if os.IsNotExist(err) {
			log.Warnf("log config %s.yml has been removed and will be ignored", container)
			delete(p.watchContainer, container)
			return nil
		}
		return fmt.Errorf("failed to stat config path %s: %v", confPath, err)
	}

	if p.canRemoveConf(container, states, configPaths) {
		log.Warnf("attempting to remove log config for container %s", container)
		if err := os.Remove(confPath); err != nil {
			return fmt.Errorf("failed to remove log config %s.yml: %v", container, err)
		}
		log.Infof("successfully removed log config for container %s", container)
		delete(p.watchContainer, container)
	}

	return nil
}

func (p *FilebeatPointer) canRemoveConf(container string, registry map[string]RegistryState, configPaths map[string]string) bool {
	config, err := p.loadConfig(container)
	if err != nil {
		return false
	}

	for _, path := range config.Paths {
		pDir := filepath.Dir(path)
		autoMount := p.isAutoMountPath(pDir)
		logFiles, _ := filepath.Glob(path)
		for _, logFile := range logFiles {
			info, err := os.Stat(logFile)
			if err != nil && os.IsNotExist(err) {
				continue
			}
			if _, ok := registry[logFile]; !ok {
				log.Warnf("%s->%s registry not exist", container, logFile)
				continue
			}
			if registry[logFile].V.Offset < info.Size() {
				if autoMount { // ephemeral logs
					log.Infof("%s->%s does not finish to read", container, logFile)
					return false
				} else if _, ok := configPaths[path]; !ok { // host path bind
					log.Infof("%s->%s does not finish to read and not exist in other config",
						container, logFile)
					return false
				}
			}
		}
	}
	return true
}

// 解析容器信息配置，获取path信息
func (p *FilebeatPointer) loadConfig(container string) (*Config, error) {
	// get config full path, /etc/filebeat/inputs.d/*.yml
	confPath := p.GetConfPath(container)
	c, err := yaml.NewConfigWithFile(confPath, configOpts...)
	if err != nil {
		log.Errorf("read %s.yml log config error: %v", container, err)
		return nil, err
	}

	var config Config
	if err := c.Unpack(&config); err != nil {
		log.Errorf("parse %s.yml log config error: %v", container, err)
		return nil, err
	}
	return &config, nil
}

// 加载容器config, path
func (p *FilebeatPointer) loadConfigPaths() map[string]string {
	paths := make(map[string]string, 0)
	// 读取 prospectors.d 目录下所有配置
	confs, _ := ioutil.ReadDir(p.GetConfHome())
	for _, conf := range confs {
		// get file name
		container := strings.TrimRight(conf.Name(), ".yml")
		if _, ok := p.watchContainer[container]; ok {
			continue // ignore removed container
		}

		config, err := p.loadConfig(container)
		if err != nil || config == nil {
			continue
		}

		for _, path := range config.Paths {
			if _, ok := paths[path]; !ok {
				paths[path] = container
			}
		}
	}
	return paths
}

// 判断容器日志路径是否和挂载点匹配
func (p *FilebeatPointer) isAutoMountPath(path string) bool {
	dockerVolumePattern := fmt.Sprintf("^%s.*$", filepath.Join(p.baseDir, DockerSystemPath))
	if ok, _ := regexp.MatchString(dockerVolumePattern, path); ok {
		return true
	}

	kubeletVolumePattern := fmt.Sprintf("^%s.*$", filepath.Join(p.baseDir, KubeletSystemPath))
	ok, _ := regexp.MatchString(kubeletVolumePattern, path)
	return ok
}

// 获取 filebeat 仓库中容器日志的基本信息
func (p *FilebeatPointer) getRegistryState() (map[string]RegistryState, error) {
	f, err := os.Open(FilebeatRegistry)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var state RegistryState
	err = decoder.Decode(&state)
	if err != nil {
		return nil, err
	}

	statesMap := make(map[string]RegistryState, 0)
	if _, ok := statesMap[state.V.Source]; !ok {
		statesMap[state.V.Source] = state
	}

	return statesMap, nil
}

// RenderLogConfig 生成日志采集配置文件
func (p *FilebeatPointer) RenderLogConfig(containerId string, container map[string]string, configList []*logtypes.LogConfig) (string, error) {
	for _, config := range configList {
		log.Infof("logs: %s = %v", containerId, config)
	}

	output := os.Getenv(EnvFilebeatOutput)
	if output == "" {
		output = os.Getenv(EnvLoggingOutput)
	}

	var buf bytes.Buffer
	m := map[string]interface{}{
		"containerId": containerId,
		"configList":  configList,
		"container":   container,
		"output":      output,
	}
	if err := p.Tmpl.Execute(&buf, m); err != nil {
		return "", err
	}
	return buf.String(), nil
}
