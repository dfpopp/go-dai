package websocket

import (
	"encoding/json"
	"errors"
)

// Router WS路由器（框架内置，非系统包，供Server内部使用）
type Router struct {
	handlers    map[string]HandlerFunc // action -> 包装后的处理器
	middlewares []MiddlewareFunc       // 全局中间件（由Server注入）
}

// NewRouter 创建WS路由器实例（框架内置）
func NewRouter() *Router {
	return &Router{
		handlers:    make(map[string]HandlerFunc),
		middlewares: make([]MiddlewareFunc, 0),
	}
}

// Use 注入全局中间件（由Server调用）
func (r *Router) Use(middlewares ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// Register 注册WS路由（由Server调用，统一处理中间件链）
func (r *Router) Register(action string, handler HandlerFunc, chain []MiddlewareFunc) {
	r.handlers[action] = buildChain(chain, handler)
}

// Dispatch WS路由分发（内部方法，供WS Server调用）
func (r *Router) Dispatch(ctx *Context) error {
	action := ctx.Action
	handler, exists := r.handlers[action]
	if !exists {
		ctx.JSON(200, map[string]interface{}{
			"code": 404,
			"msg":  "无效的接口",
			"data": nil,
		})
		return errors.New("invalid ws action: " + action)
	}
	handler(ctx)
	return nil
}

// ParseMessage 解析WS消息（内部方法，供WS Server调用）
func (r *Router) ParseMessage(rawMsg []byte) (action, requestId string, data []byte, err error) {
	type WsReq struct {
		Action    string          `json:"action"`
		RequestId string          `json:"request_id"`
		Data      json.RawMessage `json:"data"`
	}

	var req WsReq
	if err := json.Unmarshal(rawMsg, &req); err != nil {
		return "", "", nil, err
	}

	return req.Action, req.RequestId, req.Data, nil
}

// buildChain 构建中间件链（与HTTP服务逻辑一致）
func buildChain(middlewares []MiddlewareFunc, final HandlerFunc) HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		// 显式拷贝，避免闭包引用共享
		currentMid := middlewares[i]
		currentNext := final
		final = currentMid(currentNext)
	}
	return final
}
