package websocket

import (
	"errors"
	"github.com/dfpopp/go-dai/logger"
	"github.com/google/uuid"
	"sync"
	"time"
)

// ConnInfo 连接信息结构体
type ConnInfo struct {
	Conn     *Conn     // WS连接实例
	ConnID   string    // 唯一连接ID
	ClientIP string    // 客户端IP
	CreateAt time.Time // 连接创建时间
	attrs    sync.Map  // 应用层自定义属性（如用户ID）
}

// ConnManager 连接管理器（单例）
type ConnManager struct {
	connMap  sync.Map      // key: ConnID, value: *ConnInfo
	eventBus *ConnEventBus // 事件总线
}

// 全局连接管理器实例
var globalConnManager = NewConnManager()

// NewConnManager 创建连接管理器
func NewConnManager() *ConnManager {
	return &ConnManager{
		eventBus: NewConnEventBus(),
	}
}

// GetGlobalConnManager 获取全局连接管理器（应用层/框架层调用）
func GetGlobalConnManager() *ConnManager {
	return globalConnManager
}

// GetEventBus 获取事件总线（应用层用于订阅事件）
func (cm *ConnManager) GetEventBus() *ConnEventBus {
	return cm.eventBus
}

// AddConn 添加连接（触发上线事件）
func (cm *ConnManager) AddConn(conn *Conn, clientIP string) *ConnInfo {
	connID := uuid.NewString()
	connInfo := &ConnInfo{
		Conn:     conn,
		ConnID:   connID,
		ClientIP: clientIP,
		CreateAt: time.Now(),
	}
	cm.connMap.Store(connID, connInfo)
	logger.Info("WS连接上线", "connID", connID, "clientIP", clientIP, "totalConn", cm.GetConnCount())

	// 发布上线事件
	cm.eventBus.Publish(ConnEvent{
		EventType:   EventConnOnline,
		ConnInfo:    connInfo,
		TriggerTime: time.Now(),
		CloseReason: "",
	})

	return connInfo
}

// RemoveConn 移除连接（触发下线事件）
func (cm *ConnManager) RemoveConn(connID string, closeReason string) {
	connInfo, exists := cm.connMap.LoadAndDelete(connID)
	if !exists {
		return
	}
	info := connInfo.(*ConnInfo)
	logger.Info("WS连接下线", "connID", connID, "clientIP", info.ClientIP, "reason", closeReason, "totalConn", cm.GetConnCount())

	// 发布下线事件
	cm.eventBus.Publish(ConnEvent{
		EventType:   EventConnOffline,
		ConnInfo:    info,
		TriggerTime: time.Now(),
		CloseReason: closeReason,
	})

	_ = info.Conn.Close()
}

// GetConnByConnID 根据ConnID获取连接实例（应用层调用）
func (cm *ConnManager) GetConnByConnID(connID string) (*Conn, bool) {
	connInfo, exists := cm.connMap.Load(connID)
	if !exists {
		return nil, false
	}
	return connInfo.(*ConnInfo).Conn, true
}

// GetConnInfoByConnID 根据ConnID获取完整连接信息（应用层调用）
func (cm *ConnManager) GetConnInfoByConnID(connID string) (*ConnInfo, bool) {
	connInfo, exists := cm.connMap.Load(connID)
	if !exists {
		return nil, false
	}
	return connInfo.(*ConnInfo), true
}

// GetConnCount 获取当前连接总数
func (cm *ConnManager) GetConnCount() int {
	count := 0
	cm.connMap.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// Broadcast 群发消息（应用层调用）
func (cm *ConnManager) Broadcast(message string) {
	cm.connMap.Range(func(_, value interface{}) bool {
		connInfo := value.(*ConnInfo)
		_ = connInfo.Conn.WriteMessage(message)
		return true
	})
}

// Multicast 定向群发消息（应用层调用）
func (cm *ConnManager) Multicast(connIDs []string, message string) {
	for _, connID := range connIDs {
		if connInfo, exists := cm.connMap.Load(connID); exists {
			info := connInfo.(*ConnInfo)
			_ = info.Conn.WriteMessage(message)
		} else {
			logger.Warn("定向群发失败：连接不存在", "connID", connID)
		}
	}
}

// SendToConnID 给单个ConnID发送消息（应用层调用）
func (cm *ConnManager) SendToConnID(connID string, message string) error {
	connInfo, exists := cm.connMap.Load(connID)
	if !exists {
		return errors.New("connection not found: " + connID)
	}
	info := connInfo.(*ConnInfo)
	return info.Conn.WriteMessage(message)
}

// SetConnAttr 设置连接自定义属性（应用层调用）
func (cm *ConnManager) SetConnAttr(connID string, key string, value interface{}) {
	if connInfo, exists := cm.connMap.Load(connID); exists {
		info := connInfo.(*ConnInfo)
		info.attrs.Store(key, value)
	}
}

// GetConnAttr 获取连接自定义属性（应用层调用）
func (cm *ConnManager) GetConnAttr(connID string, key string) (interface{}, bool) {
	if connInfo, exists := cm.connMap.Load(connID); exists {
		info := connInfo.(*ConnInfo)
		return info.attrs.Load(key)
	}
	return nil, false
}

// CloseConnByConnID 主动关闭指定连接（应用层调用，触发下线事件）
func (cm *ConnManager) CloseConnByConnID(connID string, closeReason string) {
	cm.RemoveConn(connID, closeReason)
}
