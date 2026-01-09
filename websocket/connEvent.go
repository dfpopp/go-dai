package websocket

import (
	"sync"
	"time"
)

// 事件类型常量
const (
	EventConnOnline  = "websocket.conn.online"  // 连接上线事件
	EventConnOffline = "websocket.conn.offline" // 连接下线事件
)

// ConnEvent 连接事件结构体（携带完整事件信息）
type ConnEvent struct {
	EventType   string    // 事件类型
	ConnInfo    *ConnInfo // 连接详情
	TriggerTime time.Time // 事件触发时间
	CloseReason string    // 下线原因（仅离线事件有效）
}

// ConnEventListener 应用层事件监听器接口（应用层需实现该接口）
type ConnEventListener interface {
	OnConnEvent(event ConnEvent) // 事件回调方法
}

// ConnEventBus 事件总线（负责订阅、取消订阅、发布事件）
type ConnEventBus struct {
	listeners sync.Map // key: 监听器唯一ID, value: ConnEventListener
}

// NewConnEventBus 创建事件总线实例
func NewConnEventBus() *ConnEventBus {
	return &ConnEventBus{}
}

// Subscribe 订阅事件（应用层调用）
func (eb *ConnEventBus) Subscribe(listenerID string, listener ConnEventListener) {
	if listener == nil {
		return
	}
	eb.listeners.Store(listenerID, listener)
}

// Unsubscribe 取消订阅事件（应用层调用）
func (eb *ConnEventBus) Unsubscribe(listenerID string) {
	eb.listeners.Delete(listenerID)
}

// Publish 发布事件（框架内部调用）
func (eb *ConnEventBus) Publish(event ConnEvent) {
	eb.listeners.Range(func(_, value interface{}) bool {
		listener, ok := value.(ConnEventListener)
		if ok {
			// 异步执行，避免阻塞框架逻辑
			go listener.OnConnEvent(event)
		}
		return true
	})
}
