package bootstrap

import (
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/base"
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/db"
	"github.com/dfpopp/go-dai/grpc"
	"github.com/dfpopp/go-dai/http"
	"github.com/dfpopp/go-dai/logger"
	"github.com/dfpopp/go-dai/websocket"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// ServiceType 服务类型枚举
type ServiceType string

const (
	ServiceTypeHTTP ServiceType = "http"
	ServiceTypeWS   ServiceType = "ws"
	ServiceTypeGRPC ServiceType = "grpc"
)

// BootConfig 统一启动配置结构体
type BootConfig struct {
	AppName            string          // 应用名（如api/admin）
	AppConfigPath      string          // 应用配置文件路径
	DatabaseConfigPath string          // 数据库配置文件路径
	CustomConfigPaths  []string        // 自定义配置文件路径（可选）
	EnableServices     []ServiceType   // 需要启动的服务类型
	Router             base.BaseRouter // 应用路由实例
}

// BootContext 启动上下文（存储已启动的服务）
type BootContext struct {
	HTTPServer *http.Server
	WSServer   *websocket.Server
	GRPCServer *grpc.Server
}

// Boot 统一服务启动入口
func Boot(cfg *BootConfig) (*BootContext, error) {
	// 1. 校验配置
	if cfg.AppName == "" || cfg.AppConfigPath == "" || cfg.DatabaseConfigPath == "" || len(cfg.EnableServices) == 0 || cfg.Router == nil {
		return nil, fmt.Errorf("启动配置不完整，请检查必填参数")
	}

	// 2. 加载配置
	if err := config.LoadAppConfig(cfg.AppConfigPath, cfg.AppName); err != nil {
		return nil, fmt.Errorf("加载应用配置失败: %v", err)
	}
	if err := config.LoadDatabaseConfig(cfg.DatabaseConfigPath); err != nil {
		return nil, fmt.Errorf("加载数据库配置失败: %v", err)
	}
	// 3. 初始化日志
	if err := logger.InitLogger(cfg.AppName); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %v", err)
	}

	// 4. 初始化数据库
	startDb := make([]string, 0)
	if len(config.DbConfig.MySQL) > 0 {
		startDb = append(startDb, "mysql")
	}
	if len(config.DbConfig.Mongodb) > 0 {
		startDb = append(startDb, "mongodb")
	}
	if len(config.DbConfig.Redis) > 0 {
		startDb = append(startDb, "redis")
	}
	if len(config.DbConfig.Es) > 0 {
		startDb = append(startDb, "es")
	}
	if len(startDb) > 0 {
		db.StartDb(startDb)
	}
	// 5. 初始化并启动服务
	bootCtx := &BootContext{}
	var wg sync.WaitGroup

	for _, serviceType := range cfg.EnableServices {
		wg.Add(1)
		switch serviceType {
		case ServiceTypeHTTP:
			// 初始化HTTP服务
			bootCtx.HTTPServer = http.NewServer(cfg.AppName)
			//bootCtx.HTTPServer.Use(http.CORS(), http.Recovery())
			// 注册路由
			cfg.Router.RegisterHTTPRoutes(bootCtx.HTTPServer)
			// 异步启动
			go func() {
				defer wg.Done()
				if err := bootCtx.HTTPServer.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("HTTP服务启动失败: %v", err)
				}
			}()
			logger.Info("HTTP服务已初始化，监听地址：", bootCtx.HTTPServer.Config().Addr)
			break
		case ServiceTypeWS:
			// 初始化WebSocket服务
			bootCtx.WSServer = websocket.NewServer(cfg.AppName)
			// 注册路由
			cfg.Router.RegisterWSRoutes(bootCtx.WSServer)
			// 异步启动
			go func() {
				defer wg.Done()
				if err := bootCtx.WSServer.Run(); err != nil && !errors.Is(err, websocket.ErrServerClosed) {
					logger.Error("WebSocket服务启动失败: %v", err)
				}
			}()
			logger.Info("WebSocket服务已初始化，监听地址：", bootCtx.WSServer.Config().Addr)
			break
		case ServiceTypeGRPC:
			// 初始化gRPC服务
			bootCtx.GRPCServer = grpc.NewServer(cfg.AppName)
			// 注册路由
			cfg.Router.RegisterGRPCRoutes(bootCtx.GRPCServer)
			// 异步启动
			go func() {
				defer wg.Done()
				if err := bootCtx.GRPCServer.Run(); err != nil {
					logger.Error("gRPC服务启动失败: %v", err)
				}
			}()
			logger.Info("gRPC服务已初始化，监听地址：", bootCtx.GRPCServer.Config().Addr)
			break
		default:
			return nil, fmt.Errorf("未知服务类型: %s", serviceType)
		}
	}

	// 6. 优雅停机监听
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		logger.Info("应用开始优雅停机...")
		// 停止HTTP服务
		if bootCtx.HTTPServer != nil {
			_ = bootCtx.HTTPServer.Stop()
		}
		// 停止WebSocket服务
		if bootCtx.WSServer != nil {
			_ = bootCtx.WSServer.Stop()
		}
		// 停止gRPC服务
		//if bootCtx.GRPCServer != nil {
		//	bootCtx.GRPCServer.Stop()
		//}
		logger.Info("应用已完成停机")
	}()

	// 等待所有服务启动完成
	wg.Wait()
	return bootCtx, nil
}
