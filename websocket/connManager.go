package websocket

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// ConnManager 连接管理器
type ConnManager struct {
	conns     map[string]*Conn
	connInfos map[string]*ConnInfo // 存储连接信息
	connAttrs map[string]map[string]interface{}
	connLock  sync.RWMutex
	attrLock  sync.RWMutex
	connCount int32
	eventMgr  *ConnectionEventManager
}

// 全局连接管理器
var (
	globalConnManager *ConnManager
	once              sync.Once
)

// NewConnManager 创建连接管理器
func NewConnManager() *ConnManager {
	return &ConnManager{
		conns:     make(map[string]*Conn),
		connInfos: make(map[string]*ConnInfo), // 初始化
		connAttrs: make(map[string]map[string]interface{}),
		connCount: 0,
		eventMgr:  NewConnectionEventManager(),
	}
}

// GetEventManager 获取事件管理器
func (m *ConnManager) GetEventManager() *ConnectionEventManager {
	return m.eventMgr
}

// GetConnByConnID 根据连接ID获取连接
func (m *ConnManager) GetConnByConnID(connID string) (interface{}, bool) {
	m.connLock.RLock()
	defer m.connLock.RUnlock()
	conn, ok := m.conns[connID]
	return conn, ok
}

// AddConn 添加连接
func (m *ConnManager) AddConn(conn *Conn, clientIP string) *ConnInfo {
	connID := generateConnID()
	connInfo := &ConnInfo{
		ConnID:     connID,
		ClientIP:   clientIP,
		CreateTime: time.Now(),
	}
	m.connLock.Lock()
	m.conns[connID] = conn
	m.connInfos[connID] = connInfo // 存储连接信息
	m.connLock.Unlock()

	m.attrLock.Lock()
	m.connAttrs[connID] = make(map[string]interface{})
	m.attrLock.Unlock()

	atomic.AddInt32(&m.connCount, 1)

	// 新增：通知连接建立事件
	m.eventMgr.NotifyConnect(connInfo)

	return connInfo
}

// RemoveConn 移除连接
func (m *ConnManager) RemoveConn(connID, reason string) {
	m.connLock.Lock()
	connInfo := m.connInfos[connID] // 获取连接信息
	delete(m.conns, connID)
	delete(m.connInfos, connID) // 移除连接信息
	m.connLock.Unlock()

	m.attrLock.Lock()
	delete(m.connAttrs, connID)
	m.attrLock.Unlock()

	atomic.AddInt32(&m.connCount, -1)

	// 新增：通知连接关闭事件
	// 通知连接关闭事件
	if connInfo != nil {
		m.eventMgr.NotifyDisconnect(connInfo, reason)
	}
}

// SetConnAttr 设置连接属性
func (m *ConnManager) SetConnAttr(connID, key string, value interface{}) {
	m.attrLock.Lock()
	defer m.attrLock.Unlock()

	if attrs, ok := m.connAttrs[connID]; ok {
		attrs[key] = value
	}
}

// GetConnAttr 获取连接属性
func (m *ConnManager) GetConnAttr(connID, key string) (interface{}, bool) {
	m.attrLock.RLock()
	defer m.attrLock.RUnlock()

	if attrs, ok := m.connAttrs[connID]; ok {
		value, ok := attrs[key]
		return value, ok
	}
	return nil, false
}

// SendToConnID 给指定连接发送消息
func (m *ConnManager) SendToConnID(connID, message string) error {
	m.connLock.RLock()
	conn, ok := m.conns[connID]
	m.connLock.RUnlock()

	if !ok {
		return ErrConnNotFound
	}

	return conn.WriteMessage(message)
}

// Multicast 给多个连接发送消息
func (m *ConnManager) Multicast(connIDs []string, message string) {
	for _, connID := range connIDs {
		if err := m.SendToConnID(connID, message); err != nil {
			// 连接不存在或已关闭，忽略错误
		}
	}
}

// GetConnCount 获取连接数
func (m *ConnManager) GetConnCount() int32 {
	return atomic.LoadInt32(&m.connCount)
}

// IsConnAlive 检查连接是否存活
func (m *ConnManager) IsConnAlive(connID string) bool {
	m.connLock.RLock()
	defer m.connLock.RUnlock()

	_, ok := m.conns[connID]
	return ok
}

// CloseConnByConnID 主动(手动)关闭指定连接
// 示例：err := websocket.GetGlobalConnManager().CloseConnByConnID(connID, "手动关闭")
func (m *ConnManager) CloseConnByConnID(connID, reason string) error {
	m.connLock.Lock()
	conn, ok := m.conns[connID]
	if !ok {
		m.connLock.Unlock()
		return ErrConnNotFound
	}
	// 先从管理器中移除
	connInfo := m.connInfos[connID]
	delete(m.conns, connID)
	delete(m.connInfos, connID)
	m.connLock.Unlock()

	// 移除连接属性
	m.attrLock.Lock()
	delete(m.connAttrs, connID)
	m.attrLock.Unlock()

	// 减少连接计数
	atomic.AddInt32(&m.connCount, -1)

	// 通知连接关闭事件
	if connInfo != nil {
		m.eventMgr.NotifyDisconnect(connInfo, reason)
	}

	// 关闭底层连接
	return conn.Close()
}

// GetGlobalConnManager 获取全局连接管理器（延迟初始化）
func GetGlobalConnManager() *ConnManager {
	once.Do(func() {
		globalConnManager = NewConnManager()
	})
	return globalConnManager
}

// generateConnID 生成唯一的连接ID
func generateConnID() string {
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
