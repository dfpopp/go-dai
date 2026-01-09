package websocket

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/logger"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var ErrServerClosed = http.ErrServerClosed

// 帧操作码定义（原有逻辑不变）
const (
	opCodeContinuation = 0x0
	opCodeText         = 0x1
	opCodeBinary       = 0x2
	opCodeClose        = 0x8
	opCodePing         = 0x9
	opCodePong         = 0xA
)

// ServerConfig WS服务器配置（原有逻辑不变，已包含SSL字段）
type ServerConfig struct {
	Addr             string        // 监听地址（ip:port）
	ReadTimeout      time.Duration // 读超时
	WriteTimeout     time.Duration // 写超时
	Path             string        // WebSocket监听路径（如：/ws）
	Origin           string        // 允许的来源（* 表示允许所有）
	HandshakeTimeout time.Duration // 握手超时（默认3秒）
	MaxMessageSize   int64         // 最大消息大小（默认1MB）
	MaxConnections   int32         // 最大连接数（默认1000）
	SSL              bool          // 是否启用SSL/TLS（启用后为WSS，禁用为WS）
	SSLCertFile      string        // SSL证书路径（如：./cert/server.crt）
	SSLKeyFile       string        // SSL密钥路径（如：./cert/server.key）
}

// Conn WS连接封装（原有逻辑不变）
type Conn struct {
	conn         io.ReadWriteCloser
	readBuf      []byte
	writeBuf     []byte
	maxMsgSize   int64
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// Server WS服务器（框架内置，对齐HTTP Server使用风格）
type Server struct {
	config          *ServerConfig
	server          *http.Server
	router          *Router          // 框架WS Router（内部持有）
	connectionCount int32            // 连接计数器
	middlewares     []MiddlewareFunc // 全局中间件
}

// NewServer 创建WS服务器实例（原有逻辑不变）
func NewServer(appName string) *Server {
	cfg := loadServerConfig(appName)
	setDefaultConfig(cfg)
	router := NewRouter()
	return &Server{
		config: cfg,
		server: &http.Server{
			Addr:         cfg.Addr,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
		router:      router, // 内部初始化Router
		middlewares: make([]MiddlewareFunc, 0),
	}
}

// Config 暴露配置（原有逻辑不变）
func (s *Server) Config() *ServerConfig {
	return s.config
}

// Use 注册全局中间件（对齐HTTP Server.Use）
func (s *Server) Use(middlewares ...MiddlewareFunc) {
	s.middlewares = append(s.middlewares, middlewares...)
	// 同步到Router
	s.router.Use(middlewares...)
}

// Register 注册WS路由（对齐HTTP Server.Handle/GET/POST，核心新增方法）
func (s *Server) Register(action string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	// 构建完整中间件链：全局中间件 + 局部中间件
	chain := append(s.middlewares, middlewares...)
	// 注册到Router
	s.router.Register(action, handler, chain)
}

// Run 启动WS/WSS服务器（核心改造：添加SSL判断，支持两种监听模式）
func (s *Server) Run() error {
	// 注册WS握手处理器
	http.HandleFunc(s.config.Path, s.handleRequest)

	// 根据SSL配置选择监听模式
	if s.config.SSL {
		// 启用WSS：加载证书并创建TLS监听器
		// 验证证书文件路径
		if s.config.SSLCertFile == "" || s.config.SSLKeyFile == "" {
			return fmt.Errorf("SSL enabled but cert/key file path is empty")
		}
		// 加载X509证书和密钥
		cert, err := tls.LoadX509KeyPair(s.config.SSLCertFile, s.config.SSLKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load SSL cert/key: %w", err)
		}
		// 配置TLS
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12, // 推荐的最小TLS版本
		}
		// 创建TLS监听器
		lis, err := tls.Listen("tcp", s.config.Addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to create WSS listener: %w", err)
		}
		// 打印WSS启动日志
		logger.Info("WSS服务器启动成功，监听地址：", s.config.Addr, "路径：", s.config.Path)
		defer func(lis net.Listener) {
			err := lis.Close()
			if err != nil {
				logger.Error("WSS服务器关闭后释放lis失败，监听地址：", s.config.Addr, "路径：", s.config.Path)
			}
		}(lis)
		return s.server.Serve(lis)
	} else {
		// 启用WS：普通TCP监听（原有逻辑）
		logger.Info("WS服务器启动成功，监听地址：", s.config.Addr, "路径：", s.config.Path)
		return s.server.ListenAndServe()
	}
}

// Stop 停止WS服务器（原有逻辑不变）
func (s *Server) Stop() error {
	logger.Info("WebSocket服务器正在停止...当前连接数：", atomic.LoadInt32(&s.connectionCount))
	if s.server != nil {
		return s.server.Shutdown(nil)
	}
	return nil
}

// handleRequest 处理WS请求（使用框架Router分发，原有逻辑不变）
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// 1. 连接限流
	currentConn := atomic.AddInt32(&s.connectionCount, 1)
	defer atomic.AddInt32(&s.connectionCount, -1)

	if currentConn > s.config.MaxConnections {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	// 2. 握手超时控制
	handshakeDone := make(chan struct{})
	defer close(handshakeDone)
	timeoutTimer := time.After(s.config.HandshakeTimeout)

	// 3. 执行握手
	var (
		wsConn *Conn
		err    error
	)
	go func() {
		wsConn, err = s.upgrade(w, r)
		handshakeDone <- struct{}{}
	}()

	select {
	case <-handshakeDone:
		if err != nil {
			http.Error(w, fmt.Sprintf("handshake failed: %v", err), http.StatusBadRequest)
			return
		}
	case <-timeoutTimer:
		http.Error(w, "handshake timeout", http.StatusGatewayTimeout)
		return
	}

	// 新增：获取客户端IP
	clientIP := getClientIPFromRequest(r)
	// 新增：添加连接到全局管理器
	connInfo := GetGlobalConnManager().AddConn(wsConn, clientIP)
	connID := connInfo.ConnID

	// 新增：延迟移除连接（传递下线原因）
	closeReason := "normal closure"
	defer func() {
		GetGlobalConnManager().RemoveConn(connID, closeReason)
		_ = wsConn.Close()
	}()

	// 4. 消息循环（传递connID和closeReason指针）
	s.messageLoop(wsConn, r, connID, &closeReason)
}

// messageLoop 消息处理循环（依赖框架Router，原有逻辑不变）
func (s *Server) messageLoop(wsConn *Conn, r *http.Request, connID string, closeReason *string) {
	if s.router == nil {
		logger.Error("WS路由器未初始化")
		return
	}

	// 设置连接的消息大小和超时
	wsConn.maxMsgSize = s.config.MaxMessageSize
	wsConn.readTimeout = s.config.ReadTimeout
	wsConn.writeTimeout = s.config.WriteTimeout

	for {
		// 读取原始消息
		rawMsg, err := wsConn.ReadMessage()
		if err != nil {
			*closeReason = err.Error() // 更新下线原因
			logger.Error("WS读取消息失败：", err, "连接ID：", connID, "客户端：", wsConn.RemoteAddr())
			break
		}

		// 框架Router解析消息
		action, requestId, data, err := s.router.ParseMessage(rawMsg)
		if err != nil {
			logger.Warn("WS解析消息失败：", err, "连接ID：", connID, "客户端：", wsConn.RemoteAddr())
			_ = wsConn.WriteMessage(string(`{"code":400,"msg":"消息格式错误","data":null}`))
			continue
		}

		// 创建WS上下文（传入connID）
		ctx := NewContext(wsConn, r, action, requestId, connID, data)

		// 框架Router分发消息
		if err := s.router.Dispatch(ctx); err != nil {
			logger.Error("WS路由分发失败：", err, "action：", action, "连接ID：", connID)
		}
	}
}

// upgrade 升级为WS连接（原有逻辑不变）
func (s *Server) upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if r.Header.Get("Upgrade") != "websocket" {
		return nil, errors.New("invalid upgrade header (expected 'websocket')")
	}
	if r.Header.Get("Connection") != "Upgrade" && r.Header.Get("Connection") != "upgrade" {
		return nil, errors.New("invalid connection header (expected 'Upgrade')")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return nil, errors.New("unsupported websocket version (only RFC 6455/13 is supported)")
	}

	// 跨域校验
	origin := r.Header.Get("Origin")
	if s.config.Origin != "*" && origin != "" && origin != s.config.Origin {
		return nil, fmt.Errorf("origin '%s' not allowed", origin)
	}

	// 握手密钥校验
	clientKey := r.Header.Get("Sec-WebSocket-Key")
	if clientKey == "" {
		return nil, errors.New("missing 'Sec-WebSocket-Key' header")
	}
	serverKey := generateServerKey(clientKey)

	// 获取底层TCP连接
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("response writer does not support hijack (required for websocket)")
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijack failed: %v", err)
	}

	// 写入握手响应
	response := fmt.Sprintf(
		"HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n\r\n",
		serverKey,
	)
	_, err = conn.Write([]byte(response))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("write handshake response failed: %v", err)
	}

	return &Conn{
		conn:     conn,
		readBuf:  make([]byte, 4096),
		writeBuf: make([]byte, 4096),
	}, nil
}

// 工具方法及Conn的其他方法（原有逻辑不变）
func generateServerKey(clientKey string) string {
	hash := sha1.New()
	hash.Write([]byte(clientKey + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func (c *Conn) ReadMessage() (message []byte, err error) {
	if c.readTimeout > 0 {
		if conn, ok := c.conn.(interface{ SetReadDeadline(time.Time) error }); ok {
			_ = conn.SetReadDeadline(time.Now().Add(c.readTimeout))
		}
	}

	for {
		fin, opCode, payload, err := c.readFrame()
		if err != nil {
			return nil, err
		}

		switch opCode {
		case opCodeClose:
			return nil, errors.New("client closed connection")
		case opCodePing:
			_ = c.writeFrame(true, opCodePong, payload)
			continue
		case opCodePong:
			continue
		}

		if int64(len(message)+len(payload)) > c.maxMsgSize {
			_ = c.WriteCloseMessage(1009, "message size exceeds limit")
			return nil, errors.New("message size exceeds limit")
		}

		message = append(message, payload...)

		if fin {
			return message, nil
		}
	}
}

func (c *Conn) WriteMessage(message string) error {
	return c.writeFrame(true, opCodeText, []byte(message))
}

func (c *Conn) WriteCloseMessage(code int, reason string) error {
	payload := make([]byte, 2+len(reason))
	payload[0] = byte(code >> 8)
	payload[1] = byte(code & 0xff)
	copy(payload[2:], []byte(reason))
	return c.writeFrame(true, opCodeClose, payload)
}

func (c *Conn) Close() error {
	_ = c.WriteCloseMessage(1000, "normal closure")
	return c.conn.Close()
}

func (c *Conn) RemoteAddr() string {
	if conn, ok := c.conn.(interface{ RemoteAddr() string }); ok {
		return conn.RemoteAddr()
	}
	return "unknown"
}

func (c *Conn) readFrame() (fin bool, opCode byte, payload []byte, err error) {
	_, err = io.ReadFull(c.conn, c.readBuf[:1])
	if err != nil {
		return false, 0, nil, err
	}
	fin = (c.readBuf[0] & 0x80) != 0
	opCode = c.readBuf[0] & 0x0f

	_, err = io.ReadFull(c.conn, c.readBuf[:1])
	if err != nil {
		return false, 0, nil, err
	}
	masked := (c.readBuf[0] & 0x80) != 0
	payloadLen := uint64(c.readBuf[0] & 0x7f)

	switch payloadLen {
	case 126:
		_, err = io.ReadFull(c.conn, c.readBuf[:2])
		if err != nil {
			return false, 0, nil, err
		}
		payloadLen = uint64(c.readBuf[0])<<8 | uint64(c.readBuf[1])
	case 127:
		_, err = io.ReadFull(c.conn, c.readBuf[:8])
		if err != nil {
			return false, 0, nil, err
		}
		payloadLen = 0
		for i := 0; i < 8; i++ {
			payloadLen = payloadLen<<8 | uint64(c.readBuf[i])
		}
	}

	if payloadLen > uint64(c.maxMsgSize) {
		return false, 0, nil, errors.New("payload too large")
	}

	mask := make([]byte, 4)
	if masked {
		_, err = io.ReadFull(c.conn, mask)
		if err != nil {
			return false, 0, nil, err
		}
	}

	payload = make([]byte, payloadLen)
	if payloadLen > 0 {
		_, err = io.ReadFull(c.conn, payload)
		if err != nil {
			return false, 0, nil, err
		}
	}

	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return fin, opCode, payload, nil
}

func (c *Conn) writeFrame(fin bool, opCode byte, payload []byte) error {
	if c.writeTimeout > 0 {
		if conn, ok := c.conn.(interface{ SetWriteDeadline(time.Time) error }); ok {
			_ = conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
		}
	}

	frameHeader := make([]byte, 0, 10)
	firstByte := byte(opCode)
	if fin {
		firstByte |= 0x80
	}
	frameHeader = append(frameHeader, firstByte)

	payloadLen := len(payload)
	secondByte := byte(0x00)
	switch {
	case payloadLen < 126:
		secondByte |= byte(payloadLen)
	case payloadLen < 65536:
		secondByte |= 126
	default:
		secondByte |= 127
	}
	frameHeader = append(frameHeader, secondByte)

	switch {
	case payloadLen >= 65536:
		frameHeader = append(frameHeader,
			byte(payloadLen>>56),
			byte(payloadLen>>48),
			byte(payloadLen>>40),
			byte(payloadLen>>32),
			byte(payloadLen>>24),
			byte(payloadLen>>16),
			byte(payloadLen>>8),
			byte(payloadLen),
		)
	case payloadLen >= 126:
		frameHeader = append(frameHeader, byte(payloadLen>>8), byte(payloadLen))
	}

	_, err := c.conn.Write(frameHeader)
	if err != nil {
		return err
	}
	if payloadLen > 0 {
		_, err = c.conn.Write(payload)
	}
	return err
}

func loadServerConfig(appName string) *ServerConfig {
	appCfg := config.GetAppConfig(appName)
	wsCfg := appCfg.WebSocket
	return &ServerConfig{
		Addr:             wsCfg.Addr,
		ReadTimeout:      time.Duration(wsCfg.ReadTimeout) * time.Second,
		WriteTimeout:     time.Duration(wsCfg.WriteTimeout) * time.Second,
		Path:             wsCfg.Path,
		Origin:           wsCfg.Origin,
		HandshakeTimeout: time.Duration(wsCfg.HandshakeTimeout) * time.Second,
		MaxMessageSize:   wsCfg.MaxMessageSize,
		MaxConnections:   wsCfg.MaxConnections,
		SSL:              wsCfg.SSL,
		SSLCertFile:      wsCfg.SSLCertFile,
		SSLKeyFile:       wsCfg.SSLKeyFile,
	}
}

func setDefaultConfig(cfg *ServerConfig) {
	if cfg.HandshakeTimeout == 0 {
		cfg.HandshakeTimeout = 3 * time.Second
	}
	if cfg.MaxMessageSize == 0 {
		cfg.MaxMessageSize = 1024 * 1024
	}
	if cfg.MaxConnections == 0 {
		cfg.MaxConnections = 1000
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 60 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 60 * time.Second
	}
	if cfg.Path == "" {
		cfg.Path = "/ws"
	}
	if cfg.Origin == "" {
		cfg.Origin = "*"
	}
}

// getClientIPFromRequest 提取客户端IP（复用Context逻辑）
func getClientIPFromRequest(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
		if ip != "" {
			ip = strings.Split(ip, ",")[0]
		}
	}
	if ip == "" {
		remoteAddr := r.RemoteAddr
		host, _, err := net.SplitHostPort(remoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = remoteAddr
		}
	}
	return ip
}
