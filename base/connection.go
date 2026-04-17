package base

// ConnManager 连接管理接口
type ConnManager interface {
	// 连接操作

	GetConnByConnID(connID string) (interface{}, bool)
	RemoveConn(connID, reason string)
	CloseConnByConnID(connID, reason string) error

	// 属性操作

	SetConnAttr(connID, key string, value interface{})
	GetConnAttr(connID, key string) (interface{}, bool)
	RemoveConnAttr(connID, key string)

	// 消息发送

	SendToConnID(connID, message string) error
	Multicast(connIDs []string, message string)
	Broadcast(message string)

	// 连接状态

	GetConnCount() int32
	IsConnAlive(connID string) bool
	GetAllConnIDs() []string

	// 连接查询

	GetConnIDsByAttr(key string, value interface{}) []string
}
