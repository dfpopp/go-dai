package netContext

// -------------------------- 通用请求信息接口（解耦具体协议） --------------------------

// RequestInfo 通用请求信息接口，抽象所有协议的公共请求属性
// 新增gRPC/MQTT等协议时，只需实现该接口即可兼容
type RequestInfo interface {
	GetMethod() string           // 获取请求标识（HTTP：GET/POST；WS：action；gRPC：服务方法名）
	GetPath() string             // 获取请求路径/唯一标识（HTTP：/api/login；WS：action；gRPC：/pkg.Service/Method）
	GetClientIP() string         // 获取客户端IP（通用所有协议）
	GetHeader(key string) string // 获取请求头/元数据（HTTP：Header；WS：握手Header；gRPC：Metadata）
	GetQuery(key string) string  // 获取查询参数/附加参数（HTTP：URL.Query；WS：握手Query；gRPC：Metadata）
}

// Context 通用上下文接口（包含HTTP和WS上下文的公共方法）
type Context interface {
	JSON(code int, data map[string]interface{})
	String(code int, s string)
	Query(key string) string
	PostForm(key string) string
	PostFormAll() map[string]string
	GetBody() ([]byte, error)
	BindJSON(v interface{}) error
	SetParam(key, value string)
	GetParam(key string) string
	GetRequestInfo() RequestInfo // 返回通用请求信息，替代直接返回*http.Request
}

// GRPCHandlerFunc 通用gRPC处理器签名（框架层定义，应用层复用）
type GRPCHandlerFunc func(Context)
