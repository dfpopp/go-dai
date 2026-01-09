package http

import (
	"github.com/dfpopp/go-dai/logger"
	"net/http"
	"time"
)

// SSEvent SSE事件结构
type SSEvent struct {
	Event string // 事件类型
	Data  string // 事件数据
	ID    string // 事件ID
	Retry int    // 重连时间（毫秒）
}

// SSEContext SSE上下文
type SSEContext struct {
	Writer  http.ResponseWriter
	Flusher http.Flusher
	Closed  bool
}

// NewSSEContext 创建SSE上下文
func NewSSEContext(w http.ResponseWriter) (*SSEContext, error) {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 获取flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, http.ErrNotSupported
	}

	return &SSEContext{
		Writer:  w,
		Flusher: flusher,
		Closed:  false,
	}, nil
}

// Send 发送SSE事件
func (s *SSEContext) Send(event SSEvent) error {
	if s.Closed {
		return http.ErrClosed
	}

	// 写入事件数据
	if event.ID != "" {
		_, err := s.Writer.Write([]byte("id: " + event.ID + "\n"))
		if err != nil {
			return err
		}
	}
	if event.Event != "" {
		_, err := s.Writer.Write([]byte("event: " + event.Event + "\n"))
		if err != nil {
			return err
		}
	}
	if event.Retry > 0 {
		_, err := s.Writer.Write([]byte("retry: " + string(rune(event.Retry)) + "\n"))
		if err != nil {
			return err
		}
	}
	_, err := s.Writer.Write([]byte("data: " + event.Data + "\n\n"))
	if err != nil {
		return err
	}

	// 刷新缓冲区
	s.Flusher.Flush()
	return nil
}

// Close 关闭SSE连接
func (s *SSEContext) Close() {
	s.Closed = true
	logger.Info("SSE连接已关闭")
}

// SSEHandler SSE处理器包装
func SSEHandler(handler func(*SSEContext)) HandlerFunc {
	return func(c *Context) {
		sseCtx, err := NewSSEContext(c.Writer)
		if err != nil {
			c.Error(400, "不支持SSE协议")
			return
		}
		defer sseCtx.Close()

		// 心跳检测
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					_ = sseCtx.Send(SSEvent{Data: "ping"})
				}
			}
		}()

		// 执行业务处理器
		handler(sseCtx)
	}
}
