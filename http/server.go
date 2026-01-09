package http

import (
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/logger"
	"net/http"
	"time"
)

var ErrServerClosed = http.ErrServerClosed

// ServerConfig HTTP服务器配置（原有逻辑不变）
type ServerConfig struct {
	Addr           string        // 监听地址（ip:port）
	ReadTimeout    time.Duration // 读超时
	WriteTimeout   time.Duration // 写超时
	MaxHeaderBytes int           // 最大请求头大小
	SSL            bool          // 是否启用SSL
	SSLCertFile    string        // SSL证书路径
	SSLKeyFile     string        // SSL密钥路径
}

// Server HTTP服务器（门面角色，负责服务生命周期管理）
type Server struct {
	config *ServerConfig
	router *Router      // 注入的HTTP路由器
	server *http.Server // 系统HTTP服务实例
}

// NewServer 创建HTTP服务器实例
func NewServer(appName string) *Server {
	cfg := loadServerConfig(appName)
	router := NewRouter()
	serv := &Server{
		config: cfg,
		router: router, // 默认初始化路由器，也可通过SetRouter替换
		server: &http.Server{
			Addr:           cfg.Addr,
			ReadTimeout:    cfg.ReadTimeout,
			WriteTimeout:   cfg.WriteTimeout,
			MaxHeaderBytes: cfg.MaxHeaderBytes,
			Handler:        router, // 临时占位，SetRouter会覆盖
		},
	}
	serv.Use(CORS())
	return serv
}

// Config 暴露配置
func (s *Server) Config() *ServerConfig {
	return s.config
}

// Use 注册全局中间件（门面方法，委托给Router）
func (s *Server) Use(middlewares ...MiddlewareFunc) {
	s.router.Use(middlewares...)
}

// Handle 注册通用路由（门面方法，委托给Router）
func (s *Server) Handle(method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	s.router.Handle(method, path, handler, middlewares...)
}

// GET 快捷注册GET路由（门面方法，委托给Router）
func (s *Server) GET(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	s.router.GET(path, handler, middlewares...)
}

// POST 快捷注册POST路由（门面方法，委托给Router）
func (s *Server) POST(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	s.router.POST(path, handler, middlewares...)
}

// PUT 快捷注册PUT路由（门面方法，委托给Router）
func (s *Server) PUT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	s.router.PUT(path, handler, middlewares...)
}

// DELETE 快捷注册DELETE路由（门面方法，委托给Router）
func (s *Server) DELETE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	s.router.DELETE(path, handler, middlewares...)
}

// Run 启动HTTP服务器（原有逻辑不变）
func (s *Server) Run() error {
	logger.Info("HTTP服务器启动成功，监听地址：", s.config.Addr)
	if s.config.SSL {
		return s.server.ListenAndServeTLS(s.config.SSLCertFile, s.config.SSLKeyFile)
	}
	return s.server.ListenAndServe()
}

// Stop 停止HTTP服务器（原有逻辑不变）
func (s *Server) Stop() error {
	logger.Info("HTTP服务器正在停止...")
	return s.server.Shutdown(nil)
}

// loadServerConfig 加载配置（原有逻辑不变）
func loadServerConfig(appName string) *ServerConfig {
	appCfg := config.GetAppConfig(appName)
	httpCfg := appCfg.HTTP
	return &ServerConfig{
		Addr:           httpCfg.Addr,
		ReadTimeout:    time.Duration(httpCfg.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(httpCfg.WriteTimeout) * time.Second,
		MaxHeaderBytes: httpCfg.MaxHeaderBytes,
		SSL:            httpCfg.SSL,
		SSLCertFile:    httpCfg.SSLCertFile,
		SSLKeyFile:     httpCfg.SSLKeyFile,
	}
}
