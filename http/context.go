package http

import (
	"encoding/json"
	"net/http"
)

// Context 自定义请求上下文
type Context struct {
	Writer http.ResponseWriter
	Req    *http.Request
	Params map[string]string
}

// NewContext 创建上下文
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    r,
		Params: make(map[string]string),
	}
}

// JSON 返回JSON响应
func (c *Context) JSON(code int, data interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json;charset=utf-8")
	c.Writer.WriteHeader(code)
	_ = json.NewEncoder(c.Writer).Encode(data)
}

// JSON 返回JSON响应
func (c *Context) String(code int, s string) {
	c.Writer.Header().Set("Content-Type", "application/json;charset=utf-8")
	c.Writer.WriteHeader(code)
	_, _ = c.Writer.Write([]byte(s)) // 直接写入字符串
}

// Query 获取URL参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// PostForm 获取POST表单参数
func (c *Context) PostForm(key string) string {
	return c.Req.PostFormValue(key)
}
