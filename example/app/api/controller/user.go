package controller

import (
	"github.com/dfpopp/go-dai/example/app/api/service"
	"github.com/dfpopp/go-dai/http"
	"github.com/dfpopp/go-dai/response"
)

// UserController 用户控制器
type UserController struct {
	userService *service.UserService
}

// NewUserController 创建控制器
func NewUserController() *UserController {
	return &UserController{
		userService: service.NewUserService(),
	}
}

// Login 登录接口
func (c *UserController) Login(ctx *http.Context) {
	username := ctx.Query("username")
	password := ctx.Query("password")

	if username == "" || password == "" {
		ctx.JSON(200, response.Error(400, "用户名或密码不能为空"))
		return
	}

	user, err := c.userService.Login(username, password)
	if err != nil {
		ctx.JSON(200, response.Error(500, err.Error()))
		return
	}

	ctx.JSON(200, response.Success(user))
}
