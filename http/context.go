package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/netContext"
	"io"
	"net"
	"net/http"
	"strings"
)

// Context HTTP请求上下文
type Context struct {
	Writer http.ResponseWriter
	Req    *http.Request
	Params map[string]string // 路径参数
}

// NewContext 创建上下文实例
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    r,
		Params: make(map[string]string),
	}
}

// -------------------------- 通用控制器签名（依赖netContext通用接口） --------------------------

// HTTPHandlerFunc 通用控制器方法签名（入参为通用Context接口）
type HTTPHandlerFunc func(netContext.Context)

// -------------------------- 适配器方法（移到http包中，不再放在netContext） --------------------------
const (
	// 限制最大请求体大小（根据业务调整，比如10MB）
	maxBodySize = 10 * 1024 * 1024
)

// ToHTTPHandler 将通用控制器方法转换为http.HandlerFunc
func ToHTTPHandler(fn HTTPHandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		// 将http.Context（实现了netContext.Context接口）传入通用控制器方法
		fn(ctx)
	}
}

// -------------------------- 编译期校验（移到http包中，验证http.Context实现通用接口） --------------------------
var (
	_ netContext.Context     = (*Context)(nil) // 验证上下文接口实现
	_ netContext.RequestInfo = (*Context)(nil) // 验证请求信息接口实现
)

// -------------------------- 实现通用context.RequestInfo接口 --------------------------

func (c *Context) GetMethod() string {
	return c.Req.Method // HTTP请求方法（GET/POST/PUT等）
}

func (c *Context) GetPath() string {
	return c.Req.URL.Path // HTTP请求路径（如/merchant/login）
}

func (c *Context) GetClientIP() string {
	// 通用客户端IP获取逻辑（兼容反向代理）
	ip := c.Req.Header.Get("X-Real-IP")
	forwardedFor := c.Req.Header.Get("X-Forwarded-For")
	if strings.Contains(forwardedFor, ip) {
		ip = strings.Split(forwardedFor, ",")[0]
	} else {
		if ip == "" {
			ip = c.Req.Header.Get("X-Forwarded-For")
			if ip != "" {
				ip = strings.Split(ip, ",")[0]
			}
		}
	}
	if ip == "" {
		remoteAddr := c.Req.RemoteAddr
		host, _, err := net.SplitHostPort(remoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = remoteAddr
		}
	}
	return ip
}

func (c *Context) GetHeader(key string) string {
	return c.Req.Header.Get(key) // HTTP请求头
}

func (c *Context) GetQuery(key string) string {
	return c.Req.URL.Query().Get(key) // HTTP查询参数
}

// -------------------------- 实现通用context.Context接口 --------------------------

func (c *Context) GetRequestInfo() netContext.RequestInfo {
	return c // HTTP上下文自身实现了RequestInfo，直接返回
}

// JSON 返回JSON格式响应
func (c *Context) JSON(code int, data map[string]interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json;charset=utf-8")
	c.Writer.WriteHeader(code)
	if err := json.NewEncoder(c.Writer).Encode(data); err != nil {
		http.Error(c.Writer, "JSON序列化失败", http.StatusInternalServerError)
	}
}

// String 返回字符串格式响应
func (c *Context) String(code int, s string) {
	c.Writer.Header().Set("Content-Type", "text/plain;charset=utf-8")
	c.Writer.WriteHeader(code)
	_, _ = c.Writer.Write([]byte(s))
}

// Query 获取URL查询参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// PostForm 获取POST表单参数
func (c *Context) PostForm(key string) string {
	return c.Req.PostFormValue(key)
}

// PostFormAll 获取所有POST表单数据，返回键值对映射
func (c *Context) PostFormAll() map[string]string {
	// 1. 解析表单（必须先解析，否则无法获取全部数据）
	// ParseForm 会解析 POST/PUT/PATCH 的表单数据，同时也兼容 GET 的 URL 参数
	if err := c.Req.ParseForm(); err != nil {
		// 解析失败时返回空map，避免后续操作panic
		return make(map[string]string)
	}

	// 2. Req.PostForm 是 url.Values 类型（本质是 map[string][]string），包含所有POST表单数据
	allPostData := make(map[string]string, len(c.Req.PostForm))
	for key, values := range c.Req.PostForm {
		// 深拷贝值，避免外部修改影响内部数据
		allPostData[key] = strings.Join(values, ",")
	}
	return allPostData
}

func (c *Context) GetBody() ([]byte, error) {
	// 前置校验：避免nil指针
	if c.Req == nil {
		return nil, errors.New("request对象未初始化")
	}

	// 限制请求体大小，防止OOM
	c.Req.Body = http.MaxBytesReader(c.Writer, c.Req.Body, maxBodySize)
	// 读取原始Body
	bodyBytes, err := io.ReadAll(c.Req.Body)
	if err != nil {
		// 区分“超出大小限制”和普通读取错误
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, fmt.Errorf("请求体超出最大限制（%dMB）：%w", maxBodySize/1024/1024, err)
		}
		return nil, fmt.Errorf("读取请求体失败：%w", err)
	}
	// 重置Body：供后续代码重复读取（你的原有逻辑保留，无问题）
	c.Req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, nil
}

// BindJSON 绑定JSON请求体到结构体
func (c *Context) BindJSON(v interface{}) error {
	decoder := json.NewDecoder(c.Req.Body)
	defer c.Req.Body.Close()
	return decoder.Decode(v)
}

// SetParam 设置路径参数
func (c *Context) SetParam(key, value string) {
	c.Params[key] = value
}

// GetParam 获取路径参数
func (c *Context) GetParam(key string) string {
	return c.Params[key]
}
