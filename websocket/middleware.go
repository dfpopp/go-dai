package websocket

// HandlerFunc WS处理器函数（与http.HandlerFunc对齐）
type HandlerFunc func(*Context)

// MiddlewareFunc WS中间件函数（与http.MiddlewareFunc对齐）
type MiddlewareFunc func(HandlerFunc) HandlerFunc
