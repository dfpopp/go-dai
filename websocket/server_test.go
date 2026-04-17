package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewServer 测试创建WebSocket服务器
func TestNewServer(t *testing.T) {
	// 测试创建服务器
	server := NewServer("test")
	if server == nil {
		t.Error("NewServer should return non-nil server")
	}
	
	// 测试获取配置
	config := server.Config()
	if config == nil {
		t.Error("Server.Config should return non-nil config")
	}
	
	// 测试默认配置
	if config.Path == "" {
		t.Error("Default path should not be empty")
	}
	if config.Origin == "" {
		t.Error("Default origin should not be empty")
	}
	if config.MaxMessageSize == 0 {
		t.Error("Default max message size should not be 0")
	}
	if config.MaxConnections == 0 {
		t.Error("Default max connections should not be 0")
	}
}

// TestServer_Use 测试注册中间件
func TestServer_Use(t *testing.T) {
	server := NewServer("test")
	
	// 测试注册中间件
	middleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	}
	
	server.Use(middleware)
	if len(server.middlewares) == 0 {
		t.Error("Server should have at least one middleware")
	}
}

// TestServer_Register 测试注册路由
func TestServer_Register(t *testing.T) {
	server := NewServer("test")
	
	// 测试注册路由
	handler := func(ctx *Context) {
		// 测试处理函数
	}
	
	middleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	}
	
	server.Register("test-action", handler, middleware)
	// 路由注册后没有直接的方法检查，这里只测试注册过程不报错
}

// TestServer_Stop 测试停止服务器
func TestServer_Stop(t *testing.T) {
	server := NewServer("test")
	
	// 测试停止服务器
	err := server.Stop()
	if err != nil {
		t.Errorf("Server.Stop should not return error: %v", err)
	}
}

// TestConn_ReadMessage 测试读取消息
func TestConn_ReadMessage(t *testing.T) {
	// 创建测试连接（这里我们不实际创建网络连接，只测试结构）
	conn := &Conn{
		conn:         nil,
		readBuf:      make([]byte, 4096),
		writeBuf:     make([]byte, 4096),
		maxMsgSize:   1024 * 1024,
		readTimeout:  60 * time.Second,
		writeTimeout: 60 * time.Second,
	}
	
	// 测试ReadMessage方法（会失败，因为conn为nil）
	_, err := conn.ReadMessage()
	if err == nil {
		t.Error("Conn.ReadMessage should return error when conn is nil")
	}
	
	// 测试WriteMessage方法（会失败，因为conn为nil）
	err = conn.WriteMessage("test message")
	if err == nil {
		t.Error("Conn.WriteMessage should return error when conn is nil")
	}
	
	// 测试WriteBinaryMessage方法（会失败，因为conn为nil）
	err = conn.WriteBinaryMessage([]byte("test message"))
	if err == nil {
		t.Error("Conn.WriteBinaryMessage should return error when conn is nil")
	}
	
	// 测试WriteCloseMessage方法（会失败，因为conn为nil）
	err = conn.WriteCloseMessage(1000, "normal closure")
	if err == nil {
		t.Error("Conn.WriteCloseMessage should return error when conn is nil")
	}
	
	// 测试Close方法（会失败，因为conn为nil）
	err = conn.Close()
	if err == nil {
		t.Error("Conn.Close should return error when conn is nil")
	}
	
	// 测试RemoteAddr方法
	addr := conn.RemoteAddr()
	if addr == "" {
		t.Error("Conn.RemoteAddr should return non-empty string")
	}
}

// TestHandleRequest 测试处理WebSocket请求
func TestHandleRequest(t *testing.T) {
	server := NewServer("test")
	
	// 创建测试请求
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "test-key")
	
	// 创建测试响应
	w := httptest.NewRecorder()
	
	// 测试处理请求（会失败，因为不是真正的WebSocket连接）
	server.handleRequest(w, req)
	
	// 验证响应状态码
	if w.Code != http.StatusBadRequest && w.Code != http.StatusGatewayTimeout {
		t.Errorf("Expected status code 400 or 504, got %d", w.Code)
	}
}
