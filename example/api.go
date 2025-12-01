package main

import (
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/example/bootstrap"
	"github.com/dfpopp/go-dai/logger"
)

func main() {
	server := bootstrap.Init("api")
	appConfig := config.GetAppConfig("api")

	logger.Info("API服务启动成功，端口：", appConfig.Port)
	_ = server.Run(":" + appConfig.Port)
}
