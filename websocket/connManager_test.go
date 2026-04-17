package websocket

import (
	"testing"
)

// TestNewConnManager 测试创建连接管理器
func TestNewConnManager(t *testing.T) {
	// 测试创建连接管理器
	manager := NewConnManager()
	if manager == nil {
		t.Error("NewConnManager should return non-nil manager")
	}

	// 测试获取全局连接管理器
	globalManager := GetGlobalConnManager()
	if globalManager == nil {
		t.Error("GetGlobalConnManager should return non-nil manager")
	}
}

// TestConnManager_Methods 测试连接管理器方法
func TestConnManager_Methods(t *testing.T) {
	// 创建测试连接管理器
	manager := NewConnManager()

	// 测试添加连接
	mockConn := &Conn{}
	connInfo := manager.AddConn(mockConn, "127.0.0.1")
	if connInfo == nil {
		t.Error("AddConn should return non-nil connInfo")
	}

	// 测试获取连接
	conn, ok := manager.GetConnByConnID(connInfo.ConnID)
	if !ok {
		t.Error("GetConnByConnID should find the added conn")
	}
	if conn != mockConn {
		t.Error("GetConnByConnID should return the added conn")
	}

	// 测试设置连接属性
	manager.SetConnAttr(connInfo.ConnID, "test_key", "test_value")

	// 测试获取连接属性
	value, ok := manager.GetConnAttr(connInfo.ConnID, "test_key")
	if !ok {
		t.Error("GetConnAttr should find the set attr")
	}
	if value != "test_value" {
		t.Error("GetConnAttr should return the correct value")
	}

	// 测试发送消息到连接 - 注释掉，因为mock连接的conn字段为nil会导致空指针错误
	// _ = manager.SendToConnID(connInfo.ConnID, "test message")
	// 预期会失败，因为连接是mock的，没有实际的网络连接

	// 测试批量发送消息
	// 由于ConnManager没有提供SendToConnIDs方法，这里注释掉
	// 批量发送消息不会失败，即使连接是mock的

	// 测试移除连接
	manager.RemoveConn(connInfo.ConnID, "test reason")

	// 测试检查连接是否存在
	if manager.IsConnAlive(connInfo.ConnID) {
		t.Error("IsConnAlive should return false for removed conn")
	}

	// 测试获取连接数量
	count := manager.GetConnCount()
	if count != 0 {
		t.Error("GetConnCount should return 0 after removing conn")
	}

	// 测试关闭连接
	_ = manager.CloseConnByConnID(connInfo.ConnID, "test reason")
	// 预期会失败，因为连接已经被移除
}

// TestConnectionEventManager 测试连接事件管理器
func TestConnectionEventManager(t *testing.T) {
	// 创建测试连接事件管理器
	_ = NewConnectionEventManager()

	// 由于ConnectionEventManager没有暴露外部方法，这里只测试创建不报错
}
