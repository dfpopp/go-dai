package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/function"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/dfpopp/go-dai/grpc"
	"github.com/dfpopp/go-dai/http"
	"github.com/dfpopp/go-dai/logger"
	"github.com/dfpopp/go-dai/netContext"
	"github.com/dfpopp/go-dai/websocket"
	"github.com/google/uuid"
)

// goroutineContext 存储每个goroutine的上下文信息
type goroutineContext struct {
	ctx          netContext.Context
	cachedConnID string
	userConnMap  sync.Map
}

// BaseController 框架根控制器基类
type BaseController struct {
	BaseComponent
	connManager  ConnManager // 连接管理器（使用接口，提高可扩展性）
	UserIDField  string      // 用户ID在连接属性中的存储键（默认"user_id"）
	goroutineCtx sync.Map    // 存储每个goroutine的上下文信息
}

// NewBaseController 创建新的控制器实例
func NewBaseController() *BaseController {
	return &BaseController{
		UserIDField:  "user_id",
		goroutineCtx: sync.Map{},
	}
}

// getGoroutineID 获取当前goroutine的ID
func getGoroutineID() uint64 {
	// 使用goroutine的程序计数器作为唯一标识
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, _ := strconv.ParseUint(idField, 10, 64)
	return id
}

// getGoroutineContext 获取当前goroutine的上下文信息
func (c *BaseController) getGoroutineContext() *goroutineContext {
	if c == nil {
		return nil
	}
	// 使用goroutine ID作为key
	key := getGoroutineID()
	ctx, ok := c.goroutineCtx.Load(key)
	if !ok {
		// 创建新的goroutine上下文
		newCtx := &goroutineContext{
			userConnMap: sync.Map{},
		}
		c.goroutineCtx.Store(key, newCtx)
		return newCtx
	}
	return ctx.(*goroutineContext)
}

// Init 初始化控制器（框架自动调用，注入上下文）
func (c *BaseController) Init(ctx netContext.Context) {
	if c == nil {
		return
	}
	c.BaseComponent.Init()
	// 初始化UserIDField默认值
	if c.UserIDField == "" {
		c.UserIDField = "user_id"
	}
	// 获取当前goroutine的上下文
	grCtx := c.getGoroutineContext()
	if grCtx != nil {
		grCtx.ctx = ctx
		grCtx.cachedConnID = ""
		grCtx.userConnMap = sync.Map{}
	}
	// 注册清理函数
	ctx.RegisterCleanupFunc(func() {
		c.Cleanup()
	})
	// 兼容WS和HTTP的路径/action打印
	if c.log != nil && c.log.GetEnv() != "prod" {
		if httpCtx, ok := ctx.(*http.Context); ok {
			c.LogInfo("控制器初始化：", httpCtx.Req.URL.Path)
		} else if wsCtx, ok := ctx.(*websocket.Context); ok {
			c.LogInfo("控制器初始化：", wsCtx.Action)
		} else if grpcCtx, ok := ctx.(*grpc.Context); ok {
			c.LogInfo("控制器初始化：", grpcCtx.GetPath())
		}
	}
}

// SetConnManager 设置连接管理器（框架或应用层调用）
func (c *BaseController) SetConnManager(manager ConnManager) {
	if c == nil {
		return
	}
	c.connManager = manager
}

// WsInit WS专属初始化控制器（框架自动调用，注入上下文）
func (c *BaseController) WsInit(ctx netContext.Context) {
	if c == nil {
		return
	}
	c.BaseComponent.Init()
	// 初始化用户ID字段
	if c.UserIDField == "" {
		c.UserIDField = "user_id"
	}
	// 获取当前goroutine的上下文
	grCtx := c.getGoroutineContext()
	if grCtx != nil {
		grCtx.ctx = ctx
		grCtx.cachedConnID = ""
		grCtx.userConnMap = sync.Map{}
	}
	// 注册清理函数
	ctx.RegisterCleanupFunc(func() {
		c.Cleanup()
	})
	// 初始化并缓存ConnID
	c.initCachedConnID()
	// 兼容WS和HTTP的路径/action打印
	if c.log != nil && c.log.GetEnv() != "prod" {
		if httpCtx, ok := ctx.(*http.Context); ok {
			c.LogInfo("控制器初始化：", httpCtx.Req.URL.Path)
		} else if wsCtx, ok := ctx.(*websocket.Context); ok {
			c.LogInfo("控制器初始化：", wsCtx.Action)
		} else if grpcCtx, ok := ctx.(*grpc.Context); ok {
			c.LogInfo("控制器初始化：", grpcCtx.GetPath())
		}
	}
}

// -------------------------- 新增：内部辅助方法 --------------------------
// initCachedConnID 初始化并缓存当前连接ID（内部方法，避免重复类型断言）
func (c *BaseController) initCachedConnID() {
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		if grCtx != nil {
			grCtx.cachedConnID = ""
		}
		logger.Warn("BaseController 上下文未注入，无法获取ConnID")
		return
	}
	// 兼容两种WS上下文类型
	switch ctx := grCtx.ctx.(type) {
	case *websocket.Context:
		grCtx.cachedConnID = ctx.GetConnID()
	case websocket.ConnIDContext:
		grCtx.cachedConnID = ctx.GetConnID()
	default:
		// 非WS上下文（HTTP/GRPC），ConnID置空
		grCtx.cachedConnID = ""
	}
}

// GetConnID 获取当前连接ID（应用层直接调用）
func (c *BaseController) GetConnID() string {
	if c == nil {
		return ""
	}
	// 获取当前goroutine的上下文
	grCtx := c.getGoroutineContext()
	if grCtx == nil {
		return ""
	}
	// 若缓存为空，重新尝试初始化
	if grCtx.cachedConnID == "" {
		c.initCachedConnID()
	}
	return grCtx.cachedConnID
}

// 全局默认连接管理器
var defaultConnManager ConnManager

// SetDefaultConnManager 设置全局默认连接管理器（框架层调用）
func SetDefaultConnManager(manager ConnManager) {
	defaultConnManager = manager
}

// getConnManager 获取连接管理器（内部方法，确保非nil）
func (c *BaseController) getConnManager() ConnManager {
	if c == nil || c.connManager == nil {
		// 向后兼容：使用全局默认连接管理器
		return defaultConnManager
	}
	return c.connManager
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
	// 获取连接管理器
	connManager := c.getConnManager()
	if connManager == nil {
		return errors.New("连接管理器未初始化，无法绑定用户ID")
	}
	// 存储用户ID到连接属性
	connManager.SetConnAttr(connID, c.UserIDField, userID)
	// 获取当前goroutine的上下文
	grCtx := c.getGoroutineContext()
	if grCtx == nil {
		return errors.New("无法获取goroutine上下文")
	}
	connIDsObj, exists := grCtx.userConnMap.Load(userID)
	var connIDs []string
	if exists {
		connIDs, _ = connIDsObj.([]string)
	}
	// 去重：避免重复绑定同一连接
	for _, cid := range connIDs {
		if cid == connID {
			c.LogInfo("用户ID与连接已绑定，无需重复操作", "userID", userID, "connID", connID)
			return nil
		}
	}
	connIDs = append(connIDs, connID)
	grCtx.userConnMap.Store(userID, connIDs)
	if c.log != nil && c.log.GetEnv() != "prod" {
		c.LogInfo("用户ID与连接绑定成功", "userID", userID, "connID", connID)
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
	connManager := c.getConnManager()
	if connManager != nil {
		connManager.SetConnAttr(connID, c.UserIDField, "")
	}

	// 2. 维护用户-连接映射，移除当前连接
	grCtx := c.getGoroutineContext()
	if grCtx == nil {
		return errors.New("无法获取goroutine上下文")
	}
	connIDsObj, exists := grCtx.userConnMap.Load(userID)
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
		grCtx.userConnMap.Delete(userID)
	} else {
		grCtx.userConnMap.Store(userID, newConnIDs)
	}
	if c.log != nil && c.log.GetEnv() != "prod" {
		c.LogInfo("用户ID与连接解绑成功", "userID", userID, "connID", connID)
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

	// 获取当前goroutine的上下文
	grCtx := c.getGoroutineContext()
	if grCtx == nil {
		return nil, errors.New("无法获取goroutine上下文")
	}

	// 1. 从用户-连接映射中获取连接列表
	connIDsObj, exists := grCtx.userConnMap.Load(userID)
	if !exists {
		return []string{}, nil
	}
	connIDs, ok := connIDsObj.([]string)
	if !ok {
		return nil, errors.New("用户连接列表格式错误")
	}

	// 2. 过滤无效连接（可选：校验连接是否仍在线）
	var validConnIDs []string
	connManager := c.getConnManager()
	if connManager != nil {
		for _, connID := range connIDs {
			if _, ok := connManager.GetConnByConnID(connID); ok {
				validConnIDs = append(validConnIDs, connID)
			}
		}
	} else {
		// 如果连接管理器未初始化，直接返回所有连接ID
		validConnIDs = connIDs
	}

	// 3. 更新有效连接映射（避免无效数据堆积）
	if len(validConnIDs) != len(connIDs) {
		if len(validConnIDs) == 0 {
			grCtx.userConnMap.Delete(userID)
		} else {
			grCtx.userConnMap.Store(userID, validConnIDs)
		}
	}
	if c.log != nil && c.log.GetEnv() != "prod" {
		c.LogInfo("获取用户在线连接ID成功", "userID", userID, "connCount", len(validConnIDs))
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
		if c != nil && c.log != nil {
			c.log.Error("标准消息序列化失败", "error", err, "action", action)
		} else {
			logger.Error("标准消息序列化失败", "error", err, "action", action)
		}
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
	// 获取连接管理器
	connManager := c.getConnManager()
	if connManager == nil {
		return errors.New("连接管理器未初始化，无法发送消息")
	}
	// 组装标准消息
	msgStr := c.buildStandardMsg(msgAction, msgData)
	if msgStr == "" {
		return errors.New("消息组装失败，无法发送")
	}
	// 发送消息
	err := connManager.SendToConnID(targetConnID, msgStr)
	if err != nil {
		if c.log != nil {
			c.log.Error("给指定连接发送消息失败", "targetConnID", targetConnID, "error", err)
		} else {
			logger.Error("给指定连接发送消息失败", "targetConnID", targetConnID, "error", err)
		}
		return err
	}
	if c.log != nil && c.log.GetEnv() != "prod" {
		c.LogInfo("给指定连接发送消息成功", "targetConnID", targetConnID, "msgAction", msgAction)
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
	// 获取连接管理器
	connManager := c.getConnManager()
	if connManager == nil {
		return errors.New("连接管理器未初始化，无法发送消息")
	}
	// 组装标准消息
	msgStr := c.buildStandardMsg(msgAction, msgData)
	if msgStr == "" {
		return errors.New("消息组装失败，无法发送")
	}
	// 批量发送消息
	connManager.Multicast(targetConnIDs, msgStr)
	if c.log != nil && c.log.GetEnv() != "prod" {
		c.LogInfo("批量给多个连接发送消息成功", "connCount", len(targetConnIDs), "msgAction", msgAction)
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
	// 获取连接管理器
	connManager := c.getConnManager()
	if connManager == nil {
		return errors.New("连接管理器未初始化，无法发送消息")
	}

	// 1. 调用基类GetUserConnIDs获取用户在线连接
	targetConnIDs, err := c.GetUserConnIDs(targetUserID)
	if err != nil {
		return fmt.Errorf("获取目标用户在线连接失败：%w", err)
	}
	if len(targetConnIDs) == 0 {
		if c.log != nil && c.log.GetEnv() != "prod" {
			c.log.Warn("目标用户无在线连接，无需发送消息", "targetUserID", targetUserID)
		} else if c.log == nil {
			logger.Warn("目标用户无在线连接，无需发送消息", "targetUserID", targetUserID)
		}
		return nil
	}

	// 2. 组装标准消息
	msgStr := c.buildStandardMsg(msgAction, msgData)
	if msgStr == "" {
		return errors.New("消息组装失败，无法发送")
	}

	// 3. 给该用户的所有在线连接发送消息
	connManager.Multicast(targetConnIDs, msgStr)
	if c.log != nil && c.log.GetEnv() != "prod" {
		c.LogInfo("给指定用户发送消息成功", "targetUserID", targetUserID, "connCount", len(targetConnIDs), "msgAction", msgAction)
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
	// 获取连接管理器
	connManager := c.getConnManager()
	if connManager == nil {
		return errors.New("连接管理器未初始化，无法发送消息")
	}

	// 1. 批量收集所有目标用户的在线连接ID
	var allTargetConnIDs []string
	for _, userID := range targetUserIDs {
		connIDs, err := c.GetUserConnIDs(userID)
		if err != nil {
			if c.log != nil && c.log.GetEnv() != "prod" {
				c.log.Warn("获取用户在线连接失败，跳过该用户", "userID", userID, "error", err)
			} else if c.log == nil {
				logger.Warn("获取用户在线连接失败，跳过该用户", "userID", userID, "error", err)
			}
			continue
		}
		allTargetConnIDs = append(allTargetConnIDs, connIDs...)
	}
	if len(allTargetConnIDs) == 0 {
		if c.log != nil && c.log.GetEnv() != "prod" {
			c.log.Warn("所有目标用户均无在线连接，无需发送消息", "userCount", len(targetUserIDs))
		} else if c.log == nil {
			logger.Warn("所有目标用户均无在线连接，无需发送消息", "userCount", len(targetUserIDs))
		}
		return nil
	}

	// 2. 组装标准消息
	msgStr := c.buildStandardMsg(msgAction, msgData)
	if msgStr == "" {
		return errors.New("消息组装失败，无法发送")
	}

	// 3. 批量发送消息
	connManager.Multicast(allTargetConnIDs, msgStr)
	if c.log != nil && c.log.GetEnv() != "prod" {
		c.LogInfo("批量给多个用户发送消息成功", "userCount", len(targetUserIDs), "connCount", len(allTargetConnIDs), "msgAction", msgAction)
	}
	return nil
}
func (c *BaseController) GetCtx() (netContext.Context, error) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Success响应")
		return nil, errors.New("控制器未初始化（是使用c.Init(ctx)进行初始化），无法执行Success响应")
	}
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.Success 之前未设置上下文")
		return nil, errors.New("控制器未设置上下文，无法执行Success响应")
	}
	return grCtx.ctx, nil
}

// Success 统一成功响应（JSON格式）
func (c *BaseController) Success(data interface{}, msg ...string) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行Success响应")
		return
	}
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.Success 之前未设置上下文")
		return
	}
	message := "操作成功"
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	grCtx.ctx.Set("resultCode", 200)
	grCtx.ctx.Set("resultMsg", function.Json_encode(map[string]interface{}{
		"msg":  "操作成功",
		"data": data,
	}))
	grCtx.ctx.JSON(200, map[string]interface{}{
		"code": 200,
		"msg":  message,
		"data": data,
	})
}

// DataSuccess 统一成功响应（JSON格式）
func (c *BaseController) DataSuccess(data interface{}, count int64) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行DataSuccess响应")
		return
	}
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.DataSuccess 之前未设置上下文")
		return
	}
	grCtx.ctx.Set("resultCode", 200)
	grCtx.ctx.Set("resultMsg", function.Json_encode(map[string]interface{}{
		"msg":   "操作成功",
		"data":  data,
		"count": count,
	}))
	grCtx.ctx.JSON(200, map[string]interface{}{
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
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.Error 之前未设置上下文")
		return
	}
	reqInfo := grCtx.ctx.GetRequestInfo()
	if c.log != nil && c.log.GetEnv() != "prod" {
		c.LogError("接口响应失败：", "code=", code, "msg=", msg, "path=", reqInfo.GetPath())
	}
	grCtx.ctx.Set("resultCode", code)
	grCtx.ctx.Set("resultMsg", msg)
	grCtx.ctx.JSON(200, map[string]interface{}{
		"code": code,
		"msg":  msg,
		"data": nil,
	})
}

// RespText 统一响应（字符串格式）
func (c *BaseController) RespText(msg string) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行RespText响应")
		return
	}
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.RespText 之前未设置上下文")
		return
	}
	grCtx.ctx.Set("resultCode", 200)
	grCtx.ctx.Set("resultMsg", msg)
	grCtx.ctx.String(200, msg)
}

// InternalError 服务器内部错误响应
func (c *BaseController) InternalError() {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行InternalError响应")
		return
	}
	c.Error(500, "服务器内部错误")
}

// GetQuery 获取URL查询参数
func (c *BaseController) GetQuery(key string) string {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行GetQuery")
		return ""
	}
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.GetQuery 之前未设置上下文")
		return ""
	}
	return grCtx.ctx.Query(key)
}

// GetPostForm 获取POST表单参数
func (c *BaseController) GetPostForm(key string) string {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行GetPostForm")
		return ""
	}
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.GetPostForm 之前未设置上下文")
		return ""
	}
	return grCtx.ctx.PostForm(key)
}
func (c *BaseController) GetPostFormAll() map[string]string {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行GetPostFormAll")
		return nil
	}
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.GetPostFormAll 之前未设置上下文")
		return nil
	}
	return grCtx.ctx.PostFormAll()
}
func (c *BaseController) GetBody() ([]byte, error) {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行GetBody")
		return nil, nil
	}
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.GetBody 之前未设置上下文")
		return nil, nil
	}
	return grCtx.ctx.GetBody()
}

// BindJSON 绑定JSON请求体到结构体
func (c *BaseController) BindJSON(v interface{}) error {
	if c == nil {
		logger.Error("BaseController 未初始化（指针为nil），无法执行BindJSON")
		return errors.New("BaseController 未初始化（指针为nil），无法执行BindJSON")
	}
	grCtx := c.getGoroutineContext()
	if grCtx == nil || grCtx.ctx == nil {
		c.LogError("调用框架BaseController.BindJSON 之前未设置上下文")
		return errors.New("调用框架BaseController.BindJSON 之前未设置上下文")
	}
	grCtx.ctx.Set("resultCode", 200)
	grCtx.ctx.Set("resultMsg", function.Json_encode(v))
	return grCtx.ctx.BindJSON(v)
}

// Cleanup 清理当前goroutine的上下文信息（框架自动调用）
func (c *BaseController) Cleanup() {
	if c == nil {
		return
	}
	// 使用goroutine ID作为key
	key := getGoroutineID()
	c.goroutineCtx.Delete(key)
}
