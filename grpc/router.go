package grpc

import (
	"errors"
)

// Router gRPC路由器（框架内置）
type Router struct {
	handlers    map[string]HandlerFunc // 服务方法名 -> 处理器
	middlewares []MiddlewareFunc       // 全局中间件
}

// NewRouter 创建gRPC路由器实例
func NewRouter() *Router {
	return &Router{
		handlers:    make(map[string]HandlerFunc),
		middlewares: make([]MiddlewareFunc, 0),
	}
}

// Use 注册全局中间件
func (r *Router) Use(middlewares ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// Register 注册gRPC路由
func (r *Router) Register(method string, handler HandlerFunc, chain []MiddlewareFunc) {
	r.handlers[method] = buildChain(chain, handler)
}

// Dispatch 路由分发
func (r *Router) Dispatch(ctx *Context) error {
	method := ctx.Method
	handler, exists := r.handlers[method]
	if !exists {
		ctx.JSON(404, map[string]interface{}{
			"code": 404,
			"msg":  "无效的gRPC服务方法",
			"data": nil,
		})
		return errors.New("invalid gRPC method: " + method)
	}
	handler(ctx)
	return nil
}

// buildChain 构建中间件链（与HTTP/WS逻辑一致）
func buildChain(middlewares []MiddlewareFunc, final HandlerFunc) HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		currentMid := middlewares[i]
		currentNext := final
		final = currentMid(currentNext)
	}
	return final
}
