package http

import (
	"fmt"
	"github.com/dfpopp/go-dai/db/redisDb"
	"net/http"
)

// MiddlewareFunc 中间件函数类型
type MiddlewareFunc func(next HandlerFunc, c *Context)

// HandlerFunc 自定义处理器
type HandlerFunc func(*Context)

// Recovery 异常恢复中间件
func Recovery() MiddlewareFunc {
	fmt.Println("hello")
	defer func() {
		err := redisDb.CloseRedis()
		if err != nil {
			fmt.Println("《《关闭错误：" + err.Error())
		}
	}()
	return func(next HandlerFunc, c *Context) {
		fmt.Println("hello1")
		defer func() {
			fmt.Println("hello2")
			if err := recover(); err != nil {
				c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"code": 500,
					"msg":  "服务器内部错误",
					"err":  err,
				})
			}
		}()
		next(c)
	}
}
