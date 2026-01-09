package http

import (
	"github.com/dfpopp/go-dai/logger"
	"net/http"
)

// HandlerFunc 自定义HTTP处理器
type HandlerFunc func(*Context)

// MiddlewareFunc 中间件函数类型
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Recovery 异常恢复中间件
func Recovery() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("请求异常：", err)
					c.JSON(http.StatusInternalServerError, map[string]interface{}{
						"code": 500,
						"msg":  "服务器内部错误",
					})
				}
			}()
			next(c)
		}
	}
}

// CORS 跨域中间件（默认实现）
func CORS() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // 允许所有域
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
			if c.Req.Method == "OPTIONS" {
				c.Writer.WriteHeader(http.StatusOK)
				return
			}
			next(c)
		}
	}
}
