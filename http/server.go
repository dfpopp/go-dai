package http

import (
	"net/http"
)

// Server HTTP服务结构体
type Server struct {
	router      *http.ServeMux
	middlewares []MiddlewareFunc
	appName     string
}

// NewServer 创建服务实例
func NewServer(appName string) *Server {
	return &Server{
		router:  http.NewServeMux(),
		appName: appName,
	}
}

// Use 注册全局中间件
func (s *Server) Use(middlewares ...MiddlewareFunc) {
	s.middlewares = append(s.middlewares, middlewares...)
}

// Handle 注册路由
func (s *Server) Handle(method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	chain := append(s.middlewares, middlewares...)
	finalHandler := wrapHandler(handler, chain)

	fullPath := method + " " + path
	s.router.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		finalHandler(w, r)
	})
}

// GET 快捷注册GET路由
func (s *Server) GET(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	s.Handle("GET", path, handler, middlewares...)
}

// POST 快捷注册POST路由
func (s *Server) POST(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	s.Handle("POST", path, handler, middlewares...)
}

// Run 启动服务
func (s *Server) Run(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

// wrapHandler 包装处理器和中间件
func wrapHandler(handler HandlerFunc, middlewares []MiddlewareFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(w, r)
		chain := buildChain(middlewares, handler)
		chain(ctx)
	}
}

// buildChain 构建中间件链
func buildChain(middlewares []MiddlewareFunc, final HandlerFunc) HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		mid := middlewares[i]
		next := final
		final = func(c *Context) {
			mid(next, c)
		}
	}
	return final
}