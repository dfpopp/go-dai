package http

import (
	"net/http"
)

// Router HTTP路由器（框架内置，负责路由注册、映射存储与中间件链构建）
type Router struct {
	mux               *http.ServeMux         // 系统ServeMux，负责HTTP请求分发
	handlers          map[string]HandlerFunc // 存储「method+path」与处理器的映射
	globalMiddlewares []MiddlewareFunc       // 全局中间件
}

// NewRouter 创建HTTP路由器实例
func NewRouter() *Router {
	return &Router{
		mux:               http.NewServeMux(),
		handlers:          make(map[string]HandlerFunc),
		globalMiddlewares: make([]MiddlewareFunc, 0),
	}
}

// ServeHTTP 实现http.Handler接口，兼容系统HTTP服务
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Use 注册全局中间件
func (r *Router) Use(middlewares ...MiddlewareFunc) {
	r.globalMiddlewares = append(r.globalMiddlewares, middlewares...)
}

// buildChain 构建中间件链（内部方法）
func (r *Router) buildChain(handler HandlerFunc, localMiddlewares []MiddlewareFunc) HandlerFunc {
	// 合并全局中间件与局部中间件
	allMiddlewares := append(r.globalMiddlewares, localMiddlewares...)
	finalHandler := handler

	// 倒序构建中间件链
	for i := len(allMiddlewares) - 1; i >= 0; i-- {
		currentMid := allMiddlewares[i]
		currentNext := finalHandler
		finalHandler = currentMid(currentNext)
	}
	return finalHandler
}

// wrapHandler 包装处理器为http.HandlerFunc（内部方法）
func (r *Router) wrapHandler(handler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := NewContext(w, req)
		handler(ctx)
	}
}

// Handle 注册通用路由（核心方法，接收HTTP方法、路径、处理器与局部中间件）
func (r *Router) Handle(method, path string, handler HandlerFunc, localMiddlewares ...MiddlewareFunc) {
	// 1. 构建完整中间件链
	chainHandler := r.buildChain(handler, localMiddlewares)
	// 2. 生成唯一路由键（method + path）
	routeKey := method + " " + path
	// 3. 存储路由映射
	r.handlers[routeKey] = chainHandler
	// 4. 注册到系统ServeMux
	r.mux.HandleFunc(path, r.wrapHandler(chainHandler))
}

// GET 快捷注册GET请求路由
func (r *Router) GET(path string, handler HandlerFunc, localMiddlewares ...MiddlewareFunc) {
	r.Handle("GET", path, handler, localMiddlewares...)
}

// POST 快捷注册POST请求路由
func (r *Router) POST(path string, handler HandlerFunc, localMiddlewares ...MiddlewareFunc) {
	r.Handle("POST", path, handler, localMiddlewares...)
}

// PUT 快捷注册PUT请求路由（可选扩展，保持风格一致）
func (r *Router) PUT(path string, handler HandlerFunc, localMiddlewares ...MiddlewareFunc) {
	r.Handle("PUT", path, handler, localMiddlewares...)
}

// DELETE 快捷注册DELETE请求路由（可选扩展，保持风格一致）
func (r *Router) DELETE(path string, handler HandlerFunc, localMiddlewares ...MiddlewareFunc) {
	r.Handle("DELETE", path, handler, localMiddlewares...)
}
