package middleware

import (
	"github.com/dfpopp/go-dai/http"
	"github.com/dfpopp/go-dai/response"
	"sync/atomic"
)

// RateLimit 限流中间件
func RateLimit(maxReq int64) http.MiddlewareFunc {
	var count int64
	return func(next http.HandlerFunc, c *http.Context) {
		if atomic.AddInt64(&count, 1) > maxReq {
			c.JSON(200, response.Error(429, "请求过于频繁，请稍后再试"))
			return
		}
		next(c)
	}
}
