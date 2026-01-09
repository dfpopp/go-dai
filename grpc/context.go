package grpc

import (
	"encoding/json"
	"github.com/dfpopp/go-dai/netContext"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Context gRPC上下文（实现netContext.Context和netContext.RequestInfo接口）
type Context struct {
	Req      *http.Request          // 兼容原有上下文结构（实际gRPC场景可忽略，保持接口一致性）
	MD       metadata.MD            // gRPC元数据（对应HTTP Header）
	PeerInfo *peer.Peer             // 客户端信息（用于获取IP）
	Method   string                 // gRPC服务方法名（如/merchant.MemberService/Login）
	Path     string                 // 等同于Method，保持接口一致性
	params   map[string]string      // 自定义参数（对齐HTTP/WS）
	rawData  []byte                 // 原始请求数据（对齐HTTP Body/WS消息）
	respData map[string]interface{} // 响应数据
}

// NewContext 创建gRPC上下文实例
func NewContext(md metadata.MD, peerInfo *peer.Peer, method string, rawData []byte) *Context {
	return &Context{
		MD:       md,
		PeerInfo: peerInfo,
		Method:   method,
		Path:     method,
		params:   make(map[string]string),
		rawData:  rawData,
		respData: make(map[string]interface{}),
	}
}

// -------------------------- 通用控制器签名 --------------------------
type GRPCHandlerFunc func(netContext.Context)

// -------------------------- 适配器方法 --------------------------
func ToGRPCHandler(fn GRPCHandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		fn(ctx)
	}
}

// -------------------------- 编译期校验 --------------------------
var (
	_ netContext.Context     = (*Context)(nil)
	_ netContext.RequestInfo = (*Context)(nil)
)

// -------------------------- 实现netContext.RequestInfo接口 --------------------------
func (c *Context) GetMethod() string {
	return c.Method // gRPC服务方法名
}

func (c *Context) GetPath() string {
	return c.Path // 等同于方法名，保持接口一致性
}

func (c *Context) GetClientIP() string {
	if c.PeerInfo == nil {
		return "unknown"
	}
	addr, ok := c.PeerInfo.Addr.(net.Addr)
	if !ok {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err == nil {
		return host
	}
	return addr.String()
}

func (c *Context) GetHeader(key string) string {
	if c.MD == nil {
		return ""
	}
	vals := c.MD.Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (c *Context) GetQuery(key string) string {
	// gRPC场景下，Query参数从元数据中获取
	return c.GetHeader("x-grpc-query-" + key)
}

// -------------------------- 实现netContext.Context接口 --------------------------
func (c *Context) GetRequestInfo() netContext.RequestInfo {
	return c
}

func (c *Context) JSON(code int, data map[string]interface{}) {
	c.respData = data
}

func (c *Context) String(code int, s string) {
	c.respData = map[string]interface{}{
		"msg":  s,
		"code": code,
	}
}

func (c *Context) Query(key string) string {
	if c.params[key] != "" {
		return c.params[key]
	}
	return c.GetQuery(key)
}

func (c *Context) PostForm(key string) string {
	// gRPC场景下，PostForm参数从原始数据中解析
	if c.params[key] != "" {
		return c.params[key]
	}
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
func (c *Context) BindJSON(v interface{}) error {
	if len(c.rawData) == 0 {
		return json.Unmarshal([]byte("{}"), v)
	}
	return json.Unmarshal(c.rawData, v)
}

func (c *Context) SetParam(key, value string) {
	c.params[key] = value
}

func (c *Context) GetParam(key string) string {
	return c.params[key]
}

// GetResponse 获取响应数据（gRPC特有，用于构建返回结果）
func (c *Context) GetResponse() map[string]interface{} {
	return c.respData
}
