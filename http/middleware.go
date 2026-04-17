package http

import (
	"net/http"

	"github.com/dfpopp/go-dai/logger"
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
					return
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
			// 为所有请求设置CORS响应头
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // 允许所有域
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24小时
			// 对于OPTIONS请求，直接返回200状态码
			if c.Req.Method == "OPTIONS" {
				c.Writer.WriteHeader(http.StatusOK)
				return
			}
			// 非OPTIONS请求，继续处理
			next(c)
		}
	}
}
