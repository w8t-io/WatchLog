package tools

import (
	"fmt"
	"watchlog/log/nodeInfo"
)

// FormatConverter converts node info to map
type FormatConverter func(info *nodeInfo.LogInfoNode) (map[string]string, error)

var converters = make(map[string]FormatConverter)

// Register format converter instance
func Register(format string, converter FormatConverter) {
	converters[format] = converter
}

// Convert convert node info to map
func Convert(info *nodeInfo.LogInfoNode) (map[string]string, error) {
	converter := converters[info.Value]
	if converter == nil {
		return nil, fmt.Errorf("unsupported log format: %s", info.Value)
	}
	return converter(info)
}

// SimpleConverter simple format converter
type SimpleConverter struct {
	properties map[string]bool
}

func init() {
	simpleConverter := func(properties []string) FormatConverter {
		return func(info *nodeInfo.LogInfoNode) (map[string]string, error) {
			validProperties := make(map[string]bool)
			for _, property := range properties {
				validProperties[property] = true
			}
			ret := make(map[string]string)
			for k, v := range info.Children {
				if _, ok := validProperties[k]; !ok {
					return nil, fmt.Errorf("%s is not a valid properties for format %s", k, info.Value)
				}
				ret[k] = v.Value
			}
			return ret, nil
		}
	}

	Register("nonex", simpleConverter([]string{}))
	Register("csv", simpleConverter([]string{"time_key", "time_format", "keys"}))
	Register("json", simpleConverter([]string{"time_key", "time_format"}))
	Register("regexp", simpleConverter([]string{"time_key", "time_format"}))
	Register("apache2", simpleConverter([]string{}))
	Register("apache_error", simpleConverter([]string{}))
	Register("nginx", simpleConverter([]string{}))
	Register("regexp", func(info *nodeInfo.LogInfoNode) (map[string]string, error) {
		ret, err := simpleConverter([]string{"pattern", "time_format"})(info)
		if err != nil {
			return ret, err
		}
		if ret["pattern"] == "" {
			return nil, fmt.Errorf("regex pattern can not be empty")
		}
		return ret, nil
	})
}
