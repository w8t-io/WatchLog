package controller

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logc"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	logtypes "watchlog/log/config"
	"watchlog/pkg/ctx"
	"watchlog/pkg/runtime"
)

type InterRuntime interface {
	ProcessContainers() error
}

// Collect 判断是否需要收集日志
func Collect(env []string, logPrefix string) bool {
	var exist bool
	for _, e := range env {
		if strings.HasPrefix(e, logPrefix) {
			exist = true
		}
	}

	return exist
}

// Exists 判断采集容器日志的配置是否存在
func Exists(ctx *ctx.Context, containId string) bool {
	if _, err := os.Stat(ctx.FilebeatPointer.GetConfPath(containId)); os.IsNotExist(err) {
		return false
	}
	return true
}

// DelContainerLogFile 销毁采集容器日志文件
func DelContainerLogFile(ctx *ctx.Context, id string) error {
	logc.Infof(context.Background(), "Try removing log config %s", id)
	if err := os.Remove(ctx.FilebeatPointer.GetConfPath(id)); err != nil {
		return fmt.Errorf("removing %s log config failure, err: %s", id, err.Error())
	}

	return nil
}

type CollectFields struct {
	Id      string
	Env     []string
	Labels  map[string]string
	LogPath string
}

// NewCollectFile 创建Filebeat采集配置
func NewCollectFile(ctx *ctx.Context, cf CollectFields) error {
	id := cf.Id
	env := cf.Env
	labels := cf.Labels
	jsonLogPath := cf.LogPath
	ct := runtime.BuildContainerLabels(labels)
	logEnvs := getLogEnvs(env)

	logPath := filepath.Join(ctx.BaseDir, jsonLogPath) // /host/var/lib/containerd/log/pods/intl_diagon-alley-5cf4c7cddc-7nd94_*/diagon-alley/*.log
	logConfigs, err := logtypes.GetLogConfigs(ctx.LogPrefix, logPath, logEnvs)
	if err != nil {
		return fmt.Errorf("GetLogConfigs failed, err: %s", err.Error())
	}

	if len(logConfigs) == 0 {
		return nil
	}

	//生成 filebeat 采集配置
	logConfig, err := ctx.FilebeatPointer.RenderLogConfig(id, ct, logConfigs)
	if err != nil {
		return fmt.Errorf("RenderLogConfig failed, err: %s", err.Error())
	}

	//TODO validate config before save
	logc.Infof(context.Background(), fmt.Sprintf("Write Log config, path: %s", ctx.FilebeatPointer.GetConfPath(id)))
	if err = ioutil.WriteFile(ctx.FilebeatPointer.GetConfPath(id), []byte(logConfig), os.FileMode(0644)); err != nil {
		return fmt.Errorf("WriteFile failed, err: %s", err.Error())
	}

	return nil
}

// getLogEnvs 获取关键 Envs
func getLogEnvs(env []string) map[string]string {
	var logEnv = map[string]string{} //  map[aliyun_logs_tencent-prod-diagon-alley:stdout]
	for _, e := range env {
		envLabel := strings.SplitN(e, "=", 2) // [aliyun_logs_tencent-prod-diagon-alley stdout] 2
		if len(envLabel) == 2 {
			logEnv[envLabel[0]] = envLabel[1] // aliyun_logs_tencent-prod-diagon-alley stdout
		}
	}
	return logEnv
}
