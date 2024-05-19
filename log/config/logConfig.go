package config

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"sort"
	"strings"
	"watchlog/log/nodeInfo"
	"watchlog/pkg/util"
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
	Target       string
	EstimateTime bool
	Stdout       bool

	CustomFields  map[string]string
	CustomConfigs map[string]string
}

const LabelServiceLogsTmpl = "%s."

func GetLogConfigs(logPrefix string, jsonLogPath string, labels map[string]string) ([]*LogConfig, error) {
	var ret []*LogConfig

	var labelNames []string
	//sort keys
	for k := range labels {
		labelNames = append(labelNames, k)
	}
	sort.Strings(labelNames)

	root := nodeInfo.NewLogInfoNode("")

	for _, label := range labelNames {
		if !strings.HasPrefix(label, fmt.Sprintf(LabelServiceLogsTmpl, logPrefix)) {
			continue
		}

		logLabel := strings.TrimPrefix(label, fmt.Sprintf(LabelServiceLogsTmpl, logPrefix))
		key := strings.Split(logLabel, ".")
		if err := root.Insert(key, labels[label]); err != nil {
			log.Errorf("%s", err.Error())
			return nil, err
		}
	}

	for name, node := range root.Children {
		logConfig, err := parseLogConfig(name, node, jsonLogPath)
		if err != nil {
			return nil, err
		}
		ret = append(ret, logConfig)
	}
	return ret, nil
}

func parseLogConfig(name string, info *nodeInfo.LogInfoNode, jsonLogPath string) (*LogConfig, error) {
	path := strings.TrimSpace(info.Value)
	if path == "" {
		return nil, fmt.Errorf("path for %s is empty", name)
	}

	tags := info.Get("tags")
	tagMap, err := parseTags(tags)
	if err != nil {
		return nil, fmt.Errorf("parse tags for %s error: %v", name, err)
	}

	target := info.Get("target")
	// add default index or topic
	if _, ok := tagMap["index"]; !ok {
		if target != "" {
			tagMap["index"] = target
		} else {
			tagMap["index"] = name
		}
	}

	if _, ok := tagMap["topic"]; !ok {
		if target != "" {
			tagMap["topic"] = target
		} else {
			tagMap["topic"] = name
		}
	}

	format := info.Children["format"]
	if format == nil || format.Value == "none" {
		format = nodeInfo.NewLogInfoNode("nonex")
	}

	formatConfig, err := util.Convert(format)
	if err != nil {
		return nil, fmt.Errorf("in log %s: format error: %v", name, err)
	}

	// 特殊处理regex
	if format.Value == "regexp" {
		format.Value = fmt.Sprintf("/%s/", formatConfig["pattern"])
		delete(formatConfig, "pattern")
	}

	cfg := new(LogConfig)
	// 标准输出日志
	if path == "stdout" {
		logFile := filepath.Base(jsonLogPath) + "*"

		cfg = &LogConfig{
			File:         logFile,
			Name:         name,
			HostDir:      filepath.Dir(jsonLogPath),
			Format:       format.Value,
			Tags:         tagMap,
			FormatConfig: map[string]string{"time_format": "%Y-%m-%dT%H:%M:%S.%NZ"},
			Target:       target,
			EstimateTime: false,
			Stdout:       true,
		}
	}

	return cfg, nil
}

func parseTags(tags string) (map[string]string, error) {
	tagMap := make(map[string]string)

	if tags == "" {
		return tagMap, nil
	}

	kvArray := strings.Split(tags, ",")
	for _, kv := range kvArray {
		arr := strings.Split(kv, "=")
		if len(arr) != 2 {
			return nil, fmt.Errorf("%s is not a valid k=v format", kv)
		}
		key := strings.TrimSpace(arr[0])
		value := strings.TrimSpace(arr[1])
		if key == "" || value == "" {
			return nil, fmt.Errorf("%s is not a valid k=v format", kv)
		}
		tagMap[key] = value
	}

	return tagMap, nil
}
