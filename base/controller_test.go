package base

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/dfpopp/go-dai/netContext"
)

// MockContext 模拟上下文实现
type MockContext struct {
	connID string
	path   string
}

func (m *MockContext) GetConnID() string {
	return m.connID
}

func (m *MockContext) JSON(code int, data map[string]interface{}) {
	// 模拟JSON响应
}

func (m *MockContext) String(code int, data string) {
	// 模拟字符串响应
}

func (m *MockContext) Query(key string) string {
	return ""
}

func (m *MockContext) PostForm(key string) string {
	return ""
}

func (m *MockContext) PostFormAll() map[string]string {
	return nil
}

func (m *MockContext) GetBody() ([]byte, error) {
	return nil, nil
}

func (m *MockContext) BindJSON(v interface{}) error {
	return nil
}

func (m *MockContext) GetRequestInfo() netContext.RequestInfo {
	return &MockRequestInfo{path: m.path}
}

// 实现netContext.Context接口的其他方法
func (m *MockContext) Deadline() (deadline time.Time, ok bool) {
	return
}

func (m *MockContext) Done() <-chan struct{} {
	return nil
}

func (m *MockContext) Err() error {
	return nil
}

func (m *MockContext) Value(key interface{}) interface{} {
	return nil
}

// 实现netContext.Context接口的Get方法
func (m *MockContext) Get(key string) (interface{}, bool) {
	return nil, false
}

// 实现netContext.Context接口的Set方法
func (m *MockContext) Set(key string, value interface{}) {
	// 模拟设置值
}

// 实现netContext.Context接口的GetParam方法
func (m *MockContext) GetParam(key string) string {
	return ""
}

// 实现netContext.Context接口的SetParam方法
func (m *MockContext) SetParam(key string, value string) {
	// 模拟设置参数
}

// 实现netContext.Context接口的GetRequest方法
func (m *MockContext) GetRequest() (*http.Request, error) {
	return nil, nil
}

// 实现netContext.Context接口的RegisterCleanupFunc方法
func (m *MockContext) RegisterCleanupFunc(f func()) {
	// 模拟注册清理函数
}

// MockRequestInfo 模拟请求信息
type MockRequestInfo struct {
	path string
}

func (m *MockRequestInfo) GetPath() string {
	return m.path
}

// 实现netContext.RequestInfo接口的GetQuery方法
func (m *MockRequestInfo) GetQuery(key string) string {
	return ""
}

// 实现netContext.RequestInfo接口的其他方法
func (m *MockRequestInfo) GetClientIP() string {
	return "127.0.0.1"
}

func (m *MockRequestInfo) GetMethod() string {
	return "GET"
}

func (m *MockRequestInfo) GetHeader(key string) string {
	return ""
}

func (m *MockRequestInfo) GetHeaders() map[string]string {
	return nil
}

func (m *MockRequestInfo) GetQueryParam(key string) string {
	return ""
}

func (m *MockRequestInfo) GetQueryParams() map[string]string {
	return nil
}

func (m *MockRequestInfo) GetPostParam(key string) string {
	return ""
}

func (m *MockRequestInfo) GetPostParams() map[string]string {
	return nil
}

func (m *MockRequestInfo) GetCookie(key string) string {
	return ""
}

func (m *MockRequestInfo) GetCookies() map[string]string {
	return nil
}

func (m *MockRequestInfo) GetReferer() string {
	return ""
}

func (m *MockRequestInfo) GetUserAgent() string {
	return ""
}

func (m *MockRequestInfo) GetProtocol() string {
	return "HTTP/1.1"
}

func (m *MockRequestInfo) GetHost() string {
	return "localhost"
}

func (m *MockRequestInfo) GetPort() string {
	return "8080"
}

func (m *MockRequestInfo) GetURL() string {
	return "http://localhost:8080" + m.path
}

func (m *MockRequestInfo) GetRawQuery() string {
	return ""
}

func (m *MockRequestInfo) GetContentLength() int64 {
	return 0
}

func (m *MockRequestInfo) GetContentType() string {
	return "application/json"
}

// MockConnManager 模拟连接管理器
type MockConnManager struct {
	conns map[string]interface{}
	attrs map[string]map[string]interface{}
}

func NewMockConnManager() *MockConnManager {
	return &MockConnManager{
		conns: make(map[string]interface{}),
		attrs: make(map[string]map[string]interface{}),
	}
}

func (m *MockConnManager) GetConnByConnID(connID string) (interface{}, bool) {
	conn, ok := m.conns[connID]
	return conn, ok
}

func (m *MockConnManager) RemoveConn(connID, reason string) {
	delete(m.conns, connID)
	delete(m.attrs, connID)
}

func (m *MockConnManager) CloseConnByConnID(connID, reason string) error {
	delete(m.conns, connID)
	delete(m.attrs, connID)
	return nil
}

func (m *MockConnManager) SetConnAttr(connID, key string, value interface{}) {
	if _, ok := m.attrs[connID]; !ok {
		m.attrs[connID] = make(map[string]interface{})
	}
	m.attrs[connID][key] = value
}

func (m *MockConnManager) GetConnAttr(connID, key string) (interface{}, bool) {
	if attrs, ok := m.attrs[connID]; ok {
		value, ok := attrs[key]
		return value, ok
	}
	return nil, false
}

func (m *MockConnManager) RemoveConnAttr(connID, key string) {
	if attrs, ok := m.attrs[connID]; ok {
		delete(attrs, key)
	}
}

func (m *MockConnManager) SendToConnID(connID, message string) error {
	_, ok := m.conns[connID]
	if !ok {
		return errors.New("connection not found")
	}
	return nil
}

func (m *MockConnManager) Multicast(connIDs []string, message string) {
	// 模拟广播
}

func (m *MockConnManager) Broadcast(message string) {
	// 模拟广播
}

func (m *MockConnManager) GetConnCount() int32 {
	return int32(len(m.conns))
}

func (m *MockConnManager) IsConnAlive(connID string) bool {
	_, ok := m.conns[connID]
	return ok
}

func (m *MockConnManager) GetAllConnIDs() []string {
	connIDs := make([]string, 0, len(m.conns))
	for connID := range m.conns {
		connIDs = append(connIDs, connID)
	}
	return connIDs
}

func (m *MockConnManager) GetConnIDsByAttr(key string, value interface{}) []string {
	return nil
}

// TestBaseController_Init 测试控制器初始化
func TestBaseController_Init(t *testing.T) {
	controller := &BaseController{}
	ctx := &MockContext{path: "/test"}
	controller.Init(ctx)

	// 验证控制器是否能正常工作（通过调用一个方法来测试）
	query := controller.GetQuery("test")
	if query != "" {
		t.Errorf("Expected empty string, got %s", query)
	}
}

// TestBaseController_WsInit 测试WebSocket控制器初始化
func TestBaseController_WsInit(t *testing.T) {
	controller := &BaseController{}
	ctx := &MockContext{connID: "test-conn-id", path: "/ws/test"}
	controller.WsInit(ctx)

	// 验证控制器是否能正常工作（通过调用一个方法来测试）
	query := controller.GetQuery("test")
	if query != "" {
		t.Errorf("Expected empty string, got %s", query)
	}

	// 验证连接ID是否缓存
	if controller.GetConnID() != "test-conn-id" {
		t.Error("ConnID should be cached after WsInit")
	}
}

// TestBaseController_GetConnID 测试获取连接ID
func TestBaseController_GetConnID(t *testing.T) {
	controller := &BaseController{}

	// 测试nil控制器
	var nilController *BaseController
	if nilController.GetConnID() != "" {
		t.Error("Nil controller should return empty string")
	}

	// 测试未初始化的控制器
	if controller.GetConnID() != "" {
		t.Error("Uninitialized controller should return empty string")
	}

	// 测试已初始化的控制器
	ctx := &MockContext{connID: "test-conn-id"}
	controller.WsInit(ctx)
	if controller.GetConnID() != "test-conn-id" {
		t.Error("Initialized controller should return correct connID")
	}
}

// TestBaseController_SetConnManager 测试设置连接管理器
func TestBaseController_SetConnManager(t *testing.T) {
	controller := &BaseController{}
	mockManager := NewMockConnManager()

	// 测试nil控制器
	var nilController *BaseController
	nilController.SetConnManager(mockManager)

	// 测试正常设置
	controller.SetConnManager(mockManager)

	// 测试全局默认连接管理器
	SetDefaultConnManager(mockManager)
}

// TestBaseController_BindUserID 测试绑定用户ID
func TestBaseController_BindUserID(t *testing.T) {
	controller := &BaseController{}
	mockManager := NewMockConnManager()
	controller.SetConnManager(mockManager)

	// 测试nil控制器
	var nilController *BaseController
	err := nilController.BindUserID("test-user")
	if err == nil {
		t.Error("Nil controller should return error")
	}

	// 测试空用户ID
	err = controller.BindUserID("")
	if err == nil {
		t.Error("Empty userID should return error")
	}

	// 测试非WebSocket连接
	err = controller.BindUserID("test-user")
	if err == nil {
		t.Error("Non-WebSocket connection should return error")
	}

	// 测试正常绑定
	ctx := &MockContext{connID: "test-conn-id"}
	controller.WsInit(ctx)
	err = controller.BindUserID("test-user")
	if err != nil {
		t.Errorf("Failed to bind userID: %v", err)
	}
}

// TestBaseController_UnbindUserID 测试解绑用户ID
func TestBaseController_UnbindUserID(t *testing.T) {
	controller := &BaseController{}
	mockManager := NewMockConnManager()
	controller.SetConnManager(mockManager)

	// 测试nil控制器
	var nilController *BaseController
	err := nilController.UnbindUserID("test-user")
	if err == nil {
		t.Error("Nil controller should return error")
	}

	// 测试空用户ID
	err = controller.UnbindUserID("")
	if err == nil {
		t.Error("Empty userID should return error")
	}

	// 测试非WebSocket连接
	err = controller.UnbindUserID("test-user")
	if err == nil {
		t.Error("Non-WebSocket connection should return error")
	}
}

// TestBaseController_GetUserConnIDs 测试获取用户连接ID
func TestBaseController_GetUserConnIDs(t *testing.T) {
	controller := &BaseController{}

	// 测试nil控制器
	var nilController *BaseController
	_, err := nilController.GetUserConnIDs("test-user")
	if err == nil {
		t.Error("Nil controller should return error")
	}

	// 测试空用户ID
	_, err = controller.GetUserConnIDs("")
	if err == nil {
		t.Error("Empty userID should return error")
	}

	// 测试正常获取
	ctx := &MockContext{connID: "test-conn-id"}
	controller.WsInit(ctx)
	mockManager := NewMockConnManager()
	// 添加连接到连接管理器
	mockManager.conns["test-conn-id"] = &MockConn{}
	controller.SetConnManager(mockManager)

	err = controller.BindUserID("test-user")
	if err != nil {
		t.Errorf("Failed to bind userID: %v", err)
	}

	connIDs, err := controller.GetUserConnIDs("test-user")
	if err != nil {
		t.Errorf("Failed to get user connIDs: %v", err)
	}
	if len(connIDs) == 0 {
		t.Error("Should return at least one connID")
	}
}

// MockConn 模拟WebSocket连接
type MockConn struct{}

// TestBaseController_ResponseMethods 测试响应方法
func TestBaseController_ResponseMethods(t *testing.T) {
	controller := &BaseController{}
	ctx := &MockContext{}
	controller.Init(ctx)

	// 测试nil控制器
	var nilController *BaseController
	nilController.Success("test data")
	nilController.DataSuccess("test data", 10)
	nilController.Error(500, "test error")
	nilController.RespText("test text")
	nilController.InternalError()

	// 测试正常控制器
	controller.Success("test data")
	controller.DataSuccess("test data", 10)
	controller.Error(500, "test error")
	controller.RespText("test text")
	controller.InternalError()
}

// TestBaseController_RequestMethods 测试请求方法
func TestBaseController_RequestMethods(t *testing.T) {
	controller := &BaseController{}
	ctx := &MockContext{}
	controller.Init(ctx)

	// 测试nil控制器
	var nilController *BaseController
	if nilController.GetQuery("test") != "" {
		t.Error("Nil controller should return empty string")
	}
	if nilController.GetPostForm("test") != "" {
		t.Error("Nil controller should return empty string")
	}
	if nilController.GetPostFormAll() != nil {
		t.Error("Nil controller should return nil")
	}
	if _, err := nilController.GetBody(); err != nil {
		t.Error("Nil controller should return nil error")
	}

	// 测试正常控制器
	if controller.GetQuery("test") != "" {
		t.Error("Controller should return empty string for non-existent query")
	}
	if controller.GetPostForm("test") != "" {
		t.Error("Controller should return empty string for non-existent post form")
	}
	if controller.GetPostFormAll() != nil {
		t.Error("Controller should return nil for post form all")
	}
	if _, err := controller.GetBody(); err != nil {
		t.Error("Controller should return nil error for get body")
	}

	// 测试BindJSON
	var testData map[string]interface{}
	err := controller.BindJSON(&testData)
	if err != nil {
		t.Errorf("BindJSON should not return error: %v", err)
	}
}

// TestBaseController_SendMethods 测试消息发送方法
func TestBaseController_SendMethods(t *testing.T) {
	controller := &BaseController{}
	mockManager := NewMockConnManager()
	controller.SetConnManager(mockManager)
	SetDefaultConnManager(mockManager)

	// 测试nil控制器
	var nilController *BaseController
	err := nilController.SendToConnID("test-conn", "test-action", "test-data")
	if err == nil {
		t.Error("Nil controller should return error")
	}

	// 测试正常控制器
	err = controller.SendToConnID("test-conn", "test-action", "test-data")
	// 预期会失败，因为连接不存在

	err = controller.SendToConnIDs([]string{"test-conn"}, "test-action", "test-data")
	// 预期不会失败，因为广播会忽略不存在的连接

	err = controller.SendToUser("test-user", "test-action", "test-data")
	// 预期不会失败，因为用户不存在时会返回nil

	err = controller.SendToUsers([]string{"test-user"}, "test-action", "test-data")
	// 预期不会失败，因为用户不存在时会跳过
}
