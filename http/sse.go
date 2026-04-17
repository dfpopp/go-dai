package http

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dfpopp/go-dai/logger"
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
	closed  bool
	mu      sync.RWMutex
	done    chan struct{}
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
		done:    make(chan struct{}),
	}, nil
}

// IsClosed 检查连接是否已关闭
func (s *SSEContext) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// Close 关闭SSE连接
func (s *SSEContext) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.done)
		logger.Info("SSE连接已关闭")
	}
}

// Send 发送SSE事件
func (s *SSEContext) Send(event SSEvent) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return http.ErrServerClosed
	}
	s.mu.RUnlock()

	// 写入事件数据
	if event.ID != "" {
		if _, err := s.Writer.Write([]byte("id: " + event.ID + "\n")); err != nil {
			s.Close()
			return err
		}
	}
	if event.Event != "" {
		if _, err := s.Writer.Write([]byte("event: " + event.Event + "\n")); err != nil {
			s.Close()
			return err
		}
	}
	if event.Retry > 0 {
		if _, err := s.Writer.Write([]byte("retry: " + string(rune(event.Retry)) + "\n")); err != nil {
			s.Close()
			return err
		}
	}
	if _, err := s.Writer.Write([]byte("data: " + event.Data + "\n\n")); err != nil {
		s.Close()
		return err
	}

	// 刷新缓冲区
	s.Flusher.Flush()
	return nil
}

// SSEHandler SSE处理器包装
func SSEHandler(handler func(*SSEContext)) HandlerFunc {
	return func(c *Context) {
		// 检查是否已经设置了Content-Type头
		if c.Writer.Header().Get("Content-Type") != "" && c.Writer.Header().Get("Content-Type") != "text/plain;charset=utf-8" {
			logger.Error(fmt.Errorf("SSEHandler: 响应头已发送，无法执行SSE处理（已发送的contentType：%s，请求路径：%s，请求方法：%s）",
				c.Writer.Header().Get("Content-Type"), c.Req.URL.Path, c.Req.Method))
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"code": 500,
				"msg":  "服务器内部错误",
			})
			return
		}

		sseCtx, err := NewSSEContext(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"code": 400,
				"msg":  "不支持SSE协议",
			})
			return
		}
		defer sseCtx.Close()

		// 心跳检测
		heartbeatDone := make(chan struct{})
		go func() {
			defer close(heartbeatDone)
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if sseCtx.IsClosed() {
						return
					}
					if err := sseCtx.Send(SSEvent{Data: "ping"}); err != nil {
						logger.Info("SSE心跳发送失败:", err)
						sseCtx.Close()
						return
					}
				case <-sseCtx.done:
					return
				}
			}
		}()

		// 执行业务处理器
		handler(sseCtx)

		// 等待心跳goroutine退出
		<-heartbeatDone
	}
}
