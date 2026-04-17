package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"

	"github.com/dfpopp/go-dai/base"
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/db"
	"github.com/dfpopp/go-dai/grpc"
	"github.com/dfpopp/go-dai/http"
	"github.com/dfpopp/go-dai/logger"
	"github.com/dfpopp/go-dai/websocket"
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
	CustomConfigPath   string          // 自定义配置文件路径（可选）
	EnableServices     []ServiceType   // 需要启动的服务类型
	Router             base.BaseRouter // 应用路由实例
	ShutdownFunc       func()          // 服务关闭时的回调函数（可选）
}

// BootContext 启动上下文（存储已启动的服务）
type BootContext struct {
	HTTPServer *http.Server
	WSServer   *websocket.Server
	GRPCServer *grpc.Server
}

// Boot 统一服务启动入口
func Boot(cfg *BootConfig) (*BootContext, error) {
	appPath := ""
	_, entryFile, _, ok := runtime.Caller(1)
	if ok {
		appPath = filepath.Dir(entryFile)
	}
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
	if cfg.CustomConfigPath != "" {
		if err := config.LoadCustomConfig(cfg.CustomConfigPath); err != nil {
			return nil, fmt.Errorf("加载自定义配置失败: %v", err)
		}
	}
	// 3. 初始化日志
	if err := logger.InitLogger(cfg.AppName, appPath); err != nil {
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

	// 错误通道，用于传递服务启动错误
	errChan := make(chan error, len(cfg.EnableServices))
	// 服务启动完成通道
	doneChan := make(chan struct{})

	for _, serviceType := range cfg.EnableServices {
		wg.Add(1)
		switch serviceType {
		case ServiceTypeHTTP:
			// 初始化HTTP服务
			bootCtx.HTTPServer = http.NewServer(cfg.AppName)
			// 注册路由
			cfg.Router.RegisterHTTPRoutes(bootCtx.HTTPServer)
			// 异步启动
			go func() {
				defer wg.Done()
				if err := bootCtx.HTTPServer.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error(fmt.Errorf("HTTP服务启动失败: %v", err))
					errChan <- fmt.Errorf("HTTP服务启动失败: %v", err)
				}
			}()
			logger.Info("HTTP服务已初始化，监听地址：", bootCtx.HTTPServer.Config().Addr)
		case ServiceTypeWS:
			// 初始化WebSocket服务
			bootCtx.WSServer = websocket.NewServer(cfg.AppName)
			// 注册路由
			cfg.Router.RegisterWSRoutes(bootCtx.WSServer)
			// 异步启动
			go func() {
				defer wg.Done()
				if err := bootCtx.WSServer.Run(); err != nil && !errors.Is(err, websocket.ErrServerClosed) {
					err := fmt.Errorf("WebSocket服务启动失败: %v", err)
					logger.Error(fmt.Errorf("WebSocket服务启动失败: %v", err))
					errChan <- fmt.Errorf("WebSocket服务启动失败: %v", err)
				}
			}()
			logger.Info("WebSocket服务已初始化，监听地址：", bootCtx.WSServer.Config().Addr)
		case ServiceTypeGRPC:
			// 初始化gRPC服务
			bootCtx.GRPCServer = grpc.NewServer(cfg.AppName)
			// 注册路由
			cfg.Router.RegisterGRPCRoutes(bootCtx.GRPCServer)
			// 异步启动
			go func() {
				defer wg.Done()
				if err := bootCtx.GRPCServer.Run(); err != nil {
					logger.Error(fmt.Errorf("gRPC服务启动失败: %v", err))
					errChan <- fmt.Errorf("gRPC服务启动失败: %v", err)
				}
			}()
			logger.Info("gRPC服务已初始化，监听地址：", bootCtx.GRPCServer.Config().Addr)
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
		// 1. 执行业务层清理逻辑
		if cfg.ShutdownFunc != nil {
			logger.Info("执行业务层清理逻辑...")
			cfg.ShutdownFunc()
		}
		// 2. 关闭数据库连接
		if len(startDb) > 0 {
			logger.Info("关闭数据库连接...")
			db.CloseDb(startDb)
		}
		// 3. 停止HTTP服务
		if bootCtx.HTTPServer != nil {
			_ = bootCtx.HTTPServer.Stop()
		}
		// 4. 停止WebSocket服务
		if bootCtx.WSServer != nil {
			_ = bootCtx.WSServer.Stop()
		}
		// 5. 停止gRPC服务
		if bootCtx.GRPCServer != nil {
			bootCtx.GRPCServer.Stop()
		}
		logger.Info("应用已完成停机")
	}()
	// 等待所有服务启动完成
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	// 等待服务启动完成或错误发生
	select {
	case <-doneChan:
		// 所有服务启动成功
		return bootCtx, nil
	case err := <-errChan:
		// 服务启动失败，停止所有已启动的服务
		if bootCtx.HTTPServer != nil {
			_ = bootCtx.HTTPServer.Stop()
		}
		if bootCtx.WSServer != nil {
			_ = bootCtx.WSServer.Stop()
		}
		if bootCtx.GRPCServer != nil {
			bootCtx.GRPCServer.Stop()
		}
		return nil, err
	}
}

// BootCron 定时任务启动入口
func BootCron(cfg *BootConfig) error {
	appPath := ""
	_, entryFile, _, ok := runtime.Caller(1)
	if ok {
		appPath = filepath.Dir(entryFile)
	}
	// 1. 校验配置
	if cfg.AppName == "" || cfg.AppConfigPath == "" || cfg.DatabaseConfigPath == "" {
		return fmt.Errorf("启动配置不完整，请检查必填参数")
	}

	// 2. 加载配置
	if err := config.LoadAppConfig(cfg.AppConfigPath, cfg.AppName); err != nil {
		return fmt.Errorf("加载应用配置失败: %v", err)
	}
	if err := config.LoadDatabaseConfig(cfg.DatabaseConfigPath); err != nil {
		return fmt.Errorf("加载数据库配置失败: %v", err)
	}
	if cfg.CustomConfigPath != "" {
		if err := config.LoadCustomConfig(cfg.CustomConfigPath); err != nil {
			return fmt.Errorf("加载自定义配置失败: %v", err)
		}
	}
	// 3. 初始化日志
	if err := logger.InitLogger(cfg.AppName, appPath); err != nil {
		return fmt.Errorf("初始化日志失败: %v", err)
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
	// 5. 优雅停机监听
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("应用开始优雅停机...")
		// 1. 执行业务层清理逻辑
		if cfg.ShutdownFunc != nil {
			logger.Info("执行业务层清理逻辑...")
			cfg.ShutdownFunc()
		}
		// 2. 关闭数据库连接
		if len(startDb) > 0 {
			logger.Info("关闭数据库连接...")
			db.CloseDb(startDb)
		}
		logger.Info("应用已完成停机")
	}()
	return nil
}
