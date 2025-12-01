package router

import (
	"github.com/dfpopp/go-dai/example/app/api/controller"
	"github.com/dfpopp/go-dai/example/app/api/middleware"
	"github.com/dfpopp/go-dai/http"
)

// Init 初始化API路由
func Init(server *http.Server) {
	userCtrl := controller.NewUserController()
	// 注册登录路由，添加限流（最大200次请求）
	server.GET("/api/user/login", userCtrl.Login, middleware.RateLimit(200))
}
