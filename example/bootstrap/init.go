package bootstrap

import (
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/db/mongoDb"
	"github.com/dfpopp/go-dai/db/mysql"
	"github.com/dfpopp/go-dai/db/redis"
	apiRouter "github.com/dfpopp/go-dai/example/app/api/router"
	"github.com/dfpopp/go-dai/http"
	"github.com/dfpopp/go-dai/logger"
)

// Init 初始化框架
func Init(appName string) *http.Server {
	// 1. 加载配置
	config.LoadAppConfig("config/app.json")
	config.LoadDatabaseConfig("config/database.json")

	// 2. 初始化日志
	logger.InitLogger()

	// 3. 初始化数据库
	mysql.InitMySQL()
	mongoDb.InitMongoDB()
	redis.InitRedis()

	// 4. 创建HTTP服务
	server := http.NewServer(appName)
	server.Use(http.Recovery()) // 全局异常恢复

	// 5. 加载应用路由
	switch appName {
	case "api":
		apiRouter.Init(server)
	default:
		panic("未知应用名: " + appName)
	}

	return server
}
