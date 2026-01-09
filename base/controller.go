package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/grpc"
	"github.com/dfpopp/go-dai/http"
	"github.com/dfpopp/go-dai/logger"
	"github.com/dfpopp/go-dai/netContext"
	"github.com/dfpopp/go-dai/websocket"
	"github.com/google/uuid"
	"sync"
)

// BaseController 框架根控制器基类
type BaseController struct {
	Ctx          netContext.Context     // 注入HTTP上下文
	connManager  *websocket.ConnManager // 连接管理器（全局单例）
	cachedConnID string                 // 缓存当前连接ID，避免重复断言
	UserIDField  string                 // 用户ID在连接属性中的存储键（默认"user_id"）
	userConnMap  sync.Map               //维护用户ID -> 连接ID列表的映射（无需应用层额外维护）
	Log          logger.Logger          // 日志实例
}

// Init 初始化控制器（框架自动调用，注入上下文）
func (c *BaseController) Init(ctx netContext.Context) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return
	}
	c.Ctx = ctx
	c.Log = logger.GetLogger()
	// 兼容WS和HTTP的路径/action打印
	if c.Log.GetEnv() != "prod" {
		if httpCtx, ok := ctx.(*http.Context); ok {
			c.Log.Info("控制器初始化：", httpCtx.Req.URL.Path)
		} else if wsCtx, ok := ctx.(*websocket.Context); ok {
			c.Log.Info("控制器初始化：", wsCtx.Action)
		} else if grpcCtx, ok := ctx.(*grpc.Context); ok {

			c.Log.Info("控制器初始化：", grpcCtx.GetPath())
		}
	}
}

// WsInit WS专属初始化控制器（框架自动调用，注入上下文）
func (c *BaseController) WsInit(ctx netContext.Context) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return
	}
	c.Ctx = ctx
	c.Log = logger.GetLogger()
	// 新增：初始化连接管理器和用户ID字段
	c.connManager = websocket.GetGlobalConnManager()
	if c.UserIDField == "" {
		c.UserIDField = "user_id"
	}
	// 新增：初始化并缓存ConnID
	c.initCachedConnID()
	// 兼容WS和HTTP的路径/action打印
	if c.Log.GetEnv() != "prod" {
		if httpCtx, ok := ctx.(*http.Context); ok {
			c.Log.Info("控制器初始化：", httpCtx.Req.URL.Path)
		} else if wsCtx, ok := ctx.(*websocket.Context); ok {
			c.Log.Info("控制器初始化：", wsCtx.Action)
		} else if grpcCtx, ok := ctx.(*grpc.Context); ok {
			c.Log.Info("控制器初始化：", grpcCtx.GetPath())
		}
	}
}

// -------------------------- 新增：内部辅助方法 --------------------------
// initCachedConnID 初始化并缓存当前连接ID（内部方法，避免重复类型断言）
func (c *BaseController) initCachedConnID() {
	if c.Ctx == nil {
		c.cachedConnID = ""
		logger.Warn("BaseController 上下文未注入，无法获取ConnID")
		return
	}
	// 兼容两种WS上下文类型
	switch ctx := c.Ctx.(type) {
	case *websocket.Context:
		c.cachedConnID = ctx.GetConnID()
	case websocket.ConnIDContext:
		c.cachedConnID = ctx.GetConnID()
	default:
		// 非WS上下文（HTTP/GRPC），ConnID置空
		c.cachedConnID = ""
	}
}

// GetConnID 获取当前连接ID（应用层直接调用）
func (c *BaseController) GetConnID() string {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法获取ConnID")
		return ""
	}
	// 若缓存为空，重新尝试初始化
	if c.cachedConnID == "" {
		c.initCachedConnID()
	}
	return c.cachedConnID
}

// BindUserID 绑定当前连接与用户ID（应用层直接调用）
func (c *BaseController) BindUserID(userID string) error {
	if c == nil {
		return errors.New("BaseController 未初始化（指针为nil），无法绑定用户ID")
	}
	if userID == "" {
		return errors.New("用户ID不能为空")
	}
	connID := c.GetConnID()
	if connID == "" {
		return errors.New("非WebSocket连接，无法绑定用户ID")
	}
	if c.connManager == nil {
		return errors.New("连接管理器未初始化，无法绑定用户ID")
	}
	// 存储用户ID到连接属性
	c.connManager.SetConnAttr(connID, c.UserIDField, userID)
	connIDsObj, exists := c.userConnMap.Load(userID)
	var connIDs []string
	if exists {
		connIDs, _ = connIDsObj.([]string)
	}
	// 去重：避免重复绑定同一连接
	for _, cid := range connIDs {
		if cid == connID {
			c.Log.Info("用户ID与连接已绑定，无需重复操作", "userID", userID, "connID", connID)
			return nil
		}
	}
	connIDs = append(connIDs, connID)
	c.userConnMap.Store(userID, connIDs)
	if c.Log.GetEnv() != "prod" {
		c.Log.Info("用户ID与连接绑定成功", "userID", userID, "connID", connID)
	}
	return nil
}

// UnbindUserID 解绑当前连接与用户ID（应用层直接调用，如下线时）
func (c *BaseController) UnbindUserID(userID string) error {
	if c == nil {
		return errors.New("BaseController 未初始化（指针为nil），无法解绑用户ID")
	}
	if userID == "" {
		return errors.New("用户ID不能为空")
	}
	connID := c.GetConnID()
	if connID == "" {
		return errors.New("非WebSocket连接，无法解绑用户ID")
	}

	// 1. 移除连接属性中的用户ID
	if c.connManager != nil {
		c.connManager.SetConnAttr(connID, c.UserIDField, "")
	}

	// 2. 维护用户-连接映射，移除当前连接
	connIDsObj, exists := c.userConnMap.Load(userID)
	if !exists {
		return nil
	}
	connIDs, ok := connIDsObj.([]string)
	if !ok {
		return errors.New("用户连接列表格式错误")
	}
	var newConnIDs []string
	for _, cid := range connIDs {
		if cid != connID {
			newConnIDs = append(newConnIDs, cid)
		}
	}
	if len(newConnIDs) == 0 {
		c.userConnMap.Delete(userID)
	} else {
		c.userConnMap.Store(userID, newConnIDs)
	}
	if c.Log.GetEnv() != "prod" {
		c.Log.Info("用户ID与连接解绑成功", "userID", userID, "connID", connID)
	}
	return nil
}

// GetUserConnIDs 根据用户ID获取所有在线连接ID（应用层直接调用，无需额外实现）
func (c *BaseController) GetUserConnIDs(userID string) ([]string, error) {
	if c == nil {
		return nil, errors.New("BaseController 未初始化（指针为nil），无法获取用户连接ID")
	}
	if userID == "" {
		return nil, errors.New("用户ID不能为空")
	}

	// 1. 从用户-连接映射中获取连接列表
	connIDsObj, exists := c.userConnMap.Load(userID)
	if !exists {
		return []string{}, nil
	}
	connIDs, ok := connIDsObj.([]string)
	if !ok {
		return nil, errors.New("用户连接列表格式错误")
	}

	// 2. 过滤无效连接（可选：校验连接是否仍在线）
	var validConnIDs []string
	if c.connManager != nil {
		for _, connID := range connIDs {
			if _, ok := c.connManager.GetConnByConnID(connID); ok {
				validConnIDs = append(validConnIDs, connID)
			}
		}
	} else {
		validConnIDs = connIDs
	}

	// 3. 更新有效连接映射（避免无效数据堆积）
	if len(validConnIDs) != len(connIDs) {
		if len(validConnIDs) == 0 {
			c.userConnMap.Delete(userID)
		} else {
			c.userConnMap.Store(userID, validConnIDs)
		}
	}
	if c.Log.GetEnv() != "prod" {
		c.Log.Info("获取用户在线连接ID成功", "userID", userID, "connCount", len(validConnIDs))
	}
	return validConnIDs, nil
}

// buildStandardMsg 组装标准WS消息格式（内部方法，复用消息构建逻辑）
func (c *BaseController) buildStandardMsg(action string, data interface{}) string {
	if action == "" || data == nil {
		logger.Warn("消息动作或内容不能为空，无法组装消息")
		return ""
	}
	msg := map[string]interface{}{
		"action":     action,
		"request_id": uuid.NewString(),
		"data":       data,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		c.Log.Error("标准消息序列化失败", "error", err, "action", action)
		return ""
	}
	return string(msgBytes)
}

// SendToConnID 给指定单个连接ID发送消息（应用层直接调用）
func (c *BaseController) SendToConnID(targetConnID string, msgAction string, msgData interface{}) error {
	if c == nil {
		return errors.New("BaseController 未初始化（指针为nil），无法发送消息")
	}
	if targetConnID == "" || msgAction == "" {
		return errors.New("目标连接ID和消息动作不能为空")
	}
	if c.connManager == nil {
		return errors.New("连接管理器未初始化，无法发送消息")
	}
	// 组装标准消息
	msgStr := c.buildStandardMsg(msgAction, msgData)
	if msgStr == "" {
		return errors.New("消息组装失败，无法发送")
	}
	// 发送消息
	err := c.connManager.SendToConnID(targetConnID, msgStr)
	if err != nil {
		c.Log.Error("给指定连接发送消息失败", "targetConnID", targetConnID, "error", err)
		return err
	}
	if c.Log.GetEnv() != "prod" {
		c.Log.Info("给指定连接发送消息成功", "targetConnID", targetConnID, "msgAction", msgAction)
	}
	return nil
}

// SendToConnIDs 给指定多个连接ID批量发送消息（应用层直接调用）
func (c *BaseController) SendToConnIDs(targetConnIDs []string, msgAction string, msgData interface{}) error {
	if c == nil {
		return errors.New("BaseController 未初始化（指针为nil），无法发送消息")
	}
	if len(targetConnIDs) == 0 || msgAction == "" {
		return errors.New("目标连接ID列表和消息动作不能为空")
	}
	if c.connManager == nil {
		return errors.New("连接管理器未初始化，无法发送消息")
	}
	// 组装标准消息
	msgStr := c.buildStandardMsg(msgAction, msgData)
	if msgStr == "" {
		return errors.New("消息组装失败，无法发送")
	}
	// 批量发送消息
	c.connManager.Multicast(targetConnIDs, msgStr)
	if c.Log.GetEnv() != "prod" {
		c.Log.Info("批量给多个连接发送消息成功", "connCount", len(targetConnIDs), "msgAction", msgAction)
	}
	return nil
}

// SendToUser 给指定单个用户发送消息（自动获取该用户所有在线连接，应用层直接调用）
func (c *BaseController) SendToUser(targetUserID string, msgAction string, msgData interface{}) error {
	if c == nil {
		return errors.New("BaseController 未初始化（指针为nil），无法发送消息")
	}
	if targetUserID == "" || msgAction == "" {
		return errors.New("目标用户ID和消息动作不能为空")
	}
	if c.connManager == nil {
		return errors.New("连接管理器未初始化，无法发送消息")
	}

	// 1. 调用基类GetUserConnIDs获取用户在线连接
	targetConnIDs, err := c.GetUserConnIDs(targetUserID)
	if err != nil {
		return fmt.Errorf("获取目标用户在线连接失败：%w", err)
	}
	if len(targetConnIDs) == 0 {
		if c.Log.GetEnv() != "prod" {
			c.Log.Warn("目标用户无在线连接，无需发送消息", "targetUserID", targetUserID)
		}
		return nil
	}

	// 2. 组装标准消息
	msgStr := c.buildStandardMsg(msgAction, msgData)
	if msgStr == "" {
		return errors.New("消息组装失败，无法发送")
	}

	// 3. 给该用户的所有在线连接发送消息
	c.connManager.Multicast(targetConnIDs, msgStr)
	if c.Log.GetEnv() != "prod" {
		c.Log.Info("给指定用户发送消息成功", "targetUserID", targetUserID, "connCount", len(targetConnIDs), "msgAction", msgAction)
	}
	return nil
}

// SendToUsers 给指定多个用户批量发送消息（自动获取每个用户的在线连接，应用层直接调用）
func (c *BaseController) SendToUsers(targetUserIDs []string, msgAction string, msgData interface{}) error {
	if c == nil {
		return errors.New("BaseController 未初始化（指针为nil），无法发送消息")
	}
	if len(targetUserIDs) == 0 || msgAction == "" {
		return errors.New("目标用户ID列表和消息动作不能为空")
	}
	if c.connManager == nil {
		return errors.New("连接管理器未初始化，无法发送消息")
	}

	// 1. 批量收集所有目标用户的在线连接ID
	var allTargetConnIDs []string
	for _, userID := range targetUserIDs {
		connIDs, err := c.GetUserConnIDs(userID)
		if err != nil {
			if c.Log.GetEnv() != "prod" {
				c.Log.Warn("获取用户在线连接失败，跳过该用户", "userID", userID, "error", err)
			}
			continue
		}
		allTargetConnIDs = append(allTargetConnIDs, connIDs...)
	}
	if len(allTargetConnIDs) == 0 {
		if c.Log.GetEnv() != "prod" {
			c.Log.Warn("所有目标用户均无在线连接，无需发送消息", "userCount", len(targetUserIDs))
		}
		return nil
	}

	// 2. 组装标准消息
	msgStr := c.buildStandardMsg(msgAction, msgData)
	if msgStr == "" {
		return errors.New("消息组装失败，无法发送")
	}

	// 3. 批量发送消息
	c.connManager.Multicast(allTargetConnIDs, msgStr)
	if c.Log.GetEnv() != "prod" {
		c.Log.Info("批量给多个用户发送消息成功", "userCount", len(targetUserIDs), "connCount", len(allTargetConnIDs), "msgAction", msgAction)
	}
	return nil
}

// Success 统一成功响应（JSON格式）
func (c *BaseController) Success(data interface{}, msg ...string) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return
	}
	if c.Ctx == nil {
		logger.Error("调用框架BaseController.Success 之前未设置上下文")
		return
	}
	message := "操作成功"
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	c.Ctx.JSON(200, map[string]interface{}{
		"code": 200,
		"msg":  message,
		"data": data,
	})
}

// DataSuccess 统一成功响应（JSON格式）
func (c *BaseController) DataSuccess(data interface{}, count int64) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return
	}
	if c.Ctx == nil {
		logger.Error("调用框架BaseController.DataSuccess 之前未设置上下文")
		return
	}
	c.Ctx.JSON(200, map[string]interface{}{
		"code":  200,
		"msg":   "操作成功",
		"data":  data,
		"count": count,
	})
}

// Error 统一失败响应（JSON格式）
func (c *BaseController) Error(code int, msg string) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return
	}
	if c.Ctx == nil {
		logger.Error("调用框架BaseController.Error 之前未设置上下文")
		return
	}
	c.Ctx.JSON(200, map[string]interface{}{
		"code": code,
		"msg":  msg,
		"data": nil,
	})
	reqInfo := c.Ctx.GetRequestInfo()
	if c.Log.GetEnv() != "prod" {
		c.Log.Error("接口响应失败：", "code=", code, "msg=", msg, "path=", reqInfo.GetPath())
	}
}

// RespText 统一响应（字符串格式）
func (c *BaseController) RespText(msg string) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return
	}
	if c.Ctx == nil {
		logger.Error("调用框架BaseController.Success 之前未设置上下文")
		return
	}
	c.Ctx.String(200, msg)
}

// InternalError 服务器内部错误响应
func (c *BaseController) InternalError() {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return
	}
	c.Error(500, "服务器内部错误")
}

// GetQuery 获取URL查询参数
func (c *BaseController) GetQuery(key string) string {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return ""
	}
	if c.Ctx == nil {
		logger.Error("调用框架BaseController.GetQuery 之前未设置上下文")
		return ""
	}
	return c.Ctx.Query(key)
}

// GetPostForm 获取POST表单参数
func (c *BaseController) GetPostForm(key string) string {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return ""
	}
	if c.Ctx == nil {
		logger.Error("调用框架BaseController.GetQuery 之前未设置上下文")
		return ""
	}
	return c.Ctx.PostForm(key)
}
func (c *BaseController) GetPostFormAll() map[string]string {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return nil
	}
	if c.Ctx == nil {
		logger.Error("调用框架BaseController.GetQuery 之前未设置上下文")
		return nil
	}
	return c.Ctx.PostFormAll()
}
func (c *BaseController) GetBody() ([]byte, error) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Error响应")
		return nil, nil
	}
	if c.Ctx == nil {
		logger.Error("调用框架BaseController.GetQuery 之前未设置上下文")
		return nil, nil
	}
	return c.Ctx.GetBody()
}

// BindJSON 绑定JSON请求体到结构体
func (c *BaseController) BindJSON(v interface{}) error {
	if c == nil {
		return errors.New("BaseController 未初始化（指针为nil），无法执行Error响应")
	}
	if c.Ctx == nil {
		return errors.New("调用框架BaseController.GetQuery 之前未设置上下文")
	}
	return c.Ctx.BindJSON(v)
}
