package websocket

import (
	"sync"
	"time"
)

// ConnEvent 连接事件
type ConnEvent struct {
	EventType   ConnEventType
	ConnInfo    *ConnInfo
	CloseReason string
	TriggerTime time.Time
}

// ConnInfo 连接信息
type ConnInfo struct {
	ConnID     string
	ClientIP   string
	CreateTime time.Time
}

// ConnEventType 连接事件类型
type ConnEventType string

// ConnectionEventHandler 连接事件处理器接口
type ConnectionEventHandler interface {
	// OnConnect 连接建立时触发
	OnConnect(connInfo *ConnInfo)
	// OnDisconnect 连接关闭时触发
	OnDisconnect(connInfo *ConnInfo, reason string)
}

// ConnectionEventManager 连接事件管理器
type ConnectionEventManager struct {
	handlers []ConnectionEventHandler
	mu       sync.RWMutex
}

const (
	EventConnOnline  ConnEventType = "online"
	EventConnOffline ConnEventType = "offline"
)

// NewConnectionEventManager 创建事件管理器
func NewConnectionEventManager() *ConnectionEventManager {
	return &ConnectionEventManager{
		handlers: make([]ConnectionEventHandler, 0),
	}
}

// RegisterHandler 注册事件处理器
func (em *ConnectionEventManager) RegisterHandler(handler ConnectionEventHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.handlers = append(em.handlers, handler)
}

// NotifyConnect 通知连接建立事件
func (em *ConnectionEventManager) NotifyConnect(connInfo *ConnInfo) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	for _, handler := range em.handlers {
		handler.OnConnect(connInfo)
	}
}

// NotifyDisconnect 通知连接关闭事件
func (em *ConnectionEventManager) NotifyDisconnect(connInfo *ConnInfo, reason string) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	for _, handler := range em.handlers {
		handler.OnDisconnect(connInfo, reason)
	}
}
