package main

import (
	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/src/internal/commandcatalog"
)

var (
	// allowList 保存当前进程允许执行的车辆命令；只有 ENABLE_COMMANDS=true 时才会初始化。
	allowList []string
)

func initCommandAllowList() {
	allowList = commandcatalog.LoadAllowedCommands(getEnvAsBool, getEnv, gin.IsDebugging())
}
