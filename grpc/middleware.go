package grpc

import (
	"github.com/dfpopp/go-dai/logger"
)

// HandlerFunc gRPC处理器函数
type HandlerFunc func(*Context)

// MiddlewareFunc gRPC中间件函数类型
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Recovery 异常恢复中间件（对齐HTTP Recovery）
func Recovery() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("gRPC请求异常：", err)
					c.JSON(500, map[string]interface{}{
						"code": 500,
						"msg":  "服务器内部错误",
						"data": nil,
					})
				}
			}()
			next(c)
		}
	}
}
