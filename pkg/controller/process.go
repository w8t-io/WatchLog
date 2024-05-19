package controller

import (
	log "github.com/sirupsen/logrus"
	"os"
	"watchlog/pkg/ctx"
)

// Exists 判断采集容器日志的配置是否存在
func Exists(ctx *ctx.Context, containId string) bool {
	if _, err := os.Stat(ctx.Piloter.GetConfPath(containId)); os.IsNotExist(err) {
		return false
	}
	return true
}

// DelContainerLogFile 销毁采集容器日志文件
func DelContainerLogFile(ctx *ctx.Context, id string) error {
	log.Infof("Try removing log config %s", id)
	if err := os.Remove(ctx.Piloter.GetConfPath(id)); err != nil {
		log.Warnf("removing %s log config failure", id)
		return err
	}

	//ctx.TryReload()
	return ctx.Piloter.OnDestroyEvent(id)
}
