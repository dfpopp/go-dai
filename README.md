# go-Dai框架简介

go-Dai 是一款集多协议、多数据库引擎于一体的 Go 服务开发框架，特别沿用 PHP 主流框架设计思路，更适配 PHP 程序员的开发习惯，降低 Go 语言技术栈的学习与迁移成本；框架原生支持 HTTP、WebSocket、gRPC 多种服务协议，兼容 MySQL、MongoDB、Redis、ElasticSearch 等主流数据库引擎，且数据库操作支持简洁高效的链式调用；采用经典 MVC 架构设计，实现路由（Router）、控制器（Controller）、服务（Service）、模型（Model）全链路高一致性，为多协议服务开发提供统一、可靠的技术基座。

### 为什么选择go-Dai

对于 PHP 程序员转 Go 开发，或需要快速落地多协议服务的团队，go-Dai 是更贴合需求的选择：它精准解决了跨语言迁移的思维切换痛点，同时打破了多协议开发需学习多种框架的壁垒，让开发者无需投入大量时间适配技术差异，就能用熟悉的模式高效开发高性能服务；更通过 MVC 全链路一致性设计，降低业务迭代与服务扩展的成本，助力快速落地业务、提升团队研发效率。

### 核心优势

- PHP友好，低迁移成本：沿用 Laravel/ThinkPHP 等主流 PHP 框架的 MVC 分层思想、路由设计规范及控制器写法，契合 PHP 程序员的开发习惯，无需大幅切换思维模式，轻松上手 Go 开发

- 多协议统一，开发范式一致：一套编码规范即可开发 HTTP、WebSocket、gRPC 多种服务接口，无需学习多种 Go 框架的差异化用法，降低多协议开发的学习与维护成本

- 数据库操作便捷，贴近PHP ORM体验：兼容 MySQL、MongoDB、Redis、ElasticSearch 等主流数据库引擎，支持简洁高效的链式调用，无需熟悉复杂的 Go 原生数据库操作语法，上手门槛低

- 开箱即用，快速启动：内置完善的默认配置（服务端口、超时时间等）、统一错误处理、常用中间件（限流、认证等）基础能力，无需重复搭建基础架构，一键启动开发

- 服务切换成本低，复用性强：基于 MVC 全链路高一致性设计，模型（Model）与服务（Service）层完全脱离协议依赖。当需要在 HTTP/WS/gRPC 服务间切换或新增服务协议时，仅需修改少量路由配置及控制器适配代码，核心业务逻辑（Model/Service）无需改动，大幅提升代码复用率与业务迭代效率

# 1. 环境要求与快速安装

- 环境要求：Go 1.24+（建议使用1.24及以上稳定版本，低版本兼容性未充分测试）、Protobuf（如需使用gRPC）、对应数据库客户端（MySQL/MongoDB等）

- 安装命令：简洁的go get安装指令，示例：go get github.com/dfpopp/go-dai

# 2. 应用层目录结构

```text

├─ app/                应用根目录
│  └─ api/             应用核心目录（支持多应用）
│     ├─ controller/   控制器目录：处理请求分发
│     ├─ service/      服务目录：封装业务逻辑
│     ├─ model/        模型目录：映射数据实体
│     ├─ router/       路由目录：配置接口路由
│     ├─ middleware/   中间件目录：请求拦截与增强
│     └─ init/         初始化目录（仅WS服务适用）
├─ config/             配置文件根目录
│  ├─ app.json         应用配置（必选，支持多应用拆分）
│  ├─ database.json    数据库引擎配置（必选）
│  └─ custom.json      自定义配置（文件名可自定义，需自行实现读取逻辑，框架支持配置钩子）
├─ gen/                gRPC专用目录：存放*.pb.go、*_grpc.pb.go文件
├─ proto/              gRPC专用目录：存放*.proto文件
├─ pkg/                依赖包目录：存放第三方依赖或自定义包（含配置钩子函数）
├─ runtime/            运行时目录：存放日志文件
├─ scripts/            脚本目录：存放自定义脚本（可选）
├─ api.go              入口文件（多应用场景可配置多个）
└─ go.mod              项目依赖配置文件
```

# 3. 快速上手：三种协议服务示例

本节通过简单的「用户信息查询」场景，展示 HTTP、WebSocket、gRPC 三种服务的开发流程，体现框架统一的开发范式。

## 3.0 核心分层逻辑说明

框架采用严格的MVC分层设计，各层职责清晰，是多协议服务可复用、可维护的基础，适用于所有服务开发：

- **控制器（Controller）**：仅负责请求分发、参数校验、响应封装，不直接操作数据；所有业务逻辑通过调用服务层实现，不同协议（HTTP/WS/gRPC）仅需适配上下文和请求/响应格式，核心逻辑复用。

- **服务（Service）**：封装核心业务逻辑，是多协议复用的核心；仅通过模型层操作数据，与具体协议完全解耦，支持在不同控制器中直接复用。

- **模型（Model）**：映射数据实体，负责数据的CRUD操作；仅被服务层调用，不与控制器直接交互，确保数据操作的一致性。

通过该分层设计，当需要新增或切换服务协议时，仅需开发/修改对应协议的控制器适配代码，服务层和模型层无需任何改动，实现“一次开发，多协议复用”。

## 3.1 前置准备：创建数据模型与服务层

核心业务逻辑（Model/Service）与协议无关，一次开发多协议复用。框架严格遵循「控制器仅调用服务、服务调用模型」的分层规范：

关键设计原则：控制器不允许直接调用模型，仅能通过服务层间接操作数据。因此在HTTP/WS/gRPC服务间切换时，仅需修改控制器的上下文适配和请求/响应格式转换，服务层和模型层无需任何改动，大幅提升代码复用率。

### 3.1.1 定义用户模型（Model）

文件路径：app/api/model/user.go

```Plain Text


package model

import (
	"github.com/dfpopp/go-dai/database"
)

// User 用户数据模型，映射MySQL users表
type User struct {
	ID         uint64  `json:"id" db:"id,primaryKey"` // 主键ID
	MerchantCode string `json:"merchant_code" db:"merchant_code"` // 商户编码（数据隔离）
	Username   string  `json:"username" db:"username"` // 用户名
	Age        int     `json:"age" db:"age"`           // 年龄
	Email      string  `json:"email" db:"email"`       // 邮箱
	CreateAt   int64   `json:"create_at" db:"create_at"` // 创建时间
}

// TableName 指定模型对应的数据表名（框架规范：必须实现TableName方法）
func (u *User) TableName() string {
	return "users"
}

// NewModel 创建用户模型实例（供服务层调用初始化）
func NewModel() *User {
	return &User{}
}

// GetOne 单条查询（框架链式调用风格，贴近PHP ORM体验）
// fields：查询字段，where：查询条件（支持多条件拼接），args：条件参数
func (u *User) GetOne(fields string, where ...interface{}) (*User, error) {
	result := database.DB().Model(u).Fields(fields)
	// 拼接多条件查询（支持商户编码、ID等多维度条件）
	if len(where) > 0 {
		result = result.Where(where[0].(string), where[1:]...)
	}
	result = result.First()
	if result.Error != nil {
		return nil, result.Error
	}
	return u, nil
}

// GetById 根据ID查询用户信息（兼容原有用法）
func (u *User) GetById(id uint64) (*User, error) {
	return u.GetOne(
		"id,username,age,email,create_at",
		"id = ?", id,
	)
}
```

### 3.1.2 封装用户服务（Service）

文件路径：app/api/service/user.go

```Plain Text


package service

import (
	"fmt"
	"merchantServer/app/api/model"
)

// Service 用户业务服务（框架标准命名：统一使用Service作为结构体名）
type Service struct {
	// 注入用户模型实例（依赖注入，解耦服务与模型）
	UserModel *model.User
}

// NewService 初始化用户服务（框架标准命名：New+Service）
// 供控制器调用，完成模型依赖注入
func NewService() *Service {
	return &Service{
		// 初始化模型（模型层需提供NewModel方法）
		UserModel: model.NewModel(),
	}
}

// GetUserById 调用模型层查询用户信息，封装业务逻辑
// 入参/出参均为基础类型/结构体，无协议相关依赖，支持多协议复用
func (s *Service) GetUserById(merchantCode string, id uint64) (*model.User, error) {
	// 1. 业务参数校验（服务层核心职责：封装业务逻辑与参数校验）
	if merchantCode == "" {
		return nil, fmt.Errorf("商户编码不能为空")
	}
	if id <= 0 {
		return nil, fmt.Errorf("用户ID非法")
	}
	// 2. 调用模型层查询数据（服务层仅通过模型操作数据，不直接操作数据库）
	// 多条件查询（商户编码+用户ID），确保数据隔离
	return s.UserModel.GetOne(
		"id,username,age,email,create_at", // 指定查询字段，避免冗余
		"merchant_code = ?", // 数据隔离条件
		"id=?", id,
	)
}
```

关键设计原则：服务层代码仅依赖模型层，不依赖任何协议相关的上下文（如netContext、context.Context）或工具类。因此无论控制器是HTTP、WS还是gRPC类型，都能直接调用服务方法，实现「一次开发，多协议复用」。

文件路径：app/api/service/user.go

```Plain Text

package service

import (
	"fmt"
	"merchantServer/app/api/model"
)

// UserService 用户业务服务
type UserService struct {
	userModel *model.User
}

// NewUserService 初始化用户服务
func NewUserService() *UserService {
	return &UserService{
		userModel: &model.User{},
	}
}

// GetUserById 调用模型层查询用户信息，封装业务逻辑
func (s *UserService) GetUserById(id uint64) (*model.User, error) {
	// 可在此处添加业务校验、数据转换等逻辑
	if id <= 0 {
		return nil, fmt.Errorf("用户ID非法")
	}
	return s.userModel.GetById(id)
}
```

## 3.2 HTTP服务开发

### 3.2.1 编写HTTP控制器

文件路径：app/api/controller/user_controller.go

```Plain Text


package controller

import (
	"github.com/dfpopp/go-dai/base"
	"github.com/dfpopp/go-dai/netContext"
	"merchantServer/app/api/service/user"
	"strconv"
	"strings"
)

// UserController HTTP用户控制器（遵循：控制器仅调用服务，不直接操作模型）
type UserController struct {
	*base.BaseController // 继承框架基础控制器，获得请求处理、响应封装基础能力
	Service *user.Service // 注入用户服务，仅通过服务层调用业务逻辑
}

// NewUserController 创建控制器实例（初始化服务依赖）
func NewUserController() *UserController {
	return &UserController{
		BaseController: &base.BaseController{},
		Service:        user.NewUserService(), // 初始化服务层，解除与模型的直接依赖
	}
}

// GetUser 处理用户查询HTTP请求
// @Summary 查询用户信息
// @Description 根据用户ID查询用户详情
// @Param id path uint64 true "用户ID"
// @Success 200 {object} model.User "成功返回用户信息"
// @Failure 400 {string} string "参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /user/{id} [get]
func (c *UserController) GetUser(ctx netContext.Context) {
	// 1. 获取路径参数（框架不支持Param方法，通过GetRequestInfo().GetPath()解析）
	path := ctx.GetRequestInfo().GetPath()
	// 解析路径中的id（示例：从路径/user/123中提取123）
	userIdStr := strings.TrimPrefix(path, "/user/")
	if userIdStr == path { // 未提取到id
		c.BaseController.Error(400, "用户ID不能为空")
		return
	}
	// 转换为uint64类型
	userId, err := strconv.ParseUint(userIdStr, 10, 64)
	if err != nil {
		c.BaseController.Error(400, "用户ID格式错误")
		return
	}

	// 2. 调用服务层逻辑（控制器仅与服务交互，不直接操作模型）
	userInfo, err := c.Service.GetUserById(userId)
	if err != nil {
		c.BaseController.Error(500, err.Error())
		return
	}
	if userInfo == nil {
		c.BaseController.Error(500, "用户不存在")
		return
	}

	// 3. 统一响应格式
	c.BaseController.Success(userInfo)
}
```

### 3.2.2 配置HTTP路由（实现BaseRouter接口）

文件路径：app/api/router/http_router.go（实现框架BaseRouter接口，统一管理HTTP路由）

```Plain Text

package router

import (
	"github.com/dfpopp/go-dai/http"
	"merchantServer/app/api/controller"
	"merchantServer/app/api/middleware"
)

// ApiRouter API路由（实现框架BaseRouter接口）
type ApiRouter struct {
	userController *controller.UserController // 仅保留UserController，贴合示例场景
}

// NewApiRouter 创建API路由实例（供入口文件调用初始化）
func NewApiRouter() *ApiRouter {
	return &ApiRouter{
		userController: controller.NewUserController(), // 初始化UserController
	}
}

// RegisterHTTPRoutes 注册HTTP路由（框架约定方法名，用于绑定HTTP服务路由）
func (r *ApiRouter) RegisterHTTPRoutes(server *http.Server) {
	// 1. 公开路由（无需认证，仅添加限流中间件）
	// 限流中间件：middleware.RateLimit(200) 限制每秒200次请求
	server.GET("/user/:id", http.ToHTTPHandler(r.userController.GetUser), middleware.RateLimit(200))

	// 2. 路由分组：需认证路由（统一添加认证中间件+自定义中间件）
	// 定义认证分组函数，自动拼接认证中间件与自定义中间件
	authGroup := func(path string, handler http.HandlerFunc, middlewares ...http.MiddlewareFunc) {
		allMiddlewares := append([]http.MiddlewareFunc{middleware.Auth()}, middlewares...)
		server.POST(path, handler, allMiddlewares...)
	}
	// 注册认证路由（自动携带middleware.Auth()认证中间件）
	authGroup("/user/edit", http.ToHTTPHandler(r.userController.EditInfo), middleware.RateLimit(100)) // 认证+限流（100次/秒）
}
```

### 3.2.3 编写HTTP服务入口

文件路径：api.go

```Plain Text

package main

import (
	"github.com/dfpopp/go-dai/bootstrap"
	"github.com/dfpopp/go-dai/config"
	"merchantServer/app/api/router"
	pkgConfig "merchantServer/pkg/config"
)

func init() {
	// 注册自定义配置文件加载钩子（自定义配置读取逻辑）
	config.RegisterPostLoadHook(pkgConfig.CustomConfigLoadHook)
}
func main() {
	// 1. 加载配置
	appName := "api"
	if err := config.LoadAppConfig("config/app.json", appName); err != nil {
		panic("加载应用配置失败：" + err.Error())
	}
	if err := config.LoadDatabaseConfig("config/database.json"); err != nil {
		panic("加载数据库配置失败：" + err.Error())
	}

	// 2. 初始化路由实例（统一管理HTTP路由）
	apiRouter := router.NewApiRouter()

	// 3. 构建启动配置（启用HTTP服务）
	bootCfg := &bootstrap.BootConfig{
		AppName:            "api",
		AppConfigPath:      "config/app.json",
		DatabaseConfigPath: "config/database.json",
		EnableServices:     []bootstrap.ServiceType{bootstrap.ServiceTypeHTTP}, // 启用HTTP服务
		Router:             apiRouter, // 绑定路由实例，框架自动调用RegisterHTTPRoutes方法
	}
	// 4. 一键启动服务
	_, err := bootstrap.Boot(bootCfg)
	if err != nil {
		panic("应用启动失败: " + err.Error())
	}
}
```

## 3.3 WebSocket服务开发

### 3.3.1 编写WS控制器

文件路径：app/api/controller/ws_user_controller.go

```Plain Text

package controller

import (
	"github.com/dfpopp/go-dai/base"
	"github.com/dfpopp/go-dai/netContext"
	"merchantServer/app/api/service/user"
)

// WsUserController WebSocket用户控制器（与HTTP控制器逻辑一致，无需修改业务代码）
type WsUserController struct {
	*base.BaseController // 继承WS基础控制器，获得WS消息处理能力
	Service *user.Service // 复用用户服务，服务层逻辑完全复用
}

// NewWsUserController 创建WS控制器实例（初始化服务依赖）
func NewWsUserController() *WsUserController {
	return &WsUserController{
		BaseController: &base.BaseController{},
		Service:        user.NewUserService(), // 复用同一服务层，无需重复开发
	}
}

// GetUser 处理WS用户查询请求（业务逻辑与HTTP控制器一致，仅上下文类型适配）
// 约定：客户端发送JSON格式数据 {"id": "1"}
func (c *WsUserController) GetUser(ctx netContext.Context) {
	// 1. 解析客户端消息（仅上下文适配WS场景，业务逻辑无改动）
	type Req struct {
		ID string `json:"id"`
	}
	var req Req
	if err := ctx.BindJson(&req); err != nil {
		c.BaseController.Error(400, "消息格式错误："+err.Error())
		return
	}
	if req.ID == "" {
		c.BaseController.Error(400, "用户ID不能为空")
		return
	}

	// 2. 调用服务层逻辑（完全复用HTTP控制器的服务调用逻辑，无任何修改）
	userInfo, err := c.Service.GetUserById(req.ID)
	if err != nil {
		c.BaseController.Error(500, err.Error())
		return
	}
	if userInfo == nil {
		c.BaseController.Error(500, "用户不存在")
		return
	}

	// 3. 统一响应格式
	c.BaseController.Success(userInfo)
}
```

### 3.3.2 配置WS路由（实现BaseRouter接口）

文件路径：app/api/router/ws_router.go（实现框架BaseRouter接口，统一管理WS路由）

```Plain Text

package router

import (
	"github.com/dfpopp/go-dai/websocket"
	"merchantServer/app/api/controller"
	"merchantServer/app/api/middleware"
)

// ApiRouter API路由（实现框架BaseRouter接口，复用结构体，统一管理多协议路由）
type ApiRouter struct {
	wsUserController *controller.WsUserController // 仅保留WS用户控制器
}

// NewApiRouter 创建API路由实例（供入口文件调用初始化）
func NewApiRouter() *ApiRouter {
	return &ApiRouter{
		wsUserController: controller.NewWsUserController(), // 初始化WS用户控制器
	}
}

// RegisterWSRoutes 注册WebSocket路由（框架约定方法名，用于绑定WS服务路由）
func (r *ApiRouter) RegisterWSRoutes(wsServer *websocket.Server) {
	// 注册WS路由，使用websocket.ToWSHandler转换控制器方法，添加WS专属限流中间件
	wsServer.Register("user.get", websocket.ToWSHandler(r.wsUserController.GetUser), middleware.WsRateLimit(200))
	// 可添加更多WS路由...
}
```

### 3.3.3 配置WS监听器（监听用户上下线）

文件路径：app/api/init/ws/ws.go（WS服务专用，需在服务启动前注册）

以下是完整的监听器实现代码，用于监听用户WS连接的上下线事件并执行自定义业务逻辑：

文件路径：app/api/init/ws/ws.go（WS服务专用，需在服务启动前注册）

以下是完整的监听器实现代码，用于监听用户WS连接的上下线事件并执行自定义业务逻辑：

```Plain Text

package ws

import (
	"encoding/json"
	"github.com/dfpopp/go-dai/logger"
	"github.com/dfpopp/go-dai/websocket"
	"github.com/google/uuid"
)

// CustomConnListener 应用层自定义监听器（实现ConnEventListener接口）
type CustomConnListener struct {
	// 可注入应用层业务服务（如好友服务、消息服务）
	// friendService *service.FriendService
}

// NewCustomConnListener 创建监听器实例
func NewCustomConnListener() *CustomConnListener {
	return &CustomConnListener{
		// 可在此处初始化注入的服务
		// friendService: service.NewFriendService(),
	}
}

// OnConnEvent 上下线事件回调（应用层核心业务逻辑）
func (l *CustomConnListener) OnConnEvent(event websocket.ConnEvent) {
	// 1. 获取连接ID（从事件中直接获取）
	connID := event.ConnInfo.ConnID
	logger.Info("收到连接事件", "eventType", event.EventType, "connID", connID, "triggerTime", event.TriggerTime)

	// 2. 根据事件类型执行自定义逻辑
	switch event.EventType {
	case websocket.EventConnOnline:
		l.handleOnline(event) // 处理上线逻辑
	case websocket.EventConnOffline:
		l.handleOffline(event) // 处理下线逻辑
	}
}

// handleOnline 处理用户上线逻辑（应用层自定义）
func (l *CustomConnListener) handleOnline(event websocket.ConnEvent) {
	connID := event.ConnInfo.ConnID
	clientIP := event.ConnInfo.ClientIP

	// 示例1：打印上线日志
	logger.Info("用户上线", "connID", connID, "clientIP", clientIP, "onlineTime", event.TriggerTime)

	// 示例2：获取连接绑定的用户ID（若应用层已设置）
	//userIDVal, exists := websocket.GetGlobalConnManager().GetConnAttr(connID, "user_id")
	//if exists {
	//	userID, _ := userIDVal.(string)
	//	logger.Info("用户绑定成功", "userID", userID, "connID", connID)
	//
	//	// 示例3：通知该用户的好友（应用层需自行实现好友关系查询）
	//	friends := l.friendService.GetFriendList(userID)
	//	// 组装上线通知
	//	notifyMsg := map[string]interface{}{
	//		"action":     "friend.online",
	//		"request_id": uuid.NewString(),
	//		"data": map[string]interface{}{
	//			"user_id":   userID,
	//			"conn_id":   connID,
	//			"online_at": event.TriggerTime.Format("2006-01-02 15:04:05"),
	//		},
	//	}
	//	msgBytes, _ := json.Marshal(notifyMsg)
	//	// 给好友群发通知（需传入好友的ConnID列表）
	//	websocket.GetGlobalConnManager().Multicast(friends, string(msgBytes))
	//}
}

// handleOffline 处理用户下线逻辑（应用层自定义）
func (l *CustomConnListener) handleOffline(event websocket.ConnEvent) {
	connID := event.ConnInfo.ConnID
	clientIP := event.ConnInfo.ClientIP
	closeReason := event.CloseReason
	// 示例1：打印下线日志
	logger.Info("用户下线", "connID", connID, "clientIP", clientIP, "offlineTime", event.TriggerTime, "reason", closeReason)

	// 示例2：获取连接绑定的用户ID
	//userIDVal, exists := websocket.GetGlobalConnManager().GetConnAttr(connID, "user_id")
	//if exists {
	//	userID, _ := userIDVal.(string)
	//	logger.Info("用户下线", "userID", userID, "connID", connID)
	//
	//	// 示例3：通知该用户的好友
	//	notifyMsg := map[string]interface{}{
	//		"action":     "friend.offline",
	//		"request_id": uuid.NewString(),
	//		"data": map[string]interface{}{
	//			"user_id":    userID,
	//			"conn_id":    connID,
	//			"offline_at": event.TriggerTime.Format("2006-01-02 15:04:05"),
	//			"reason":     closeReason,
	//		},
	//	}
	//	msgBytes, _ := json.Marshal(notifyMsg)
	//	// 给好友群发通知（需传入好友的ConnID列表）
	//	websocket.GetGlobalConnManager().Multicast([]string{}, string(msgBytes))
	//}

	// 示例4：主动关闭连接（应用层可手动调用，此处仅演示）
	// websocket.GetGlobalConnManager().CloseConnByConnID(connID, "manual close")
}
```

### 3.3.4 启动WS服务

修改api.go，添加WS服务启动逻辑（**关键：监听器必须在服务启动前注册**）：

```Plain Text

package main

import (
	"github.com/dfpopp/go-dai/bootstrap"
	"github.com/dfpopp/go-dai/config"
	"github.com/dfpopp/go-dai/websocket"
	"merchantServer/app/api/router"
	"merchantServer/app/api/init/ws" // 引入WS监听器包
	pkgConfig "merchantServer/pkg/config"
)

func init() {
	config.RegisterPostLoadHook(pkgConfig.CustomConfigLoadHook)
}
func main() {
	// 1. 加载配置
	appName := "api"
	if err := config.LoadAppConfig("config/app.json", appName); err != nil {
		panic("加载应用配置失败：" + err.Error())
	}
	if err := config.LoadDatabaseConfig("config/database.json"); err != nil {
		panic("加载数据库配置失败：" + err.Error())
	}

	// 关键：启动WS服务前，注册用户上下线监听器
	// 1. 创建自定义监听器实例
	customListener := ws.NewCustomConnListener()
	// 2. 注册监听器到框架事件总线
	connManager := websocket.GetGlobalConnManager()
	connManager.GetEventBus().Subscribe("custom_conn_listener", customListener)

	// 2. 初始化路由实例（统一管理WS路由）
	apiRouter := router.NewApiRouter()

	// 3. 构建启动配置（启用WS服务）
	bootCfg := &bootstrap.BootConfig{
		AppName:            "api",
		AppConfigPath:      "config/app.json",
		DatabaseConfigPath: "config/database.json",
		EnableServices:     []bootstrap.ServiceType{bootstrap.ServiceTypeWS}, // 启用WS服务
		Router:             apiRouter, // 绑定路由实例，框架自动调用RegisterWSRoutes方法
	}
	// 4. 一键启动服务
	_, err := bootstrap.Boot(bootCfg)
	if err != nil {
		panic("应用启动失败: " + err.Error())
	}
}
```

## 3.4 gRPC服务开发

### 3.4.1 定义Protobuf文件

文件路径：proto/user.proto

### 3.4.2 生成gRPC代码

执行以下命令生成.pb.go和_grpc.pb.go文件（需安装Protobuf编译器及go插件）：

```Plain Text


syntax = "proto3";

package user;
option go_package = "./gen/user;user";

// 用户请求参数（增加商户编码，适配服务层）
message UserRequest {
    string merchant_code = 1; // 商户编码
    uint64 id = 2; // 用户ID
}

// 用户响应数据
message UserData {
    uint64 id = 1;
    string merchant_code = 2; // 商户编码
    string username = 3;
    int32 age = 4;
    string email = 5;
    int64 create_at = 6;
}

// 用户服务响应
message UserResponse {
    int32 code = 1; // 状态码（0成功，非0失败）
    string msg = 2; // 提示信息
    UserData data = 3; // 响应数据
}

// 用户服务接口定义
service UserService {
    rpc GetUser(UserRequest) returns (UserResponse);
}
```

```bash

# 生成代码到gen/user目录
protoc --go_out=./gen/user --go_opt=paths=source_relative \
  --go-grpc_out=./gen/user --go-grpc_opt=paths=source_relative \
  proto/user.proto
```

### 3.4.3 编写gRPC控制器

文件路径：app/api/controller/grpc_user_controller.go

```Plain Text

package controller

import (
	"context"
	"github.com/dfpopp/go-dai/base"
	"merchantServer/app/api/service/user"
	"merchantServer/gen/user" // 引入生成的gRPC代码包
)

// GrpcUserController gRPC用户控制器（最小改动适配标准gRPC，核心业务逻辑复用服务层）
type GrpcUserController struct {
	*base.BaseController                // 继承基础控制器，复用公共能力
	user.UnimplementedUserServiceServer // 嵌入gRPC默认实现，保证兼容性
	Service *user.Service               // 复用用户服务，服务层逻辑完全无改动
}

// NewGrpcUserController 创建gRPC控制器实例（初始化服务依赖）
func NewGrpcUserController() *GrpcUserController {
	return &GrpcUserController{
		BaseController: &base.BaseController{},
		Service:        user.NewUserService(), // 复用同一服务层，无需重复开发
	}
}

// GetUser 实现gRPC的UserService接口（仅适配gRPC上下文和请求/响应格式，业务逻辑复用服务层）
func (c *GrpcUserController) GetUser(ctx context.Context, req *user.UserRequest) (*user.UserResponse, error) {
	// 1. 校验gRPC请求参数（仅参数格式适配，校验逻辑与HTTP/WS一致）
	userId := req.GetId()
	if userId == "" {
		// 适配gRPC的错误响应格式
		return &user.UserResponse{
			Code: 400,
			Msg:  "用户ID不能为空",
		}, nil
	}

	// 2. 调用服务层逻辑（完全复用HTTP/WS的服务调用逻辑，无任何修改）
	userInfo, err := c.Service.GetUserById(userId)
	if err != nil {
		return &user.UserResponse{
			Code: 500,
			Msg:  err.Error(),
		}, nil
	}
	if userInfo == nil {
		return &user.UserResponse{
			Code: 500,
			Msg:  "用户不存在",
		}, nil
	}

	// 3. 适配gRPC响应格式（仅数据格式转换，无业务逻辑改动）
	return &user.UserResponse{
		Code:     0,
		Msg:      "操作成功",
		Data: &user.UserData{
			Id:       userInfo.ID,
			Username: userInfo.Username,
			Age:      int32(userInfo.Age),
			Email:    userInfo.Email,
			CreateAt: userInfo.CreateAt,
		},
	}, nil
}
```

### 3.4.4 配置gRPC路由（实现BaseRouter接口）

文件路径：app/api/router/grpc_router.go（实现框架BaseRouter接口，统一管理gRPC路由）

```Plain Text

package router

import (
	"google.golang.org/grpc"
	"merchantServer/app/api/controller"
	"merchantServer/gen/user" // 引入生成的user服务gRPC代码
)

// ApiRouter API路由（实现框架BaseRouter接口，复用结构体，统一管理多协议路由）
type ApiRouter struct {
	grpcUserController *controller.GrpcUserController // 仅保留gRPC用户控制器
}

// NewApiRouter 创建API路由实例（供入口文件调用初始化）
func NewApiRouter() *ApiRouter {
	return &ApiRouter{
		grpcUserController: controller.NewGrpcUserController(), // 初始化gRPC用户控制器
	}
}

// RegisterGRPCRoutes 注册gRpc路由（框架约定方法名，用于绑定gRPC服务路由）
func (r *ApiRouter) RegisterGRPCRoutes(grpcServer *grpc.Server) {
	// 使用标准gRPC注册方法，将用户控制器实现注册到gRPC服务
	user.RegisterUserServiceServer(grpcServer, r.grpcUserController)
	// 可添加更多gRPC服务注册...
}
```

通过该分层设计，当需要新增或切换服务协议时，仅需开发/修改对应协议的控制器适配代码，服务层和模型层无需任何改动，实现“一次开发，多协议复用”。

### 3.4.5 启动gRPC服务

文件路径：api.go（支持单独启动gRPC服务，或与HTTP/WS服务同时启动）

```Plain Text

package main

import (
	"github.com/dfpopp/go-dai/bootstrap"
	"github.com/dfpopp/go-dai/config"
	"google.golang.org/grpc"
	"merchantServer/app/api/router"
	pkgConfig "merchantServer/pkg/config"
)

func init() {
	config.RegisterPostLoadHook(pkgConfig.CustomConfigLoadHook)
}
func main() {
	// 1. 加载配置
	appName := "api"
	if err := config.LoadAppConfig("config/app.json", appName); err != nil {
		panic("加载应用配置失败：" + err.Error())
	}
	if err := config.LoadDatabaseConfig("config/database.json"); err != nil {
		panic("加载数据库配置失败：" + err.Error())
	}

	// 2. 初始化路由实例（统一管理gRPC路由）
	apiRouter := router.NewApiRouter()

	// 3. 构建启动配置（单独启用gRPC服务）
	bootCfg := &bootstrap.BootConfig{
		AppName:            "api",
		AppConfigPath:      "config/app.json",
		DatabaseConfigPath: "config/database.json",
		EnableServices:     []bootstrap.ServiceType{bootstrap.ServiceTypeGRPC}, // 仅启用gRPC服务
		Router:             apiRouter, // 绑定路由实例，框架自动调用RegisterGRPCRoutes方法
		// 自定义gRPC服务配置（可选）
		GrpcServerOpts: []grpc.ServerOption{
			grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 最大接收消息大小4MB
		},
	}
	// 4. 一键启动服务
	_, err := bootstrap.Boot(bootCfg)
	if err != nil {
		panic("应用启动失败: " + err.Error())
	}
}
```

# 4. 核心模块详解

## 4.1 路由模块（Router）

go-Dai框架的路由模块通过实现BaseRouter接口统一管理，支持路由分组、中间件绑定等能力，以下是贴合本框架的示例：

### 4.1.1 路由分组示例（HTTP）

```Plain Text


// app/api/router/http_router.go（本框架标准路由分组实现）
package router

import (
    "github.com/dfpopp/go-dai/http"
    "merchantServer/app/api/controller"
    "merchantServer/app/api/middleware"
)

// ApiRouter 实现框架BaseRouter接口
type ApiRouter struct {
    userController *controller.UserController
}

func NewApiRouter() *ApiRouter {
    return &ApiRouter{
        userController: controller.NewUserController(),
    }
}

// RegisterHTTPRoutes 注册HTTP路由（框架约定方法）
func (r *ApiRouter) RegisterHTTPRoutes(server *http.Server) {
    // 1. 路由分组：/api/v1（统一前缀）
    v1Group := server.Group("/api/v1")
    {
        // 分组绑定全局中间件（如日志+认证）
        v1Group.Use(middleware.LogMiddleware, middleware.AuthMiddleware)

        // 2. 子分组：用户相关路由
        userGroup := v1Group.Group("/user")
        {
            // 注册路由：/api/v1/user/:id（GET）
            userGroup.GET(":id", http.ToHTTPHandler(r.userController.GetUser), middleware.RateLimit(100))
            // 注册路由：/api/v1/user（POST）
            userGroup.POST("", http.ToHTTPHandler(r.userController.CreateUser))
        }

        // 3. 子分组：订单相关路由（示例，复用控制器注入方式）
        orderController := controller.NewOrderController()
        orderGroup := v1Group.Group("/order")
        {
            orderGroup.GET(":id", http.ToHTTPHandler(orderController.GetOrderInfo))
        }
    }
}
```

### 4.1.2 路由参数解析

框架的netContext上下文对象通过`GetRequestInfo()`方法获取请求相关信息，仅支持Query、PostForm、BindJSON获取数据，以下是贴合框架实际功能的参数解析示例：

```Plain Text


// 1. 路径参数解析（对应路由：/api/v1/user/:id）
// 通过GetRequestInfo().GetPath()获取完整路径，自行解析路径参数（框架不提供Param相关方法）
func (c *UserController) GetUser(ctx netContext.Context) {
    // 获取完整请求路径
    path := ctx.GetRequestInfo().GetPath()
    // 解析路径中的id参数（示例：从路径/api/v1/user/123中提取123）
    // 实际项目中可封装路径参数解析工具函数，此处简化演示
    userIdStr := strings.TrimPrefix(path, "/api/v1/user/")
    userId, err := strconv.ParseUint(userIdStr, 10, 64)
    if err != nil || userId <= 0 {
        c.Error(400, "用户ID格式错误")
        return
    }
}

// 2. 查询参数解析（对应路由：/api/v1/user?name=test&age=20）
// 使用框架支持的GetQuery方法
func (c *UserController) SearchUser(ctx netContext.Context) {
    // 获取查询参数（字符串类型）
    userName := ctx.GetQuery("name")
    // 获取查询参数（整数类型，需自行转换）
    ageStr := ctx.GetQuery("age")
    age := 18 // 默认值
    if ageStr != "" {
        ageInt, err := strconv.Atoi(ageStr)
        if err == nil {
            age = ageInt
        }
    }
}

// 3. 表单参数解析（POST x-www-form-urlencoded）
// 使用框架支持的PostForm方法
func (c *UserController) CreateUser(ctx netContext.Context) {
    // 单个表单参数获取
    username := ctx.PostForm("username")
    ageStr := ctx.PostForm("age")
    age := 0
    if ageStr != "" {
        age, _ = strconv.Atoi(ageStr)
    }

    // 批量绑定到结构体（使用框架支持的BindJSON，表单参数需先转为JSON或自行封装BindForm）
    type CreateUserReq struct {
        Username string `json:"username"`
        Age      int    `json:"age"`
        Email    string `json:"email"`
    }
    var req CreateUserReq
    // 若前端提交表单，可先通过PostForm获取所有参数组装为结构体，再调用BindJSON（或框架支持的表单绑定方式）
    // 此处演示适配BindJSON的用法，实际可根据框架表单绑定能力调整
    req.Username = username
    req.Age = age
    req.Email = ctx.PostForm("email")
}

// 4. JSON参数解析（POST application/json，WS消息也支持此方法）
// 使用框架支持的BindJSON方法
func (c *WsUserController) GetUser(ctx netContext.Context) {
    type Req struct {
        ID string `json:"id"`
    }
    var req Req
    if err := ctx.BindJson(&req); err != nil {
        c.Error(400, "JSON格式错误")
        return
    }
    // 解析ID为uint64
    userId, err := strconv.ParseUint(req.ID, 10, 64)
    if err != nil || userId <= 0 {
        c.Error(400, "用户ID格式错误")
        return
    }
}

// 5. 其他请求信息获取（请求方式、IP、请求头）
func (c *UserController) TestRequestInfo(ctx netContext.Context) {
    // 获取请求方式（GET/POST等）
    method := ctx.GetRequestInfo().GetMethod()
    // 获取客户端IP
    clientIP := ctx.GetRequestInfo().GetClientIP()
    // 获取请求头信息
    token := ctx.GetRequestInfo().GetHeader("Authorization")
    fmt.Printf("请求方式：%s，客户端IP：%s，Authorization：%s\n", method, clientIP, token)
}
```

## 4.2 数据库模块（Database）

数据库模块支持多引擎兼容，提供统一的链式调用API，语法贴近PHP的Eloquent ORM，降低数据库操作学习成本。

### 4.2.1 核心操作示例（MySQL）

```go

// 1. 新增数据
user := &model.User{Username: "test", Age: 20, Email: "test@example.com"}
result := database.DB().Model(user).Create()
if result.Error != nil {
// 错误处理
}
fmt.Println("新增用户ID：", user.ID)

// 2. 更新数据
result = database.DB().Model(&model.User{}).Where("id = ?", 1).Update(map[string]interface{}{
"age": 21,
})

// 3. 条件查询
var users []*model.User
database.DB().Model(&model.User{}).
Where("age > ?", 18).
OrderBy("create_at DESC").
Limit(10).
Find(&users)

// 4. 事务操作
tx := database.DB().Begin()
defer func() {
if r := recover(); r != nil {
tx.Rollback()
}
}()
if err := tx.Error; err != nil {
// 事务启动失败
}
// 事务内操作
tx.Model(&model.User{}).Create(&user1)
tx.Model(&model.User{}).Create(&user2)
tx.Commit()
```

### 4.2.2 多数据库引擎切换

通过配置文件区分不同数据库引擎，使用时指定即可：

```go

// 操作MySQL（默认引擎，配置在database.json的default字段）
database.DB().Model(&model.User{}).First()

// 操作Redis
redisClient := database.Redis("default") // default对应database.json中redis的配置
redisClient.Set("key", "value", 3600)

// 操作ElasticSearch
esClient := database.ES("default")
esClient.Index("users").Id("1").BodyJson(user).Do(context.Background())
```

## 4.3 中间件模块（Middleware）

框架支持HTTP/WS/gRPC通用的中间件机制，可用于请求认证、日志记录、限流、跨域处理等场景。中间件支持全局注册、路由分组注册、单个路由注册。

### 4.3.1 自定义中间件示例（日志记录）

```go

// app/api/middleware/log_middleware.go
package middleware

import (
	"fmt"
	"time"
	"github.com/dfpopp/go-dai/context"
)

// LogMiddleware 日志中间件
func LogMiddleware(ctx *context.HttpContext) {
	// 前置处理：记录请求开始时间
	startTime := time.Now()

	// 执行下一个中间件/控制器
	ctx.Next()

	// 后置处理：记录请求信息
	endTime := time.Now()
	fmt.Printf("请求路径：%s，方法：%s，耗时：%v，状态码：%d\n",
		ctx.Request.URL.Path,
		ctx.Request.Method,
		endTime.Sub(startTime),
		ctx.Response.StatusCode)
}
```

### 4.3.2 中间件注册

```go

// 1. 全局注册（所有HTTP请求生效）
app.HttpGlobalMiddleware(middleware.LogMiddleware)

// 2. 路由分组注册（仅该分组生效）
v1 := r.Group("/api/v1")
v1.Use(middleware.LogMiddleware, middleware.AuthMiddleware)

// 3. 单个路由注册（仅该路由生效）
r.GET("/user/:id", userCtrl.GetUser, middleware.LogMiddleware)
```

## 4.4 配置模块（Config）

配置模块支持JSON格式配置文件，支持多环境（开发、测试、生产）配置切换，支持自定义配置读取钩子。

### 4.4.1 配置文件示例（config/app.json）

```json

{
  "app_name": "go-dai-example",
  "env": "dev", // 环境：dev/test/prod
  "http": {
    "port": 8080,
    "read_timeout": 30,
    "write_timeout": 30
  },
  "ws": {
    "port": 8081,
    "max_conn": 10000
  },
  "grpc": {
    "port": 8082,
    "max_recv_msg_size": 4194304
  }
}
```

### 4.4.2 配置读取方式

```go

import "github.com/dfpopp/go-dai/config"

// 读取HTTP端口
httpPort := config.GetInt("http.port")

// 读取字符串配置
appName := config.GetString("app_name")

// 读取自定义配置（config/custom.json）
customVal := config.GetString("custom.key")

// 配置钩子：自定义配置读取逻辑
config.RegisterHook(func() error {
// 此处可添加配置加密解密、动态配置拉取等逻辑
return nil
})
```

# 5. 进阶配置与扩展

## 5.1 多应用配置

框架支持多应用部署，通过在app目录下创建多个应用目录（如api、admin），并在配置文件中区分应用配置：

```text

├─ app/
│  ├─ api/       接口应用
│  └─ admin/     管理后台应用
├─ config/
│  ├─ app.json
│  ├─ api.app.json  api应用专属配置
│  └─ admin.app.json  admin应用专属配置
```

启动多应用：

```go

// 启动api应用
app.RunApp("api")

// 同时启动多个应用
app.RunApps([]string{"api", "admin"})
```

## 5.2 自定义日志

框架默认将日志输出到runtime/log目录，支持自定义日志输出格式、级别、输出方式（文件/控制台）：

```go

import "github.com/dfpopp/go-dai/log"

// 配置日志级别（debug/info/warn/error）
log.SetLevel(log.DebugLevel)

// 自定义日志输出格式
log.SetFormatter(&log.TextFormatter{
TimestampFormat: "2006-01-02 15:04:05",
FullTimestamp:   true,
})

// 输出日志
log.Debug("调试日志")
log.Info("信息日志")
log.Warn("警告日志")
log.Error("错误日志")
```

## 5.3 框架扩展

框架支持自定义扩展，可通过注册钩子、替换默认实现等方式扩展核心能力：

- 配置钩子：RegisterHook 用于扩展配置读取逻辑

- 数据库驱动扩展：通过 database.RegisterDriver 注册自定义数据库驱动

- 中间件扩展：自定义中间件实现特定业务需求（如链路追踪、监控等）

# 6. 常见问题与解决方案

## 6.1 环境依赖问题

Q：安装框架后启动失败，提示Go版本过低？
A：请升级Go版本至1.24及以上，框架部分特性依赖Go 1.24的新API。

## 6.2 数据库连接问题

Q：无法连接数据库，提示认证失败？
A：检查config/database.json中的数据库配置（地址、端口、用户名、密码）是否正确，确保数据库服务正常运行且网络可通。

## 6.3 gRPC代码生成问题

Q：执行protoc命令生成代码失败？
A：请确保已正确安装Protobuf编译器（protoc）和Go语言插件（protoc-gen-go、protoc-gen-go-grpc），并将插件添加到系统环境变量中。

## 6.4 多协议服务端口冲突

Q：同时启动HTTP和WS服务时提示端口被占用？
A：修改config/app.json中http和ws的port配置，确保不同服务使用不同端口。

# 7. 贡献与反馈

- GitHub仓库：[https://github.com/dfpopp/go-dai](https://github.com/dfpopp/go-dai)（示例地址，需替换为实际仓库）

- Issue提交：如遇bug或需求，可在GitHub仓库提交Issue

- 贡献代码：欢迎Fork仓库，提交Pull Request参与框架开发

- 交流群：XXX（可添加框架作者获取交流群信息）

# 8. 许可证

本框架采用 MIT 开源许可证，允许自由使用、修改和分发，详情见LICENSE文件。
> （注：文档部分内容可能由 AI 生成）