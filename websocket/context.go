package websocket

import (
	"encoding/json"
	"github.com/dfpopp/go-dai/logger"
	"github.com/dfpopp/go-dai/netContext"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Context WebSocket上下文（与http.Context方法签名完全一致）
type Context struct {
	Conn      *Conn             // WS连接实例
	Req       *http.Request     // 握手阶段的HTTP请求（兼容ctx.Req）
	Action    string            // 对应HTTP的URL.Path（WS消息action）
	RequestId string            // 请求唯一标识
	params    map[string]string // 存储查询参数/POST参数（模拟HTTP参数）
	rawData   []byte            // 原始消息数据（对应HTTP请求体）
	ConnID    string            // 新增：当前连接的唯一ID
}

// NewContext 创建WS上下文（对应HTTP上下文初始化）
func NewContext(conn *Conn, req *http.Request, action, requestId, connID string, rawData []byte) *Context {
	return &Context{
		Conn:      conn,
		Req:       req,
		Action:    action,
		RequestId: requestId,
		rawData:   rawData,
		params:    make(map[string]string),
		ConnID:    connID, // 赋值连接ID
	}
}

// -------------------------- 通用控制器签名 --------------------------

// WSHandlerFunc 通用控制器方法签名（入参为通用Context接口）
type WSHandlerFunc func(netContext.Context)

// -------------------------- 适配器方法（移到websocket包中） --------------------------

// ToWSHandler 将通用控制器方法转换为websocket.HandlerFunc
func ToWSHandler(fn WSHandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		fn(ctx)
	}
}

// -------------------------- 编译期校验 --------------------------
var (
	_ netContext.Context     = (*Context)(nil) // 验证上下文接口实现
	_ netContext.RequestInfo = (*Context)(nil) // 验证请求信息接口实现
)

// -------------------------- 新增：ConnIDContext接口（扩展通用上下文） --------------------------
type ConnIDContext interface {
	netContext.Context
	GetConnID() string // 获取连接ID
}

// 验证Context实现ConnIDContext接口
var _ ConnIDContext = (*Context)(nil)

// GetConnID 获取当前连接的唯一ID（应用层控制器可调用）
func (c *Context) GetConnID() string {
	return c.ConnID
}

// -------------------------- 实现通用context.RequestInfo接口 --------------------------

func (c *Context) GetMethod() string {
	return c.Action // WS场景：用action作为请求方法标识
}

func (c *Context) GetPath() string {
	return c.Action // WS场景：用action作为请求唯一标识
}

func (c *Context) GetClientIP() string {
	// 复用IP获取逻辑（从WS握手请求中提取）
	ip := c.Req.Header.Get("X-Real-IP")
	if ip == "" {
		ip = c.Req.Header.Get("X-Forwarded-For")
		if ip != "" {
			ip = strings.Split(ip, ",")[0]
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
	return c.Req.Header.Get(key) // WS场景：从握手请求获取头信息
}

func (c *Context) GetQuery(key string) string {
	return c.Req.URL.Query().Get(key) // WS场景：从握手请求获取查询参数
}

// -------------------------- 实现通用context.Context接口 --------------------------

func (c *Context) GetRequestInfo() netContext.RequestInfo {
	return c // WS上下文自身实现了RequestInfo，直接返回
}

// -------------------------- 与http.Context一致的方法实现 --------------------------

// JSON 统一JSON响应（与HTTP上下文JSON方法完全一致）
func (c *Context) JSON(code int, data map[string]interface{}) {
	// 序列化响应并发送
	respBytes, err := json.Marshal(data)
	if err != nil {
		logger.Error("WS上下文JSON序列化失败：", err)
		return
	}
	_ = c.Conn.WriteMessage(string(respBytes))
}
func (c *Context) String(code int, s string) {
	_ = c.Conn.WriteMessage(s)
}

// Query 获取URL查询参数（模拟HTTP Query，从握手请求中获取）
func (c *Context) Query(key string) string {
	if c.params[key] != "" {
		return c.params[key]
	}
	// 从握手请求的URL查询参数中获取
	return c.Req.URL.Query().Get(key)
}

// PostForm 获取POST表单参数（模拟HTTP PostForm，从WS消息数据中解析）
func (c *Context) PostForm(key string) string {
	if c.params[key] != "" {
		return c.params[key]
	}
	// 若消息数据是表单格式（key=value&...），解析后返回
	if len(c.rawData) > 0 {
		formData := strings.Split(string(c.rawData), "&")
		for _, item := range formData {
			kv := strings.Split(item, "=")
			if len(kv) == 2 && kv[0] == key {
				c.params[key] = kv[1]
				return kv[1]
			}
		}
	}
	return ""
}

// PostFormAll 获取所有POST表单参数，返回解析后的键值对
func (c *Context) PostFormAll() map[string]string {
	if len(c.params) > 0 {
		return c.params
	}
	// 若消息数据是表单格式（key=value&...），解析后返回
	if len(c.rawData) > 0 {
		// 分割原始数据为多个键值对（& 分隔）
		formData := strings.Split(string(c.rawData), "&")
		for _, item := range formData {
			// 跳过空项（比如数据末尾多一个&的情况）
			if item == "" {
				continue
			}

			// 分割键值对（= 分隔）
			kv := strings.SplitN(item, "=", 2) // 用 SplitN 避免值中包含=的情况
			if len(kv) == 1 {
				// 只有key没有value的情况（比如 "key3&key1=val1"），value设为空字符串
				c.params[kv[0]] = ""
			} else if len(kv) == 2 {
				// 标准键值对，处理 URL 解码（比如空格转义为%20、中文转义等）
				key := kv[0]
				val, err := url.QueryUnescape(kv[1])
				if err != nil {
					val = kv[1] // 解码失败则用原始值
				}
				c.params[key] = val
			}
		}
	}
	return c.params
}
func (c *Context) GetBody() ([]byte, error) {
	// 若消息数据是表单格式（key=value&...），解析后返回
	if len(c.rawData) > 0 {
		return c.rawData, nil
	}
	return []byte{}, nil
}

// BindJSON 绑定JSON请求体到结构体（与HTTP上下文BindJSON方法完全一致）
func (c *Context) BindJSON(v interface{}) error {
	if len(c.rawData) == 0 {
		return json.Unmarshal([]byte("{}"), v)
	}
	return json.Unmarshal(c.rawData, v)
}

// SetParam 手动设置参数（供中间件使用，兼容HTTP上下文参数传递）
func (c *Context) SetParam(key, value string) {
	c.params[key] = value
}

// GetParam 获取自定义参数（兼容HTTP上下文扩展）
func (c *Context) GetParam(key string) string {
	return c.params[key]
}

// GetRequest 实现通用context.Context接口（返回握手阶段的HTTP请求）
func (c *Context) GetRequest() *http.Request {
	return c.Req
}
