package grpc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"net"
	"time"
)

var ErrServerClosed = grpc.ErrServerStopped

// ServerConfig gRPC服务器配置
type ServerConfig struct {
	Addr           string        // 监听地址
	Timeout        time.Duration // 请求超时
	MaxRecvMsgSize int           // 最大接收消息大小
	MaxSendMsgSize int           // 最大发送消息大小
	SSL            bool          // 是否启用SSL
	SSLCertFile    string        // SSL证书路径
	SSLKeyFile     string        // SSL密钥路径
}

// Server gRPC服务器（门面角色，对齐HTTP/WS Server）
type Server struct {
	config     *ServerConfig
	router     *Router
	GrpcServer *grpc.Server
	services   map[string]interface{} // 存储注册的gRPC服务
}

// NewServer 创建gRPC服务器实例
func NewServer(appName string) *Server {
	cfg := loadServerConfig(appName)
	setDefaultConfig(cfg)
	router := NewRouter()

	// 构建gRPC服务器选项
	opts := buildServerOptions(cfg)

	// 创建原生gRPC服务器
	grpcServer := grpc.NewServer(opts...)

	// 新增：注册反射服务（核心！启用后测试工具可自动获取接口定义）
	reflection.Register(grpcServer)

	return &Server{
		config:     cfg,
		router:     router,
		GrpcServer: grpcServer,
		services:   make(map[string]interface{}),
	}
}

// Config 暴露配置
func (s *Server) Config() *ServerConfig {
	return s.config
}

// Use 注册全局中间件
func (s *Server) Use(middlewares ...MiddlewareFunc) {
	s.router.Use(middlewares...)
}

// Register 注册gRPC路由（对齐HTTP Handle/WS Register）
func (s *Server) Register(method string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	chain := append(s.router.middlewares, middlewares...)
	s.router.Register(method, handler, chain)
}

// RegisterService 注册gRPC服务（兼容标准gRPC注册逻辑，保证应用层正常使用）
func (s *Server) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {
	// 1. 标准gRPC服务注册
	s.GrpcServer.RegisterService(sd, ss)
	// 2. 存储服务实例，供框架内部使用
	s.services[sd.ServiceName] = ss
	logger.Info("gRPC服务注册成功：", sd.ServiceName)
}

// Run 启动gRPC服务器
func (s *Server) Run() error {
	lis, err := s.createListener()
	if err != nil {
		return fmt.Errorf("create gRPC listener failed: %w", err)
	}
	defer lis.Close()

	logger.Info("gRPC服务器启动成功，监听地址：", s.config.Addr)
	return s.GrpcServer.Serve(lis)
}

// Stop 停止gRPC服务器
func (s *Server) Stop() {
	logger.Info("gRPC服务器正在停止...")
	s.GrpcServer.GracefulStop()
	logger.Info("gRPC服务器已停止")
}

// 内部方法：创建监听器
func (s *Server) createListener() (net.Listener, error) {
	if s.config.SSL {
		if s.config.SSLCertFile == "" || s.config.SSLKeyFile == "" {
			return nil, fmt.Errorf("SSL enabled but cert/key file is empty")
		}
		// 加载证书
		cert, err := tls.LoadX509KeyPair(s.config.SSLCertFile, s.config.SSLKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load SSL cert failed: %w", err)
		}
		// 配置TLS
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		return tls.Listen("tcp", s.config.Addr, tlsConfig)
	}
	// 非SSL模式：普通TCP监听
	return net.Listen("tcp", s.config.Addr)
}

// 内部方法：构建gRPC服务器选项
func buildServerOptions(cfg *ServerConfig) []grpc.ServerOption {
	var opts []grpc.ServerOption

	// 设置消息大小限制
	if cfg.MaxRecvMsgSize > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize))
	}
	if cfg.MaxSendMsgSize > 0 {
		opts = append(opts, grpc.MaxSendMsgSize(cfg.MaxSendMsgSize))
	}

	// SSL配置：创建credentials并传入gRPC选项（修复creds未使用问题）
	if cfg.SSL {
		if cfg.SSLCertFile == "" || cfg.SSLKeyFile == "" {
			logger.Warn("SSL enabled but cert/key file is empty, skip SSL config")
			return opts
		}
		// 加载证书并创建gRPC credentials
		cert, err := tls.LoadX509KeyPair(cfg.SSLCertFile, cfg.SSLKeyFile)
		if err != nil {
			logger.Error("load SSL cert failed when build server options: ", err)
			return opts
		}
		creds := credentials.NewServerTLSFromCert(&cert)
		opts = append(opts, grpc.Creds(creds)) // 正确使用creds，传入gRPC服务器选项
	}

	// 注册通用拦截器（适配框架上下文）
	opts = append(opts, grpc.UnaryInterceptor(unaryInterceptor))

	return opts
}

// 通用gRPC拦截器（转换为框架上下文）
func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 1. 获取元数据和客户端信息
	md, _ := metadata.FromIncomingContext(ctx)
	peerInfo, _ := peer.FromContext(ctx)

	// 2. 序列化请求数据（作为原始数据）
	rawData, _ := json.Marshal(req)

	// 3. 创建框架gRPC上下文
	grpcCtx := NewContext(md, peerInfo, info.FullMethod, rawData)

	// 4. 路由分发（执行中间件和处理器）
	server := extractServerFromContext(ctx) // 实际项目中可通过上下文传递Server实例
	if server != nil {
		_ = server.router.Dispatch(grpcCtx)
	}

	// 5. 执行原始gRPC处理器
	resp, err := handler(ctx, req)
	if err != nil {
		logger.Error("gRPC handler error: ", err)
		return resp, err
	}

	// 6. 合并框架响应数据
	return mergeResponse(resp, grpcCtx.GetResponse()), nil
}

// 内部方法：提取Server实例（简化实现，实际可通过自定义上下文传递）
func extractServerFromContext(ctx context.Context) *Server {
	// 此处为简化实现，实际项目中可通过grpc.NewContextWithValue传递Server实例
	return nil
}

// 内部方法：合并响应数据
func mergeResponse(originResp interface{}, frameResp map[string]interface{}) interface{} {
	// 兼容原有响应和框架响应，保持一致性
	if frameResp == nil || len(frameResp) == 0 {
		return originResp
	}
	// 实际项目中可根据业务需求合并响应字段
	return originResp
}

// 内部方法：加载配置
func loadServerConfig(appName string) *ServerConfig {
	appCfg := config.GetAppConfig(appName)
	grpcCfg := appCfg.GRPC
	return &ServerConfig{
		Addr:           grpcCfg.Addr,
		Timeout:        time.Duration(grpcCfg.Timeout) * time.Second,
		MaxRecvMsgSize: grpcCfg.MaxRecvMsgSize,
		MaxSendMsgSize: grpcCfg.MaxSendMsgSize,
		SSL:            grpcCfg.SSL,
		SSLCertFile:    grpcCfg.SSLCertFile,
		SSLKeyFile:     grpcCfg.SSLKeyFile,
	}
}

// 内部方法：设置默认配置
func setDefaultConfig(cfg *ServerConfig) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRecvMsgSize == 0 {
		cfg.MaxRecvMsgSize = 1024 * 1024 * 4 // 4MB
	}
	if cfg.MaxSendMsgSize == 0 {
		cfg.MaxSendMsgSize = 1024 * 1024 * 4 // 4MB
	}
	if cfg.Addr == "" {
		cfg.Addr = ":50051"
	}
}
