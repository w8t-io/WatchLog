package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// LogConfig log configuration
type LogConfig struct {
	Name         string
	HostDir      string
	ContainerDir string
	Format       string
	FormatConfig map[string]string
	File         string
	Tags         map[string]string
	EstimateTime bool
	Stdout       bool
	LogType      string
}

const LabelServiceLogsTmpl = "%s_"

func GetLogConfigs(logPrefix string, jsonLogPath string, labels map[string]string) ([]LogConfig, error) {
	var ret []LogConfig
	for label, _ := range labels {
		p := fmt.Sprintf(LabelServiceLogsTmpl, logPrefix)
		logTopicName := strings.TrimPrefix(label, p) // watchlog_default, logTopicName = default
		logConfig, err := parseLogConfig(logTopicName, labels[label], jsonLogPath)
		if err != nil {
			return nil, err
		}
		ret = append(ret, logConfig)
	}
	return ret, nil
}

//func getLabelNames(logPrefix string, labels map[string]string) []string {
//	var labelNames []string
//	for k := range labels {
//		if strings.HasPrefix(k, logPrefix) {
//			labelNames = append(labelNames, k)
//		}
//	}
//	//sort keys
//	sort.Strings(labelNames)
//	return labelNames
//}

func parseLogConfig(label, value string, jsonLogPath string) (LogConfig, error) {
	cfg := new(LogConfig)
	if value == "" {
		return *cfg, fmt.Errorf("env %s value don't is null", label)
	}

	// 标准输出日志
	if value == "stdout" {
		logFile := filepath.Base(jsonLogPath) + "*"
		cfg = &LogConfig{
			File:    logFile,
			Name:    label,
			HostDir: filepath.Dir(jsonLogPath),
			Tags: map[string]string{
				"index": label,
				"topic": label,
			},
			LogType: "container",
		}
	}
	return *cfg, nil
}
