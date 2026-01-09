package base

import (
	"github.com/dfpopp/go-dai/grpc"
	"github.com/dfpopp/go-dai/http"
	"github.com/dfpopp/go-dai/websocket"
)

// BaseRouter 框架根路由基类
type BaseRouter interface {
	RegisterHTTPRoutes(server *http.Server)    // 注册HTTP路由
	RegisterWSRoutes(server *websocket.Server) // 注册WebSocket路由
	RegisterGRPCRoutes(server *grpc.Server)    // 注册gRPC路由
}

// DefaultBaseRouter 默认路由实现（应用层可嵌入复用）
type DefaultBaseRouter struct{}

// RegisterHTTPRoutes 默认HTTP路由注册（空实现，应用层重写）
func (r *DefaultBaseRouter) RegisterHTTPRoutes(server *http.Server) {}

// RegisterWSRoutes 默认WebSocket路由注册（空实现，应用层重写）
func (r *DefaultBaseRouter) RegisterWSRoutes(server *websocket.Server) {}

// RegisterGRPCRoutes 默认gRPC路由注册（空实现，应用层重写）
func (r *DefaultBaseRouter) RegisterGRPCRoutes(server *grpc.Server) {}
