package filebeat

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"text/template"
	"time"
)

// Global variables for Filebeat
const (
	FilebeatBaseConf = "/usr/share/filebeat"
	FilebeatExecCmd  = FilebeatBaseConf + "/filebeat"
	FilebeatRegistry = FilebeatBaseConf + "/data/registry/filebeat/log.json"
	FilebeatConfDir  = FilebeatBaseConf + "/inputs.d"
	FilebeatConfFile = FilebeatBaseConf + "/filebeat.yml"

	DockerSystemPath  = "/var/lib/docker/"
	KubeletSystemPath = "/var/lib/kubelet/"

	EnvFilebeatOutput = "FILEBEAT_OUTPUT"
)

// FilebeatPointer for filebeat plugin
type FilebeatPointer struct {
	name           string
	Tmpl           *template.Template
	baseDir        string
	watchDone      chan bool
	watchDuration  time.Duration
	watchContainer map[string]string
}

// Config contains all log paths
type Config struct {
	Paths []string `config:"paths"`
}

// FileInode identify a unique log file
type FileInode struct {
	Inode  uint64 `json:"inode,"`
	Device uint64 `json:"device,"`
}

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

// GetConfPath returns log configuration path
func (p *FilebeatPointer) GetConfPath(container string) string {
	return fmt.Sprintf("%s/%s.yml", FilebeatConfDir, container)
}

// GetConfHome returns configuration directory
func (p *FilebeatPointer) GetConfHome() string {
	return FilebeatConfDir
}

// Name returns plugin name
func (p *FilebeatPointer) Name() string {
	return p.name
}

// OnDestroyEvent watching destroy event
func (p *FilebeatPointer) OnDestroyEvent(container string) error {
	return p.feed(container)
}

// GetBaseConf returns plugin root directory
func (p *FilebeatPointer) GetBaseConf() string {
	return FilebeatBaseConf
}

func (p *FilebeatPointer) feed(containerID string) error {
	if _, ok := p.watchContainer[containerID]; !ok {
		p.watchContainer[containerID] = containerID
		log.Infof("begin to watch log config: %s.yml", containerID)
	}
	return nil
}
