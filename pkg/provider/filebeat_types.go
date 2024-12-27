package provider

import (
	"fmt"
	"time"
)

// RegistryState represents log offsets
type RegistryState struct {
	K string    `json:"k"`
	V RegistryV `json:"v"`
}

type RegistryV struct {
	Source      string        `json:"source"`
	Offset      int64         `json:"offset"`
	Timestamp   []time.Time   `json:"timestamp"`
	TTL         time.Duration `json:"ttl"`
	Type        string        `json:"type"`
	FileStateOS FileInode
}

type FileInode struct {
	Inode  uint64 `json:"inode,"`
	Device uint64 `json:"device,"`
}

const (
	FilebeatBaseConf = "/usr/share/filebeat"
	FilebeatExecCmd  = FilebeatBaseConf + "/filebeat"
	FilebeatConfFile = FilebeatBaseConf + "/filebeat.yml"
	FilebeatConfDir  = FilebeatBaseConf + "/inputs.d"
	FilebeatRegistry = FilebeatBaseConf + "/data/registry/filebeat/log.json"
)

// GetConfPath get configuration path FilebeatConfDir/${container}.yaml
func (f FilebeatPointer) GetConfPath(container string) string {
	return fmt.Sprintf("%s/%s.yml", FilebeatConfDir, container)
}

// GetBaseConf returns plugin root directory
func (f FilebeatPointer) GetBaseConf() string {
	return FilebeatBaseConf
}

// GetConfHome returns configuration directory
func (f FilebeatPointer) GetConfHome() string {
	return FilebeatConfDir
}
