package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/yaml"
	"github.com/zeromicro/go-zero/core/logc"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	logtypes "watchlog/log/config"
)

// FilebeatPointer Filebeat 插件
type FilebeatPointer struct {
	cmd     *exec.Cmd
	Name    string
	Tmpl    *template.Template
	BaseDir string
}

func NewFilebeatPointer(Tmpl *template.Template, BaseDir string) FilebeatPointer {
	return FilebeatPointer{
		Name:    "Filebeat",
		Tmpl:    Tmpl,
		BaseDir: BaseDir,
	}
}

// Start 启动采集器
func (f FilebeatPointer) Start() error {
	if f.cmd != nil {
		pid := f.cmd.Process.Pid
		return fmt.Errorf("Filebeat process is exists, PID: %d", pid)
	}

	f.cmd = exec.Command(FilebeatExecCmd, "-c", FilebeatConfFile)
	f.cmd.Stderr = os.Stderr
	f.cmd.Stdout = os.Stdout
	err := f.cmd.Start()
	if err != nil {
		logc.Errorf(context.Background(), "Filebeat start fail: %s", err)
	}

	go func() {
		logc.Infof(context.Background(), "Starting Filebeat pid: %v", f.cmd.Process.Pid)
		err := f.cmd.Wait()
		if err != nil {
			logc.Errorf(context.Background(), "Filebeat exited: %v", err)
			if exitError, ok := err.(*exec.ExitError); ok {
				processState := exitError.ProcessState
				logc.Errorf(context.Background(), "Filebeat exited pid: %v", processState.Pid())
			}
		}

		// try to restart filebeat
		logc.Debugf(context.Background(), "Filebeat exited and try to restart")
		f.cmd = nil
		err = f.cmd.Start()
		if err != nil {
			return
		}
	}()

	return err
}

// GetRegistryState 获取 filebeat 仓库中容器日志的基本信息
func (f FilebeatPointer) GetRegistryState() (map[string]RegistryState, error) {
	file, err := os.Open(FilebeatRegistry)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
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

// LoadConfigPaths 加载容器config, path
func (f FilebeatPointer) LoadConfigPaths() map[string]string {
	paths := make(map[string]string, 0)
	// 读取 inputs.d 目录下所有配置
	confs, _ := ioutil.ReadDir(FilebeatConfDir)
	for _, conf := range confs {
		// get file name
		container := strings.TrimRight(conf.Name(), ".yml")
		config, err := f.ParseConfig(container)
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

type Config struct {
	Paths []string `config:"paths"`
}

var configOpts = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

// ParseConfig 解析容器信息配置，获取path信息
func (f FilebeatPointer) ParseConfig(container string) (*Config, error) {
	// get config full path, /etc/filebeat/inputs.d/*.yml
	confPath := f.GetConfPath(container)
	c, err := yaml.NewConfigWithFile(confPath, configOpts...)
	if err != nil {
		logc.Errorf(context.Background(), "read %s.yml log config error: %v", container, err)
		return nil, err
	}

	var config Config
	if err := c.Unpack(&config); err != nil {
		logc.Errorf(context.Background(), "parse %s.yml log config error: %v", container, err)
		return nil, err
	}
	return &config, nil
}

// RenderLogConfig 生成日志采集配置文件
func (f FilebeatPointer) RenderLogConfig(containerId string, container map[string]string, configList []logtypes.LogConfig) (string, error) {
	for _, config := range configList {
		logc.Infof(context.Background(), "logs: %s = %v", containerId, config)
	}

	var buf bytes.Buffer
	m := map[string]interface{}{
		"containerId": containerId,
		"configList":  configList,
		"container":   container,
		"output":      "FILEBEAT_OUTPUT",
	}
	if err := f.Tmpl.Execute(&buf, m); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// CleanConfigs 清理旧配置
func (f FilebeatPointer) CleanConfigs() error {
	confDir := f.GetConfHome()
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
