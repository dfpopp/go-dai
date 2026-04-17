package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dfpopp/go-dai/logger"

	"github.com/dfpopp/go-dai/netContext"
)

// Context HTTP请求上下文
type Context struct {
	Writer       http.ResponseWriter
	Req          *http.Request
	ctx          context.Context
	cancel       context.CancelFunc
	Params       map[string]string // 路径参数
	values       map[string]interface{}
	cleanupFuncs []func() // 清理函数列表
}

// NewContext 创建上下文实例
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		Writer:       w,
		Req:          r,
		ctx:          ctx,
		cancel:       cancel,
		Params:       make(map[string]string),
		values:       make(map[string]interface{}),
		cleanupFuncs: make([]func(), 0),
	}
}

// RegisterCleanupFunc 注册清理函数
func (c *Context) RegisterCleanupFunc(f func()) {
	c.cleanupFuncs = append(c.cleanupFuncs, f)
}

// Cleanup 执行清理函数
func (c *Context) Cleanup() {
	for _, f := range c.cleanupFuncs {
		f()
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
		defer ctx.Cleanup()
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
	// 设置Content-Type
	c.Writer.Header().Set("Content-Type", "application/json;charset=utf-8")
	c.Writer.WriteHeader(code)
	if err := json.NewEncoder(c.Writer).Encode(data); err != nil {
		logger.Error(err)
	}
}

// String 返回字符串格式响应
func (c *Context) String(code int, s string) {
	// 设置Content-Type
	c.Writer.Header().Set("Content-Type", "text/plain;charset=utf-8")
	c.Writer.WriteHeader(code)
	_, err := c.Writer.Write([]byte(s))
	if err != nil {
		logger.Error(err)
	}
}

// Query 获取URL查询参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// PostForm 获取POST表单参数
func (c *Context) PostForm(key string) string {
	return c.Req.PostFormValue(key)
}

// PostFormAll 兼容获取普通表单(x-www-form-urlencoded)和multipart/form-data的表单数据
// 核心优化：根据Content-Type选择性解析，避免无效解析和错误
func (c *Context) PostFormAll() map[string]string {
	form := make(map[string]string)
	contentType := c.Req.Header.Get("Content-Type")

	// 1. 判断请求类型，针对性解析
	if strings.Contains(contentType, "multipart/form-data") {
		// 处理 multipart/form-data 类型（文件上传类表单）
		if c.Req.MultipartForm == nil {
			// 解析multipart表单，32MB内存限制
			err := c.Req.ParseMultipartForm(32 << 20)
			if err != nil {
				logger.Error("解析multipart表单失败：", err)
				return nil
			}
		}
		// 读取multipart表单的普通字段
		for k, v := range c.Req.MultipartForm.Value {
			if len(v) > 0 {
				form[k] = v[0]
			}
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") || c.Req.Method == "POST" {
		// 处理普通表单（x-www-form-urlencoded）或POST请求的默认表单
		err := c.Req.ParseForm()
		if err != nil {
			logger.Error("解析普通表单失败：", err)
			return nil
		}
		// 读取普通表单数据
		for k, v := range c.Req.Form {
			if len(v) > 0 {
				form[k] = v[0]
			}
		}
	} else {
		// 非表单类型请求（如json、text等），返回空map而非nil，避免业务报错
		logger.Warn("请求非表单类型，Content-Type：", contentType)
	}
	return form
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
		// 区分"超出大小限制"和普通读取错误
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

// GetRequest 获取http.Request对象，用于获取上传的文件等
func (c *Context) GetRequest() (*http.Request, error) {
	return c.Req, nil
}

// SetParam 设置路径参数
func (c *Context) SetParam(key, value string) {
	c.Params[key] = value
}

// GetParam 获取路径参数
func (c *Context) GetParam(key string) string {
	return c.Params[key]
}

// Set 设置上下文值
func (c *Context) Set(key string, value interface{}) {
	c.values[key] = value
}

// Get 获取上下文值
func (c *Context) Get(key string) (interface{}, bool) {
	value, ok := c.values[key]
	return value, ok
}

// Deadline 获取上下文截止时间
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

// Done 获取上下文完成通道
func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err 获取上下文错误
func (c *Context) Err() error {
	return c.ctx.Err()
}

// Value 获取上下文值
func (c *Context) Value(key interface{}) interface{} {
	if keyStr, ok := key.(string); ok {
		if value, exists := c.values[keyStr]; exists {
			return value
		}
	}
	return c.ctx.Value(key)
}
