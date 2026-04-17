# go-Dai框架详细文档

## 1. 框架概述

go-Dai 是一款集多协议、多数据库引擎于一体的 Go 服务开发框架，特别沿用 PHP 主流框架设计思路，更适配 PHP 程序员的开发习惯，降低 Go 语言技术栈的学习与迁移成本；框架原生支持 HTTP、WebSocket、gRPC 多种服务协议，兼容 MySQL、MongoDB、Redis、ElasticSearch 等主流数据库引擎，且数据库操作支持简洁高效的链式调用；采用经典 MVC 架构设计，实现路由（Router）、控制器（Controller）、服务（Service）、模型（Model）全链路高一致性，为多协议服务开发提供统一、可靠的技术基座。

### 1.1 设计理念

- **PHP友好**：沿用 Laravel/ThinkPHP 等主流 PHP 框架的设计思路，降低 PHP 开发者转 Go 的学习成本
- **多协议统一**：一套编码规范即可开发 HTTP、WebSocket、gRPC 多种服务接口
- **数据库操作便捷**：支持简洁高效的链式调用，贴近 PHP ORM 体验
- **开箱即用**：内置完善的默认配置、统一错误处理、常用中间件等基础能力
- **服务切换成本低**：基于 MVC 全链路高一致性设计，核心业务逻辑（Model/Service）无需改动

### 1.2 技术架构

![go-Dai框架架构图](https://trae-api-cn.mchost.guru/api/ide/v1/text_to_image?prompt=go%20language%20framework%20architecture%20diagram%20showing%20MVC%20pattern%20with%20HTTP%2C%20WebSocket%2C%20gRPC%20protocols%20and%20multiple%20database%20engines%20(MySQL%2C%20MongoDB%2C%20Redis%2C%20ElasticSearch)%2C%20clean%20technical%20diagram%20with%20layers%20and%20components&image_size=landscape_16_9)

## 2. 环境要求与安装

### 2.1 环境要求

- Go 1.24+（建议使用1.24及以上稳定版本，低版本兼容性未充分测试）
- Protobuf（如需使用gRPC）
- 对应数据库客户端（MySQL/MongoDB/Redis/ElasticSearch等）

### 2.2 安装方式

```bash
go get github.com/dfpopp/go-dai
```

## 3. 核心概念

### 3.1 MVC架构

- **模型（Model）**：映射数据实体，负责数据的CRUD操作
- **控制器（Controller）**：处理请求分发、参数校验、响应封装
- **服务（Service）**：封装核心业务逻辑，是多协议复用的核心
- **路由（Router）**：配置接口路由，支持HTTP、WebSocket、gRPC三种协议

### 3.2 依赖注入

框架支持依赖注入模式，通过构造函数或结构体字段注入依赖，提高代码的可测试性和可维护性。

### 3.3 中间件

中间件用于请求拦截与增强，支持HTTP、WebSocket、gRPC三种协议，可用于认证、日志记录、限流等场景。

### 3.4 上下文

框架提供统一的上下文接口（netContext.Context），封装了请求信息和响应方法，支持三种协议的上下文适配。

## 4. 快速上手

### 4.1 项目结构

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

### 4.2 配置文件

#### 4.2.1 应用配置（config/app.json）

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

#### 4.2.2 数据库配置（config/database.json）

```json
{
  "mysql": {
    "default": {
      "host": "127.0.0.1",
      "port": 3306,
      "user": "root",
      "password": "password",
      "database": "test",
      "charset": "utf8mb4",
      "max_idle_conns": 10,
      "max_open_conns": 100,
      "conn_max_lifetime": 60
    }
  },
  "mongodb": {
    "default": {
      "uri": "mongodb://localhost:27017",
      "database": "test",
      "max_pool_size": 100
    }
  },
  "redis": {
    "default": {
      "addr": "127.0.0.1:6379",
      "password": "",
      "db": 0,
      "pool_size": 10
    }
  },
  "elasticsearch": {
    "default": {
      "urls": ["http://localhost:9200"],
      "username": "",
      "password": ""
    }
  }
}
```

### 4.3 入口文件

```go
package main

import (
	"github.com/dfpopp/go-dai/bootstrap"
	"github.com/dfpopp/go-dai/config"
	"merchantServer/app/api/router"
)

func main() {
	// 1. 构建启动配置（启用HTTP服务）
	bootCfg := &bootstrap.BootConfig{
		AppName:            "api",
		AppConfigPath:      "config/app.json",
		DatabaseConfigPath: "config/database.json",
		EnableServices:     []bootstrap.ServiceType{bootstrap.ServiceTypeHTTP},
		Router:             router.NewApiRouter(),
	}
	// 2. 一键启动服务
	_, err := bootstrap.Boot(bootCfg)
	if err != nil {
		panic("应用启动失败: " + err.Error())
	}
}
```

## 5. 核心模块详解

### 5.1 路由模块

路由模块通过实现BaseRouter接口统一管理，支持路由分组、中间件绑定等能力。

#### 5.1.1 实现BaseRouter接口

```go
package router

import (
	"github.com/dfpopp/go-dai/http"
	"github.com/dfpopp/go-dai/grpc"
	"github.com/dfpopp/go-dai/websocket"
	"merchantServer/app/api/controller"
)

// ApiRouter API路由（实现框架BaseRouter接口）
type ApiRouter struct {
	userController *controller.UserController
}

// NewApiRouter 创建API路由实例
func NewApiRouter() *ApiRouter {
	return &ApiRouter{
		userController: controller.NewUserController(),
	}
}

// RegisterHTTPRoutes 注册HTTP路由
func (r *ApiRouter) RegisterHTTPRoutes(server *http.Server) {
	server.GET("/user/:id", http.ToHTTPHandler(r.userController.GetUser))
}

// RegisterWSRoutes 注册WebSocket路由
func (r *ApiRouter) RegisterWSRoutes(wsServer *websocket.Server) {
	wsServer.Register("user.get", websocket.ToWSHandler(r.userController.GetUser))
}

// RegisterGRPCRoutes 注册gRPC路由
func (r *ApiRouter) RegisterGRPCRoutes(grpcServer *grpc.Server) {
	// gRPC路由注册
}
```

#### 5.1.2 路由分组

```go
// RegisterHTTPRoutes 注册HTTP路由
func (r *ApiRouter) RegisterHTTPRoutes(server *http.Server) {
	// 路由分组：/api/v1
	v1Group := server.Group("/api/v1")
	{
		// 分组绑定全局中间件
		v1Group.Use(middleware.LogMiddleware, middleware.AuthMiddleware)

		// 子分组：用户相关路由
		userGroup := v1Group.Group("/user")
		{
			userGroup.GET(":id", http.ToHTTPHandler(r.userController.GetUser))
			userGroup.POST("", http.ToHTTPHandler(r.userController.CreateUser))
		}
	}
}
```

### 5.2 控制器模块

控制器负责处理请求分发、参数校验、响应封装，不直接操作数据，所有业务逻辑通过调用服务层实现。

#### 5.2.1 HTTP控制器

```go
package controller

import (
	"github.com/dfpopp/go-dai/base"
	"github.com/dfpopp/go-dai/netContext"
	"merchantServer/app/api/service"
	"strconv"
	"strings"
)

// UserController HTTP用户控制器
type UserController struct {
	*base.BaseController
	Service *service.UserService
}

// NewUserController 创建控制器实例
func NewUserController() *UserController {
	return &UserController{
		BaseController: &base.BaseController{},
		Service:        service.NewUserService(),
	}
}

// GetUser 处理用户查询HTTP请求
func (c *UserController) GetUser(ctx netContext.Context) {
	// 1. 获取路径参数
	path := ctx.GetRequestInfo().GetPath()
	userIdStr := strings.TrimPrefix(path, "/user/")
	userId, err := strconv.ParseUint(userIdStr, 10, 64)
	if err != nil {
		c.Error(400, "用户ID格式错误")
		return
	}

	// 2. 调用服务层逻辑
	userInfo, err := c.Service.GetUserById(userId)
	if err != nil {
		c.Error(500, err.Error())
		return
	}

	// 3. 统一响应格式
	c.Success(userInfo)
}
```

#### 5.2.2 WebSocket控制器

```go
package controller

import (
	"github.com/dfpopp/go-dai/base"
	"github.com/dfpopp/go-dai/netContext"
	"merchantServer/app/api/service"
)

// WsUserController WebSocket用户控制器
type WsUserController struct {
	*base.BaseController
	Service *service.UserService
}

// NewWsUserController 创建WS控制器实例
func NewWsUserController() *WsUserController {
	return &WsUserController{
		BaseController: &base.BaseController{},
		Service:        service.NewUserService(),
	}
}

// GetUser 处理WS用户查询请求
func (c *WsUserController) GetUser(ctx netContext.Context) {
	type Req struct {
		ID string `json:"id"`
	}
	var req Req
	if err := ctx.BindJson(&req); err != nil {
		c.Error(400, "消息格式错误："+err.Error())
		return
	}

	// 调用服务层逻辑
	userInfo, err := c.Service.GetUserById(req.ID)
	if err != nil {
		c.Error(500, err.Error())
		return
	}

	// 统一响应格式
	c.Success(userInfo)
}
```

#### 5.2.3 gRPC控制器

```go
package controller

import (
	"context"
	"github.com/dfpopp/go-dai/base"
	"merchantServer/app/api/service"
	"merchantServer/gen/user"
)

// GrpcUserController gRPC用户控制器
type GrpcUserController struct {
	*base.BaseController
	user.UnimplementedUserServiceServer
	Service *service.UserService
}

// NewGrpcUserController 创建gRPC控制器实例
func NewGrpcUserController() *GrpcUserController {
	return &GrpcUserController{
		BaseController: &base.BaseController{},
		Service:        service.NewUserService(),
	}
}

// GetUser 实现gRPC的UserService接口
func (c *GrpcUserController) GetUser(ctx context.Context, req *user.UserRequest) (*user.UserResponse, error) {
	// 调用服务层逻辑
	userInfo, err := c.Service.GetUserById(req.GetId())
	if err != nil {
		return &user.UserResponse{
			Code: 500,
			Msg:  err.Error(),
		}, nil
	}

	// 适配gRPC响应格式
	return &user.UserResponse{
		Code:     0,
		Msg:      "操作成功",
		Data: &user.UserData{
			Id:       userInfo.ID,
			Username: userInfo.Username,
			Age:      int32(userInfo.Age),
			Email:    userInfo.Email,
		},
	}, nil
}
```

### 5.3 服务模块

服务层封装核心业务逻辑，是多协议复用的核心，仅通过模型层操作数据，与具体协议完全解耦。

```go
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
		userModel: model.NewUser(),
	}
}

// GetUserById 调用模型层查询用户信息
func (s *UserService) GetUserById(id uint64) (*model.User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("用户ID非法")
	}
	return s.userModel.GetById(id)
}
```

### 5.4 模型模块

模型层映射数据实体，负责数据的CRUD操作，仅被服务层调用，不与控制器直接交互。

#### 5.4.1 MySQL模型

```go
package model

import (
	"github.com/dfpopp/go-dai/db/mysql"
)

// User 用户数据模型
type User struct {
	ID       uint64 `json:"id" db:"id,primaryKey"`
	Username string `json:"username" db:"username"`
	Age      int    `json:"age" db:"age"`
	Email    string `json:"email" db:"email"`
}

// TableName 指定模型对应的数据表名
func (u *User) TableName() string {
	return "users"
}

// NewUser 创建用户模型实例
func NewUser() *User {
	return &User{}
}

// GetById 根据ID查询用户信息
func (u *User) GetById(id uint64) (*User, error) {
	result := mysql.DB().Model(u).Where("id = ?", id).First()
	if result.Error != nil {
		return nil, result.Error
	}
	return u, nil
}
```

#### 5.4.2 MongoDB模型

```go
package model

import (
	"context"
	"github.com/dfpopp/go-dai/db/mongoDb"
)

// User 用户数据模型
type User struct {
	ID       string `json:"_id,omitempty" bson:"_id,omitempty"`
	Username string `json:"username" bson:"username"`
	Age      int    `json:"age" bson:"age"`
	Email    string `json:"email" bson:"email"`
}

// CollectionName 指定模型对应的集合名
func (u *User) CollectionName() string {
	return "users"
}

// GetById 根据ID查询用户信息
func (u *User) GetById(ctx context.Context, id string) (*User, error) {
	result := mongoDb.DB().Collection(u.CollectionName()).FindOne(ctx, map[string]interface{}{"_id": id})
	if result.Err() != nil {
		return nil, result.Err()
	}
	err := result.Decode(u)
	if err != nil {
		return nil, err
	}
	return u, nil
}
```

### 5.5 数据库模块

数据库模块支持多引擎兼容，提供统一的链式调用API，语法贴近PHP的Eloquent ORM。

#### 5.5.1 MySQL操作

```go
// 1. 新增数据
user := &model.User{Username: "test", Age: 20, Email: "test@example.com"}
result := mysql.DB().Model(user).Create()
if result.Error != nil {
	// 错误处理
}

// 2. 更新数据
result = mysql.DB().Model(&model.User{}).Where("id = ?", 1).Update(map[string]interface{}{
	"age": 21,
})

// 3. 条件查询
var users []*model.User
mysql.DB().Model(&model.User{}).
	Where("age > ?", 18).
	OrderBy("create_at DESC").
	Limit(10).
	Find(&users)

// 4. 事务操作
tx := mysql.DB().Begin()
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

#### 5.5.2 MongoDB操作

```go
// 1. 新增数据
user := &model.User{Username: "test", Age: 20, Email: "test@example.com"}
_, err := mongoDb.DB().Collection("users").InsertOne(ctx, user)

// 2. 更新数据
_, err = mongoDb.DB().Collection("users").UpdateOne(
	ctx,
	map[string]interface{}{"_id": "1"},
	map[string]interface{}{"$set": map[string]interface{}{"age": 21}},
)

// 3. 条件查询
cursor, err := mongoDb.DB().Collection("users").Find(
	ctx,
	map[string]interface{}{"age": map[string]interface{}{">$": 18}},
)
var users []*model.User
cursor.All(ctx, &users)

// 4. 聚合查询
pipeline := mongoDb.Pipeline{}
pipeline.Match(map[string]interface{}{"age": map[string]interface{}{">$": 18}})
pipeline.Group(map[string]interface{}{
	"_id":  "$gender",
	"count": map[string]interface{}{"$sum": 1},
})
cursor, err = mongoDb.DB().Collection("users").Aggregate(ctx, pipeline)
```

#### 5.5.3 Redis操作

```go
// 1. 设置键值
redisDb.Client().Set("key", "value", 3600)

// 2. 获取键值
val, err := redisDb.Client().Get("key").Result()

// 3. 哈希操作
redisDb.Client().HSet("user:1", "username", "test")
redisDb.Client().HSet("user:1", "age", 20)

// 4. 列表操作
redisDb.Client().LPush("list", "item1")
redisDb.Client().LPush("list", "item2")

// 5. 集合操作
redisDb.Client().SAdd("set", "member1")
redisDb.Client().SAdd("set", "member2")
```

#### 5.5.4 ElasticSearch操作

```go
// 1. 索引文档
user := &model.User{ID: "1", Username: "test", Age: 20, Email: "test@example.com"}
_, err := elasticSearch.Client().Index("users").Id(user.ID).BodyJson(user).Do(ctx)

// 2. 获取文档
result, err := elasticSearch.Client().Get().Index("users").Id("1").Do(ctx)

// 3. 更新文档
_, err = elasticSearch.Client().Update().Index("users").Id("1").
	BodyJson(map[string]interface{}{"doc": map[string]interface{}{"age": 21}}).
	Do(ctx)

// 4. 删除文档
_, err = elasticSearch.Client().Delete().Index("users").Id("1").Do(ctx)

// 5. 搜索
result, err := elasticSearch.Client().Search().Index("users").
	Query(elastic.NewMatchQuery("username", "test")).
	Do(ctx)
```

### 5.6 中间件模块

中间件用于请求拦截与增强，支持HTTP、WebSocket、gRPC三种协议。

#### 5.6.1 HTTP中间件

```go
package middleware

import (
	"time"
	"github.com/dfpopp/go-dai/http"
)

// LogMiddleware 日志中间件
func LogMiddleware() http.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(ctx *http.Context) {
			// 前置处理：记录请求开始时间
			startTime := time.Now()

			// 执行下一个中间件/控制器
			next(ctx)

			// 后置处理：记录请求信息
			endTime := time.Now()
			ctx.Logger.Info("请求路径：", ctx.Request.URL.Path, "，方法：", ctx.Request.Method, "，耗时：", endTime.Sub(startTime))
		}
	}
}

// AuthMiddleware 认证中间件
func AuthMiddleware() http.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(ctx *http.Context) {
			token := ctx.Request.Header.Get("Authorization")
			if token == "" {
				ctx.JSON(401, map[string]interface{}{
					"code": 401,
					"msg":  "未授权",
				})
				return
			}
			// 验证token
			// ...
			next(ctx)
		}
	}
}
```

#### 5.6.2 WebSocket中间件

```go
package middleware

import (
	"github.com/dfpopp/go-dai/websocket"
)

// WSAuthMiddleware WebSocket认证中间件
func WSAuthMiddleware() websocket.MiddlewareFunc {
	return func(next websocket.HandlerFunc) websocket.HandlerFunc {
		return func(ctx *websocket.Context) {
			token := ctx.GetQuery("token")
			if token == "" {
				ctx.Close()
				return
			}
			// 验证token
			// ...
			next(ctx)
		}
	}
}
```

#### 5.6.3 gRPC中间件

```go
package middleware

import (
	"context"
	"time"
	"google.golang.org/grpc"
)

// GRPCLogMiddleware gRPC日志中间件
func GRPCLogMiddleware() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 前置处理：记录请求开始时间
		startTime := time.Now()

		// 执行下一个中间件/处理器
		resp, err := handler(ctx, req)

		// 后置处理：记录请求信息
		endTime := time.Now()
		// 记录日志
		// ...

		return resp, err
	}
}
```

### 5.7 配置模块

配置模块支持JSON格式配置文件，支持多环境（开发、测试、生产）配置切换，支持自定义配置读取钩子。

#### 5.7.1 配置读取

```go
import "github.com/dfpopp/go-dai/config"

// 读取HTTP端口
httpPort := config.GetInt("http.port")

// 读取字符串配置
appName := config.GetString("app_name")

// 读取布尔配置
debugMode := config.GetBool("debug")

// 读取浮点配置
timeout := config.GetFloat("timeout")

// 读取自定义配置
customVal := config.GetString("custom.key")
```

#### 5.7.2 配置钩子

```go
import "github.com/dfpopp/go-dai/config"

// 注册配置加载后钩子
config.RegisterPostLoadHook(func() error {
	// 此处可添加配置加密解密、动态配置拉取等逻辑
	// ...
	return nil
})
```

### 5.8 日志模块

日志模块支持多级别日志输出，支持按环境输出到文件或控制台。

```go
import "github.com/dfpopp/go-dai/logger"

// 输出调试日志
logger.Debug("调试日志", "key", "value")

// 输出信息日志
logger.Info("信息日志", "key", "value")

// 输出警告日志
logger.Warn("警告日志", "key", "value")

// 输出错误日志
logger.Error("错误日志", "key", "value")
```

### 5.9 WebSocket连接管理

WebSocket模块支持连接管理、消息广播、用户绑定等功能。

#### 5.9.1 连接事件监听

```go
package ws

import (
	"github.com/dfpopp/go-dai/logger"
	"github.com/dfpopp/go-dai/websocket"
)

// CustomConnListener 自定义连接监听器
type CustomConnListener struct {}

// NewCustomConnListener 创建监听器实例
func NewCustomConnListener() *CustomConnListener {
	return &CustomConnListener{}
}

// OnConnEvent 上下线事件回调
func (l *CustomConnListener) OnConnEvent(event websocket.ConnEvent) {
	connID := event.ConnInfo.ConnID
	logger.Info("收到连接事件", "eventType", event.EventType, "connID", connID)

	switch event.EventType {
	case websocket.EventConnOnline:
		l.handleOnline(event)
	case websocket.EventConnOffline:
		l.handleOffline(event)
	}
}

// handleOnline 处理上线逻辑
func (l *CustomConnListener) handleOnline(event websocket.ConnEvent) {
	// 处理上线逻辑
	// ...
}

// handleOffline 处理下线逻辑
func (l *CustomConnListener) handleOffline(event websocket.ConnEvent) {
	// 处理下线逻辑
	// ...
}
```

#### 5.9.2 注册监听器

```go
import (
	"github.com/dfpopp/go-dai/websocket"
	"merchantServer/app/api/init/ws"
)

// 注册监听器
customListener := ws.NewCustomConnListener()
connManager := websocket.GetGlobalConnManager()
connManager.GetEventBus().Subscribe("custom_conn_listener", customListener)
```

#### 5.9.3 发送消息

```go
// 发送消息给指定连接
websocket.GetGlobalConnManager().SendToConnID("conn-id", "message")

// 发送消息给指定用户的所有连接
websocket.GetGlobalConnManager().SendToUser("user-id", "message")

// 广播消息给所有连接
websocket.GetGlobalConnManager().Broadcast("message")
```

## 6. 高级功能

### 6.1 优雅停机

框架支持优雅停机，当收到SIGINT或SIGTERM信号时，会先停止接收新请求，等待现有请求处理完成，然后关闭所有服务。

### 6.2 多应用部署

框架支持多应用部署，通过在app目录下创建多个应用目录（如api、admin），并在配置文件中区分应用配置。

```text
├─ app/
│  ├─ api/       接口应用
│  └─ admin/     管理后台应用
├─ config/
│  ├─ app.json
│  ├─ api.app.json  api应用专属配置
│  └─ admin.app.json  admin应用专属配置
```

### 6.3 定时任务

框架支持定时任务，通过BootCron函数启动定时任务。

```go
package main

import (
	"github.com/dfpopp/go-dai/bootstrap"
)

func main() {
	// 构建定时任务配置
	bootCfg := &bootstrap.BootConfig{
		AppName:            "cron",
		AppConfigPath:      "config/app.json",
		DatabaseConfigPath: "config/database.json",
	}
	// 启动定时任务
	err := bootstrap.BootCron(bootCfg)
	if err != nil {
		panic("定时任务启动失败: " + err.Error())
	}

	// 注册定时任务
	// ...
}
```

### 6.4 配置热更新

框架支持配置热更新，当配置文件发生变化时，会自动重新加载配置。

```go
import "github.com/dfpopp/go-dai/config"

// 注册配置变更回调
config.RegisterChangeHook(func() {
	// 配置变更时的处理逻辑
	// ...
})
```

## 7. 最佳实践

### 7.1 代码结构

- **控制器**：仅负责请求分发、参数校验、响应封装，不直接操作数据
- **服务**：封装核心业务逻辑，仅通过模型层操作数据
- **模型**：映射数据实体，负责数据的CRUD操作
- **路由**：统一管理接口路由，支持HTTP、WebSocket、gRPC三种协议

### 7.2 错误处理

- 使用统一的错误响应格式
- 服务层返回具体错误，控制器统一处理错误响应
- 中间件捕获并处理异常，确保服务稳定运行

### 7.3 日志记录

- 使用框架提供的日志工具
- 按环境配置不同的日志级别
- 关键操作必须记录日志
- 日志包含足够的上下文信息，便于排查问题

### 7.4 性能优化

- 使用连接池管理数据库连接
- 合理使用缓存，减少数据库查询
- 避免在循环中进行数据库操作
- 使用批量操作减少网络开销
- 优化SQL查询，添加适当的索引

### 7.5 安全性

- 使用HTTPS协议
- 实现合适的认证和授权机制
- 防止SQL注入、XSS攻击等安全问题
- 对敏感数据进行加密存储
- 合理设置CORS策略

## 8. 常见问题与解决方案

### 8.1 环境依赖问题

**问题**：安装框架时出现依赖冲突
**解决方案**：使用go mod tidy命令整理依赖，确保依赖版本兼容

### 8.2 数据库连接问题

**问题**：数据库连接失败
**解决方案**：检查数据库配置是否正确，确保数据库服务正常运行，检查网络连接是否畅通

### 8.3 路由注册问题

**问题**：路由注册失败
**解决方案**：检查路由注册代码是否正确，确保控制器方法签名符合要求，检查中间件是否正确实现

### 8.4 WebSocket连接问题

**问题**：WebSocket连接失败
**解决方案**：检查WebSocket服务是否正常启动，检查客户端连接地址是否正确，检查防火墙设置

### 8.5 gRPC服务问题

**问题**：gRPC服务启动失败
**解决方案**：检查Protobuf文件是否正确，确保生成的代码与服务定义一致，检查gRPC服务配置是否正确

### 8.6 性能问题

**问题**：服务响应缓慢
**解决方案**：使用性能分析工具定位瓶颈，优化数据库查询，合理使用缓存，减少网络开销

## 9. 总结

go-Dai框架是一款为PHP程序员转Go开发而设计的高性能服务开发框架，它沿用了PHP主流框架的设计思路，同时融合了Go语言的优势，为开发者提供了一套简洁、高效、统一的服务开发解决方案。

通过本文档的学习，您应该已经了解了go-Dai框架的核心功能和使用方法，能够快速上手使用框架开发HTTP、WebSocket、gRPC服务。

框架的设计理念是"一次开发，多协议复用"，通过MVC分层设计和服务层的抽象，实现了业务逻辑的高度复用，降低了多协议开发的学习和维护成本。

我们相信，go-Dai框架将成为您Go语言开发的得力助手，帮助您快速构建高性能、可维护的服务应用。