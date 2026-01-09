package controller

import (
	"encoding/json"
)

// BaseController 所有业务控制器的基类
type BaseController struct {
	Ctx *httpD.Context // 请求上下文
}

// Init 初始化控制器（注入上下文）
func (c *BaseController) Init(ctx *httpD.Context) {
	c.Ctx = ctx
}

// -------------------------- 通用响应方法（规范化） --------------------------

// Success 成功响应（统一格式）
func (c *BaseController) Success(data interface{}) {
	c.Ctx.JSON(200, map[string]interface{}{
		"code": 0,
		"msg":  "操作成功",
		"data": data,
	})
}

// Error 错误响应（统一格式）
func (c *BaseController) Error(code int, msg string) {
	c.Ctx.JSON(200, map[string]interface{}{
		"code": code,
		"msg":  msg,
		"data": nil,
	})
}

// -------------------------- 通用参数解析方法 --------------------------

// BindJSON 解析JSON参数
func (c *BaseController) BindJSON(v interface{}) error {
	return json.NewDecoder(c.Ctx.Req.Body).Decode(v)
}

// GetQuery 获取URL查询参数
func (c *BaseController) GetQuery(key string) string {
	return c.Ctx.Query(key)
}

// GetPostForm 获取POST表单参数
func (c *BaseController) GetPostForm(key string) string {
	return c.Ctx.PostForm(key)
}
