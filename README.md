# go-Dai 框架

go-Dai 是一款集多协议、多数据库引擎于一体的 Go 服务开发框架，沿用 PHP 主流框架（Laravel/ThinkPHP）的 MVC 分层思想与路由设计规范，专为 PHP 程序员转 Go 开发而设计，显著降低 Go 语言技术栈的学习与迁移成本。

## 核心特性

- **PHP 友好，低迁移成本**：沿用 Laravel/ThinkPHP 等主流 PHP 框架的 MVC 分层思想、路由设计规范及控制器写法
- **多协议统一**：一套编码规范即可开发 HTTP、WebSocket、gRPC 多种服务接口，通过 `netContext.Context` 统一抽象
- **多数据库引擎**：兼容 MySQL、PostgreSQL、ClickHouse、MongoDB、Redis、ElasticSearch，支持简洁高效的链式调用
- **开箱即用**：内置完善的默认配置、统一错误处理、CORS、Recovery、日志等基础能力
- **服务切换成本低**：基于 MVC 全链路高一致性设计，模型与服务层完全脱离协议依赖
- **优雅停机**：内置信号监听与资源清理机制，支持业务层自定义关闭回调

## 适用场景

- PHP 程序员转 Go 开发
- 需要快速落地多协议服务的团队
- 需要统一开发规范的微服务架构
- 需要高性能且保持开发效率的项目
- 多站点、多租户的 SaaS 型业务系统

---

## 目录

1. [环境要求与安装](#环境要求与安装)
2. [项目目录结构](#项目目录结构)
3. [核心架构](#核心架构)
4. [快速上手](#快速上手)
5. [核心模块详解](#核心模块详解)
6. [多协议开发指南](#多协议开发指南)
7. [数据库操作指南](#数据库操作指南)
8. [落地应用案例](#落地应用案例)
9. [常见问题](#常见问题)
10. [文档资源](#文档资源)

---

## 环境要求与安装

### 环境要求

- Go 1.26+（建议使用最新稳定版本）
- Protobuf（如需使用 gRPC）
- 对应数据库客户端（MySQL/PostgreSQL/ClickHouse/MongoDB/Redis/ElasticSearch）

### 安装框架

```bash
go get github.com/dfpopp/go-dai
```

### 依赖说明

框架核心依赖：

- `github.com/go-sql-driver/mysql` - MySQL 驱动
- `go.mongodb.org/mongo-driver/v2` - MongoDB 驱动
- `github.com/go-redis/redis` - Redis 客户端
- `github.com/elastic/go-elasticsearch` - ElasticSearch 客户端
- `github.com/jackc/pgx/v5` - PostgreSQL 驱动（stdlib 模式）
- `github.com/ClickHouse/clickhouse-go/v2` - ClickHouse 驱动（锁定 v2.42.0 适配 go 1.26）
- `github.com/google/uuid` - UUID 生成
- `github.com/gorilla/websocket` - WebSocket 支持

---

## 项目目录结构

### 框架目录结构

```
go-Dai/
├── base/                # 基础模块
│   ├── component.go     # 组件基类（日志能力）
│   ├── config.go        # 配置管理
│   ├── connection.go    # 连接管理接口
│   ├── controller.go    # 控制器基类
│   ├── model.go         # 模型基类
│   ├── router.go        # 路由接口
│   └── service.go       # 服务基类
├── bootstrap/           # 启动模块
│   └── bootstrap.go     # 统一启动入口
├── config/              # 配置模块
│   ├── api.go           # API 配置
│   ├── config.go        # 配置结构定义
│   └── loader.go        # 配置加载器
├── db/                  # 数据库模块
│   ├── db.go            # 数据库统一入口
│   ├── mysql/           # MySQL 支持
│   ├── postgresql/      # PostgreSQL 支持
│   ├── clickhouse/      # ClickHouse 支持
│   ├── mongoDb/         # MongoDB 支持
│   ├── redisDb/         # Redis 支持
│   └── elasticSearch/   # ElasticSearch 支持
├── function/            # 功能函数模块
│   ├── function.go      # 常用工具函数
│   └── regexpVar.go     # 正则表达式工具
├── grpc/                # gRPC 模块
│   ├── context.go       # gRPC 上下文
│   ├── middleware.go    # gRPC 中间件
│   ├── router.go        # gRPC 路由
│   └── server.go        # gRPC 服务器
├── http/                # HTTP 模块
│   ├── context.go       # HTTP 上下文
│   ├── middleware.go    # HTTP 中间件
│   ├── router.go        # HTTP 路由
│   ├── server.go        # HTTP 服务器
│   └── sse.go           # SSE 支持
├── logger/              # 日志模块
│   └── logger.go        # 日志管理
├── netContext/          # 网络上下文
│   └── context.go       # 统一上下文接口
├── websocket/           # WebSocket 模块
│   ├── connEvent.go     # 连接事件
│   ├── connManager.go   # 连接管理器
│   ├── context.go       # WebSocket 上下文
│   ├── middleware.go    # WebSocket 中间件
│   ├── router.go        # WebSocket 路由
│   └── server.go        # WebSocket 服务器
├── go.mod
└── go.sum
```

### 应用项目目录结构

```
your-project/
├── app/                 # 应用根目录
│   └─ api/              # 应用核心目录（支持多应用：api/admin/seller/cron）
│     ├─ controller/     # 控制器目录：处理请求分发
│     │   └─ user/       # 按业务模块拆分
│     │       └─ user.go
│     ├─ service/        # 服务目录：封装业务逻辑
│     │   └─ user/
│     │       └─ user.go
│     ├─ model/          # 模型目录：映射数据实体
│     │   └─ user/
│     │       └─ user.go
│     ├─ router/         # 路由目录：配置接口路由
│     │   └─ router.go
│     ├─ middleware/     # 中间件目录：请求拦截与增强
│     │   └─ http_middleware.go
│     └─ init/           # 初始化目录（仅 WS 服务适用）
│        └─ ws/          # WebSocket 连接事件监听器
├── config/              # 配置文件根目录
│   ├─ app.json          # 应用配置（必选，支持多应用拆分）
│   ├─ database.json     # 数据库引擎配置（必选）
│   └─ custom.json       # 自定义配置（可选）
├── gen/                 # gRPC 专用目录：存放 *.pb.go、*_grpc.pb.go 文件
├── proto/               # gRPC 专用目录：存放 *.proto 文件
├── pkg/                 # 依赖包目录：存放第三方依赖或自定义包
├── runtime/             # 运行时目录：存放日志文件
│   └─ log/
│     └─ api/            # 按应用名分目录
├── scripts/             # 脚本目录（可选）
├── api.go               # 入口文件（多应用场景可配置多个：api.go/cron.go/seller.go）
└── go.mod               # 项目依赖配置文件
```

---

## 核心架构

### MVC 分层设计

框架采用严格的 MVC 分层设计，各层职责清晰，禁止跨层调用：

```
┌─────────────────────────────────────────┐
│           Controller (控制器层)           │
│  - 请求分发、参数校验、响应封装             │
│  - 仅调用服务层，不直接操作数据             │
│  - 继承 base.BaseController              │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│            Service (服务层)               │
│  - 封装核心业务逻辑                        │
│  - 多协议复用的核心                        │
│  - 仅通过模型层操作数据                    │
│  - 继承 base.BaseService                 │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│             Model (模型层)                │
│  - 映射数据实体                           │
│  - 负责 CRUD 操作                         │
│  - 仅被服务层调用                          │
│  - 继承 base.BaseModel                   │
└─────────────────────────────────────────┘
```

**分层调用规则**：

- `Controller → Service → Model → Database` 单向调用
- Model 不能调用 Service
- Service 不能调用 Controller
- Controller 不能直接调用 Model（应通过 Service）

### 多协议统一架构

框架通过统一的上下文接口（`netContext.Context`）实现多协议统一：

```
┌─────────────────────────────────────────┐
│         netContext.Context 接口          │
│  - GetRequestInfo() RequestInfo          │
│  - JSON/String/Query/PostForm            │
│  - BindJSON/SetParam/GetParam            │
│  - Set/Get 上下文存储                     │
│  - RegisterCleanupFunc 清理回调           │
└─────────────────────────────────────────┘
          ↑           ↑           ↑
          │           │           │
    ┌─────┴───┐  ┌────┴────┐  ┌───┴─────┐
    │ HTTP    │  │WebSocket│  │  gRPC   │
    │ Context │  │ Context │  │ Context │
    └─────────┘  └─────────┘  └─────────┘
```

`netContext.RequestInfo` 接口抽象了所有协议的公共请求属性：

```go
type RequestInfo interface {
    GetMethod() string           // HTTP: GET/POST；WS: action；gRPC: 服务方法名
    GetPath() string             // HTTP: /api/login；WS: action；gRPC: /pkg.Service/Method
    GetClientIP() string         // 客户端IP（通用所有协议）
    GetHeader(key string) string // HTTP: Header；WS: 握手Header；gRPC: Metadata
    GetQuery(key string) string  // HTTP: URL.Query；WS: 握手Query；gRPC: Metadata
}
```

### 核心模块组成

1. **Bootstrap（启动模块）**：统一服务启动入口，支持多协议服务同时启动、优雅停机
2. **Config（配置模块）**：支持 JSON 格式配置文件，支持配置热更新钩子
3. **Base（基础模块）**：提供 BaseController、BaseService、BaseModel、BaseComponent 等基类
4. **HTTP（HTTP 服务模块）**：HTTP 服务器、路由、中间件（CORS、Recovery）
5. **WebSocket（WebSocket 服务模块）**：WebSocket 服务器、连接管理、事件监听
6. **gRPC（gRPC 服务模块）**：gRPC 服务器、路由、拦截器
7. **Database（数据库模块）**：MySQL、PostgreSQL、ClickHouse、MongoDB、Redis、ElasticSearch 支持，链式调用
8. **Logger（日志模块）**：统一日志管理，支持多级别、按应用分目录
9. **Function（功能函数模块）**：常用工具函数（Md5、Json_encode、Random、StrToTime 等）

---

## 快速上手

### 创建第一个 HTTP 服务

以用户查询接口为例，完整演示 MVC 三层开发流程。

#### 1. 创建模型层

文件路径：`app/api/model/user/user.go`

```go
package user

import (
    "context"
    "encoding/json"
    "errors"
    "time"
    "github.com/dfpopp/go-dai/base"
)

// User 用户实体
type User struct {
    Id       int    `json:"id,omitempty"`
    Username string `json:"username,omitempty"`
    Email    string `json:"email,omitempty"`
    Age      int    `json:"age,omitempty"`
}

type Model struct {
    *base.BaseModel
    DbTag string
    Table string
    Pk    string
}

func NewModel() *Model {
    return &Model{
        BaseModel: &base.BaseModel{},
        DbTag:     "default",
        Table:     "users",
        Pk:        "id",
    }
}

// GetOne 根据条件查询单条用户
func (m *Model) GetOne(field, whereTpl string, whereArgs []interface{}) (*User, error) {
    m.Init()
    db, err := m.GetMysqlDb(m.DbTag)
    if err != nil {
        return nil, err
    }
    db.SetTable(m.Table)
    if field != "" {
        db.SetField(field)
    }
    if whereTpl != "" {
        db.SetWhere(whereTpl, whereArgs...)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    retStr, err := db.Find(ctx)
    if err != nil {
        m.LogError(err)
        return nil, errors.New("数据查询失败")
    }
    if retStr == "" {
        return nil, nil
    }
    userInfo := &User{}
    err = json.Unmarshal([]byte(retStr), userInfo)
    return userInfo, err
}
```

#### 2. 创建服务层

文件路径：`app/api/service/user/user.go`

```go
package user

import (
    "errors"
    "github.com/dfpopp/go-dai/base"
    userModel "myproject/app/api/model/user"
)

type Service struct {
    *base.BaseService
    userModel *userModel.Model
}

func NewService() *Service {
    return &Service{
        BaseService: &base.BaseService{},
        userModel:   userModel.NewModel(),
    }
}

// GetUserById 根据用户ID查询用户信息
func (s *Service) GetUserById(userId int) (*userModel.User, error) {
    s.Init()
    if userId <= 0 {
        return nil, errors.New("用户ID非法")
    }
    field := "id,username,email,age"
    userInfo, err := s.userModel.GetOne(field, "id=?", []interface{}{userId})
    if err != nil {
        s.LogError(err)
        return nil, errors.New("用户查询失败")
    }
    if userInfo == nil {
        return nil, errors.New("用户不存在")
    }
    return userInfo, nil
}
```

#### 3. 创建控制器

文件路径：`app/api/controller/user/user.go`

```go
package user

import (
    "strconv"
    "github.com/dfpopp/go-dai/base"
    "github.com/dfpopp/go-dai/netContext"
    userService "myproject/app/api/service/user"
)

type Controller struct {
    *base.BaseController
    userService *userService.Service
}

func NewController() *Controller {
    return &Controller{
        BaseController: &base.BaseController{},
        userService:    userService.NewService(),
    }
}

// GetUser 获取用户信息
func (c *Controller) GetUser(ctx netContext.Context) {
    c.Init(ctx)
    // 1. 参数获取
    idStr := c.GetPostForm("id")
    if idStr == "" {
        c.Error(400, "用户ID不能为空")
        return
    }
    id, err := strconv.Atoi(idStr)
    if err != nil {
        c.Error(400, "用户ID格式错误")
        return
    }
    // 2. 调用服务层
    userInfo, err := c.userService.GetUserById(id)
    if err != nil {
        c.Error(400, err.Error())
        return
    }
    // 3. 响应封装
    c.Success(userInfo)
}
```

#### 4. 创建路由

文件路径：`app/api/router/router.go`

```go
package router

import (
    "github.com/dfpopp/go-dai/base"
    "github.com/dfpopp/go-dai/http"
    "myproject/app/api/controller/user"
    "myproject/app/api/middleware"
)

type ApiRouter struct {
    *base.DefaultBaseRouter
    userController *user.Controller
}

func NewApiRouter() *ApiRouter {
    return &ApiRouter{
        DefaultBaseRouter: &base.DefaultBaseRouter{},
        userController:    user.NewController(),
    }
}

func (r *ApiRouter) RegisterHTTPRoutes(server *http.Server) {
    server.POST("/user/info", http.ToHTTPHandler(r.userController.GetUser), middleware.Auth())
}
```

#### 5. 创建中间件（可选）

文件路径：`app/api/middleware/http_middleware.go`

```go
package middleware

import (
    "sort"
    "strings"
    "github.com/dfpopp/go-dai/config"
    "github.com/dfpopp/go-dai/function"
    "github.com/dfpopp/go-dai/http"
)

// Auth 签名认证中间件
func Auth() http.MiddlewareFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(c *http.Context) {
            reqInfo := c.GetRequestInfo()
            sign := reqInfo.GetHeader("sign")
            t := reqInfo.GetHeader("t")
            appId := reqInfo.GetHeader("appId")
            if appId == "" {
                c.JSON(200, map[string]interface{}{"code": 401, "msg": "请使用正确的appId"})
                return
            }
            // 从自定义配置获取 apiKey
            customConfig := config.GetCustomConfig().GetConfig()
            apiKey := customConfig.Get("outSite").Get(appId).String()
            if apiKey == "" {
                c.JSON(200, map[string]interface{}{"code": 401, "msg": "请使用正确的appId"})
                return
            }
            // 按参数名排序后拼接签名
            params := c.PostFormAll()
            keys := make([]string, 0)
            for key, value := range params {
                if value != "" {
                    keys = append(keys, key)
                }
            }
            sort.Strings(keys)
            signStr := apiKey
            for _, key := range keys {
                signStr += key + params[key]
            }
            signStr += apiKey + "t=" + t
            if sign == strings.ToUpper(function.Md5(signStr)) {
                next(c)
            } else {
                c.JSON(200, map[string]interface{}{"code": 401, "msg": "签名验证失败"})
            }
        }
    }
}
```

#### 6. 创建入口文件

文件路径：`api.go`

```go
package main

import (
    "github.com/dfpopp/go-dai/bootstrap"
    "github.com/dfpopp/go-dai/config"
    "myproject/app/api/router"
)

func main() {
    appName := "api"
    // 1. 加载配置
    if err := config.LoadAppConfig("config/app.json", appName); err != nil {
        panic("加载应用配置失败：" + err.Error())
    }
    if err := config.LoadDatabaseConfig("config/database.json"); err != nil {
        panic("加载数据库配置失败：" + err.Error())
    }
    // 2. 构建启动配置
    bootCfg := &bootstrap.BootConfig{
        AppName:            appName,
        AppConfigPath:      "config/app.json",
        DatabaseConfigPath: "config/database.json",
        CustomConfigPath:   "config/custom.json",
        EnableServices:     []bootstrap.ServiceType{bootstrap.ServiceTypeHTTP},
        Router:             router.NewApiRouter(),
    }
    // 3. 一键启动服务
    _, err := bootstrap.Boot(bootCfg)
    if err != nil {
        panic("应用启动失败: " + err.Error())
    }
}
```

#### 7. 配置文件示例

`config/app.json`：

```json
{
  "api": {
    "name": "api-service",
    "env": "prod",
    "http": {
      "addr": ":8080",
      "read_timeout": 30,
      "write_timeout": 30,
      "max_header_bytes": 1048576,
      "ssl": false,
      "ssl_cert_file": "",
      "ssl_key_file": ""
    },
    "logger": {
      "path": "runtime/log/"
    }
  }
}
```

`config/database.json`：

```json
{
  "mysql": {
    "default": {
      "host": "127.0.0.1",
      "port": "3306",
      "user": "root",
      "pwd": "123456",
      "dbname": "myproject",
      "pre": "wk_",
      "charset": "utf8",
      "max_open_conn_num": 100,
      "max_idle_conn_num": 20,
      "conn_max_lifetime": 3600
    }
  },
  "redis": {
    "default": {
      "host": "127.0.0.1",
      "port": "6379",
      "pwd": "",
      "pre": "myproject_",
      "pool_size": 64,
      "min_idle_conns": 10,
      "idle_timeout": 300,
      "timeout": 5
    }
  }
}
```

---

## 核心模块详解

### Bootstrap 启动模块

统一服务启动入口，支持多协议服务同时启动、优雅停机、业务层清理回调。

```go
type BootConfig struct {
    AppName            string          // 应用名（如 api/admin/cron）
    AppConfigPath      string          // 应用配置文件路径
    DatabaseConfigPath string          // 数据库配置文件路径
    CustomConfigPath   string          // 自定义配置文件路径（可选）
    EnableServices     []ServiceType   // 需要启动的服务类型
    Router             base.BaseRouter // 应用路由实例
    ShutdownFunc       func()          // 服务关闭回调函数（可选）
}

// 服务类型枚举
const (
    ServiceTypeHTTP ServiceType = "http"
    ServiceTypeWS   ServiceType = "ws"
    ServiceTypeGRPC ServiceType = "grpc"
)
```

**启动流程**：

1. 校验启动配置完整性
2. 加载应用配置、数据库配置、自定义配置
3. 初始化日志（按应用名分目录）
4. 初始化数据库连接（按配置自动识别 MySQL/PostgreSQL/ClickHouse/MongoDB/Redis/ES）
5. 按服务类型初始化并启动服务（HTTP/WS/gRPC）
6. 注册信号监听，支持优雅停机

**多协议混合启动**：

```go
bootCfg := &bootstrap.BootConfig{
    AppName:        appName,
    EnableServices: []bootstrap.ServiceType{
        bootstrap.ServiceTypeHTTP,
        bootstrap.ServiceTypeWS,
        bootstrap.ServiceTypeGRPC,
    },
    Router: router.NewApiRouter(),
}
```

### BaseController 控制器基类

提供统一的请求处理、响应封装、WebSocket 消息推送能力。

**核心方法**：

| 方法 | 说明 |
|-----|------|
| `Init(ctx netContext.Context)` | 初始化控制器（必须方法首行调用） |
| `Success(data interface{}, msg ...string)` | 统一成功响应 |
| `DataSuccess(data interface{}, count int64)` | 统一成功响应（带分页总数） |
| `Error(code int, msg string)` | 统一失败响应 |
| `RespText(msg string)` | 文本响应 |
| `GetCtx() (netContext.Context, error)` | 获取当前上下文 |
| `GetConnID()` | 获取当前连接ID（WebSocket） |
| `BindUserID(userID string)` | 绑定用户ID到连接（WebSocket） |
| `UnbindUserID(userID string)` | 解绑用户ID（WebSocket） |
| `GetUserConnIDs(userID string)` | 获取用户所有在线连接 |
| `SendToConnID(connID, action, data)` | 给指定连接发送消息 |
| `SendToConnIDs(connIDs, action, data)` | 批量给多个连接发送消息 |
| `SendToUser(userID, action, data)` | 给指定用户所有连接发送消息 |
| `SendToUsers(userIDs, action, data)` | 批量给多个用户发送消息 |
| `LogInfo/LogError/LogWarn` | 日志记录 |

**统一响应格式**：

```go
// 成功响应
{
    "code": 200,
    "msg": "操作成功",
    "data": {...}
}

// 失败响应
{
    "code": 400,
    "msg": "参数错误",
    "data": null
}

// 分页响应
{
    "code": 200,
    "msg": "操作成功",
    "data": [...],
    "count": 100
}
```

### BaseService 服务基类

提供基础组件能力（日志、初始化）。

```go
type BaseService struct {
    BaseComponent
}

func (s *BaseService) Init() {
    s.BaseComponent.Init()
}
```

### BaseModel 模型基类

提供统一的数据库操作能力。

**核心方法**：

| 方法 | 说明 |
|-----|------|
| `GetMysqlDb(tag string)` | 获取 MySQL 数据库实例 |
| `GetPostgresDb(tag string)` | 获取 PostgreSQL 数据库实例 |
| `GetClickhouseDb(tag string)` | 获取 ClickHouse 数据库实例 |
| `GetMongoDb(tag string)` | 获取 MongoDB 数据库实例 |
| `GetRedis(tag string)` | 获取 Redis 数据库实例 |
| `GetESDb(tag string)` | 获取 ElasticSearch 数据库实例 |
| `StrIdToObjectID(id string)` | 字符串转 MongoDB ObjectID |
| `ObjectIDToString(id)` | MongoDB ObjectID 转字符串 |
| `MapToBsonD(data)` | map 转 bson.D |
| `StringToFulltextIndexStr(input)` | 生成全文索引字符串 |
| `StringToSearchFulltextStr(input)` | 生成搜索用全文索引字符串 |
| `LogInfo/LogError/LogWarn` | 日志记录 |

### 配置管理

框架支持三类配置文件：

1. **应用配置（app.json）**：应用名、环境、HTTP/WS/gRPC 服务配置、日志配置
2. **数据库配置（database.json）**：MySQL、PostgreSQL、ClickHouse、MongoDB、Redis、ElasticSearch 连接配置
3. **自定义配置（custom.json）**：业务自定义配置（如签名密钥、第三方接口配置）

**配置热更新**：框架内置配置变更钩子，数据库配置变更时会自动重新初始化连接池（平滑切换，延迟 30 秒关闭旧连接）。

```go
// 注册配置变更钩子
config.RegisterConfigChangeHook(func(configType string, cfg interface{}) error {
    if configType == "database" {
        // 处理数据库配置变更
    }
    return nil
})
```

**读取自定义配置**：

```go
customConfig := config.GetCustomConfig().GetConfig()
rc4Pwd := customConfig.Get("rc4_pwd").String()
mgDbName := customConfig.Get("inSite").Get(site).Get("mgDb").String()
```

### 中间件机制

框架内置中间件：

- `http.Recovery()` - 异常恢复，捕获 panic 返回 500
- `http.CORS()` - 跨域处理，允许所有域

**自定义中间件签名**：

```go
type MiddlewareFunc func(next HandlerFunc) HandlerFunc
```

**中间件使用示例**：

```go
// 单中间件
server.POST("/user/info", http.ToHTTPHandler(handler), middleware.Auth())

// 多中间件（按顺序执行）
server.POST("/biddingInfo/list", http.ToHTTPHandler(handler),
    middleware.Auth(),
    middleware.RateLimit(),
    middleware.Logger(),
)
```

### WebSocket 连接管理

框架提供全局连接管理器，支持连接创建、移除、属性管理、消息推送、事件监听。

**事件类型**：

- `EventConnOnline` - 用户上线事件
- `EventConnOffline` - 用户下线事件

**实现连接事件处理器**：

```go
package ws

import (
    "github.com/dfpopp/go-dai/websocket"
    "github.com/dfpopp/go-dai/logger"
)

type CustomConnListener struct{}

func NewCustomConnListener() *CustomConnListener {
    return &CustomConnListener{}
}

func (l *CustomConnListener) OnConnect(connInfo *websocket.ConnInfo) {
    logger.Info("用户上线", "connID", connInfo.ConnID, "clientIP", connInfo.ClientIP)
}

func (l *CustomConnListener) OnDisconnect(connInfo *websocket.ConnInfo, reason string) {
    logger.Info("用户下线", "connID", connInfo.ConnID, "reason", reason)
}
```

**在入口文件中注册**：

```go
func init() {
    connManager := websocket.GetGlobalConnManager()
    eventMgr := connManager.GetEventManager()
    customListener := ws.NewCustomConnListener()
    eventMgr.RegisterHandler(customListener)
}
```

---

## 多协议开发指南

### HTTP 开发

HTTP 控制器方法签名：

```go
func (c *Controller) Method(ctx netContext.Context) {
    c.Init(ctx)
    // 业务逻辑
}
```

**参数获取**：

```go
// 获取 POST 表单参数
id := c.GetPostForm("id")
// 获取所有 POST 参数
params := c.GetPostFormAll()
// 获取 Query 参数
id := ctx.Query("id")
// 获取请求头
sign := ctx.GetRequestInfo().GetHeader("sign")
// 获取客户端IP
ip := ctx.GetRequestInfo().GetClientIP()
// 获取请求体
body, err := c.GetBody()
// JSON 绑定
var req MyRequest
err := ctx.BindJSON(&req)
```

**路由注册**：

```go
server.POST("/user/info", http.ToHTTPHandler(r.userController.GetUser), middleware.Auth())
server.GET("/user/list", http.ToHTTPHandler(r.userController.GetList))
server.PUT("/user/update", http.ToHTTPHandler(r.userController.Update))
server.DELETE("/user/delete", http.ToHTTPHandler(r.userController.Delete))
```

### WebSocket 开发

WebSocket 控制器使用 `WsInit` 初始化：

```go
func (c *Controller) Login(ctx netContext.Context) {
    c.WsInit(ctx)
    userId := c.GetPostForm("user_id")
    // 绑定用户ID到当前连接
    err := c.BindUserID(userId)
    if err != nil {
        c.Error(400, err.Error())
        return
    }
    c.Success(nil, "登录成功")
}
```

**消息推送**：

```go
// 给指定用户推送消息
c.SendToUser(userId, "message.notify", map[string]interface{}{
    "content": "您有新的消息",
})

// 给指定连接推送消息
c.SendToConnID(connID, "message.notify", data)

// 批量给多个用户推送
c.SendToUsers([]string{userId1, userId2}, "broadcast", data)
```

**WebSocket 路由注册**：

```go
func (r *ApiRouter) RegisterWSRoutes(server *websocket.Server) {
    server.Register("login", websocket.ToWSHandler(r.userController.Login))
    server.Register("message.send", websocket.ToWSHandler(r.messageController.Send))
}
```

### gRPC 开发

#### 1. 定义 Protobuf 文件

```protobuf
syntax = "proto3";
package user;
option go_package = "./gen/user;user";

message UserRequest {
    int32 id = 1;
}

message UserResponse {
    int32 code = 1;
    string msg = 2;
    UserData data = 3;
}

message UserData {
    int32 id = 1;
    string username = 2;
    string email = 3;
}

service UserService {
    rpc GetUser(UserRequest) returns (UserResponse);
}
```

#### 2. 生成 gRPC 代码

```bash
protoc --go_out=./gen/user --go_opt=paths=source_relative \
       --go-grpc_out=./gen/user --go-grpc_opt=paths=source_relative \
       proto/user.proto
```

#### 3. 编写 gRPC 控制器

```go
package user

import (
    "context"
    "github.com/dfpopp/go-dai/base"
    userGen "myproject/gen/user"
    userService "myproject/app/api/service/user"
)

type Controller struct {
    *base.BaseController
    userGen.UnimplementedUserServiceServer
    userService *userService.Service
}

func NewController() *Controller {
    return &Controller{
        BaseController:                 &base.BaseController{},
        UnimplementedUserServiceServer: userGen.UnimplementedUserServiceServer{},
        userService:                    userService.NewService(),
    }
}

func (c *Controller) GetUser(ctx context.Context, req *userGen.UserRequest) (*userGen.UserResponse, error) {
    userId := int(req.Id)
    if userId <= 0 {
        return &userGen.UserResponse{Code: 400, Msg: "用户ID非法"}, nil
    }
    userInfo, err := c.userService.GetUserById(userId)
    if err != nil {
        return &userGen.UserResponse{Code: 500, Msg: err.Error()}, nil
    }
    return &userGen.UserResponse{
        Code: 0,
        Msg:  "操作成功",
        Data: &userGen.UserData{
            Id:       int32(userInfo.Id),
            Username: userInfo.Username,
            Email:    userInfo.Email,
        },
    }, nil
}
```

#### 4. 注册 gRPC 路由

```go
func (r *ApiRouter) RegisterGRPCRoutes(grpcServer *grpc.Server) {
    userGen.RegisterUserServiceServer(grpcServer.GrpcServer, r.grpcUserController)
}
```

### 三种协议对比

| 维度 | HTTP | WebSocket | gRPC |
|-----|------|-----------|------|
| 控制器继承 | `base.BaseController` | `base.BaseController` | `base.BaseController` + `UnimplementedXXXServer` |
| 方法签名 | `func(ctx netContext.Context)` | `func(ctx netContext.Context)` | `func(ctx context.Context, req *Request) (*Response, error)` |
| 初始化方法 | `c.Init(ctx)` | `c.WsInit(ctx)` | 标准方法 |
| 参数获取 | `c.GetPostForm("id")` | `ctx.BindJSON(&req)` | 直接访问 `req.Id` |
| 响应返回 | `c.Success(data)` | `c.Success(data)` | 返回 `&Response{Code:0, Data:data}` |
| 路由标识 | URL路径 + HTTP方法 | action | 服务名 + 方法名 |
| 路由注册 | `server.POST("/path", handler)` | `wsServer.Register("action", handler)` | `RegisterXXXServiceServer(server, controller)` |
| 中间件 | `middleware.Auth()` | WS 中间件 | gRPC 拦截器 |

---

## 数据库操作指南

### MySQL 链式操作

```go
db, err := m.GetMysqlDb(m.DbTag)
if err != nil {
    return err
}
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// 单条查询
retStr, err := db.SetTable("users").
    SetField("id,username,email").
    SetWhere("id=?", userId).
    Find(ctx)

// 列表查询
retStr, err := db.SetTable("users").
    SetField("id,username,email").
    SetWhere("status=?", "Y").
    SetOrder("create_time DESC").
    SetLimit(0, 10).
    FindAll(ctx).ToString()

// 总数查询
count, err := db.SetTable("users").
    SetWhere("status=?", "Y").
    FindCount(ctx)

// 插入
id, err := db.SetTable("users").Insert(ctx, map[string]interface{}{
    "username": "test",
    "email":    "test@example.com",
})

// Upsert（幂等写入：唯一键冲突时把指定列更新为本次插入的对应值）
_, err = db.SetTable("ios_uid_map").SetUpsert("uid", "product_id", "account").Insert(ctx, map[string]interface{}{
    "uid":                    "10001",
    "original_transaction_id": "2000012345678901",
    "product_id":             "com.xxx.member.12m",
    "account":                "userA",
})
// 生成：INSERT INTO ios_uid_map (...) VALUES (...)
//       ON DUPLICATE KEY UPDATE `uid` = ?, `product_id` = ?, `account` = ?

// 更新
affected, err := db.SetTable("users").
    SetWhere("id=?", userId).
    Update(ctx, map[string]interface{}{"email": "new@example.com"})

// 删除
affected, err := db.SetTable("users").
    SetWhere("id=?", userId).
    Delete(ctx)

// 事务
err = db.ToBegin()
// ... 业务操作
err = db.Commit()
// 或 err = db.Rollback()
```

**MySQL 链式方法**：

| 方法 | 说明 |
|-----|------|
| `SetTable(table)` | 设置表名 |
| `SetField(field)` | 设置查询字段 |
| `SetWhere(tpl, args...)` | 设置 WHERE 条件 |
| `SetWhereOr(data)` | 设置 OR 条件 |
| `SetWhereIn(field, args...)` | 设置 IN 条件 |
| `SetOrder(order)` | 设置排序 |
| `SetGroup(group)` | 设置分组 |
| `SetJoin(table, condition, joinType)` | 设置 JOIN |
| `SetLimit(skip, num)` | 设置分页 |
| `Find(ctx)` | 查询单条 |
| `FindAll(ctx)` | 查询多条（需配合 ToString） |
| `FindCount(ctx)` | 查询总数 |
| `Insert(ctx, data)` | 插入 |
| `InsertAll(ctx, dataList)` | 批量插入 |
| `SetUpsert(cols...)` | 设置唯一键冲突更新的列（配合 Insert/InsertAll 生成 ON DUPLICATE KEY UPDATE） |
| `Update(ctx, data)` | 更新 |
| `Delete(ctx)` | 删除 |
| `ToBegin/Commit/Rollback` | 事务控制 |

### MongoDB 链式操作

```go
mgDb, err := m.GetMongoDb(m.DbTag)
if err != nil {
    return err
}
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// 单条查询
retStr, err := mgDb.SetTable("member").
    SetWhere(bson.D{{"uid", uid}, {"site", site}}).
    Find(ctx)

// 列表查询
retStr, err := mgDb.SetTable("member").
    SetWhere(bson.D{{"partner", "wendou"}}).
    SetSort(bson.D{{"createTime", -1}}).
    SetSkip((page - 1) * pageSize).
    SetLimit(pageSize).
    FindAll(ctx).ToString()

// 跨库查询（动态切换数据库与表前缀）
retStr, err := mgDb.SetDbName("gczb").
    SetTablePre("").
    SetTable("member").
    SetProjection(bson.D{{"_id", 1}, {"account", 1}}).
    SetWhere(bson.D{{"_id", objectId}}).
    Find(ctx)

// 插入
objectId, err := mgDb.SetTable("member").Insert(ctx, data)

// 更新
affected, err := mgDb.SetTable("member").
    SetWhere(bson.D{{"_id", objectId}}).
    Update(ctx, bson.D{{"$set", bson.D{{"mobile", phone}}}})

// 删除
affected, err := mgDb.SetTable("member").
    SetWhere(bson.D{{"_id", objectId}}).
    Delete(ctx)
```

**MongoDB 链式方法**：

| 方法 | 说明 |
|-----|------|
| `SetDbName(dbName)` | 设置数据库名（支持跨库） |
| `SetTablePre(pre)` | 设置表前缀 |
| `SetTable(col)` | 设置集合名 |
| `SetWhere(filter)` | 设置查询条件（bson.D） |
| `SetSort(sort)` | 设置排序 |
| `SetLimit(limit)` | 设置限制条数 |
| `SetSkip(skip)` | 设置跳过条数 |
| `SetProjection(proj)` | 设置投影字段 |
| `SetAgg(pipeline)` | 设置聚合管道 |
| `Find(ctx)` | 查询单条 |
| `FindAll(ctx)` | 查询多条 |
| `FindCount(ctx)` | 查询总数 |
| `Aggregate(ctx)` | 聚合查询 |
| `Insert(ctx, doc)` | 插入 |
| `InsertAll(ctx, docs)` | 批量插入 |
| `Update(ctx, update)` | 更新 |
| `Delete(ctx)` | 删除 |

### Redis 操作

```go
redisDb, err := m.GetRedis("default")
if err != nil {
    return err
}

// 设置值（带过期时间）
err = redisDb.Db.Set(ctx, "key", "value", 10*time.Minute).Err()

// 获取值
val, err := redisDb.Db.Get(ctx, "key").Result()

// 分布式锁
ok, err := redisDb.Db.SetNX(ctx, "lock_key", 1, 2*time.Second).Result()

// 自增
err = redisDb.Db.Incr(ctx, "counter").Err()

// 删除
err = redisDb.Db.Del(ctx, "key").Err()
```

**注意**：Redis 操作使用 `redisDb.DbPre` 作为 key 前缀，避免多应用 key 冲突。

### ElasticSearch 操作

```go
esDb, err := m.GetESDb(m.DbTag)
if err != nil {
    return err
}
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// 列表查询
retStr, err := esDb.SetHighlight("title", elasticSearch.HighlightOption{
    PreTag: "<font style='color:red'>",
    PostTag: "</font>",
}).
    SetExcludeSource("content", "absentFields").
    SetScriptFieldTruncate("content", 100).
    SetLimit((page-1)*pageSize, pageSize).
    FindAll(ctx).ToString()

// 总数查询
count, err := esDb.FindCount(ctx)
```

**常用链式能力**：`SetAggs`（聚合）、`SetCollapse`（字段去重折叠）、`SetTrackTotalHits`（精确总命中数，ES 7+ 默认 1 万上限）、`SetSearchTimeout`（服务端查询超时降级）、`SetHighlight`（高亮）、`SetScriptField*`（脚本字段）等。

### PostgreSQL 链式操作

链式 API 与 MySQL 同名同参，差异由框架内部消化：

```go
db, err := m.GetPostgresDb(m.DbTag)

// 查询：代码照常写 ?，执行前自动转 $1..$n；SetLimit(skip,num) 自动转 LIMIT num OFFSET skip
retStr, err := db.SetTable("orders").
    SetField("id,order_no,amount").
    SetWhere("user_id=?", userId).
    SetLimit(0, 10).
    FindAll(ctx).ToString()

// 插入：数据含 id 字段时自动 RETURNING "id" 返回自增值，否则返回 0
id, err := db.SetTable("orders").Insert(ctx, data)
```

> 注意：`Delete` 不支持 LIMIT；表名引用自动加双引号，支持 `SetTable("schema.table")`。

### ClickHouse 链式操作

链式 API 与 MySQL 同名同参；无事务；`Update`/`Delete` 自动转为 `ALTER TABLE` 异步 mutation：

```go
db, err := m.GetClickhouseDb(m.DbTag)

// 聚合统计
count, err := db.SetTable("events").
    SetField("id").
    SetWhere("action=?", action).
    FindCount(ctx)

// 行级展开（ARRAY JOIN）：数组列逐元素展开成行后分组/计数
count, err := db.SetTable("ztb").
    SetArrayJoin("zbhx AS x").
    SetField("DISTINCT x").
    SetWhere("arrayExists(x -> x = ?, zbhx)", companyName).
    FindCount(ctx)
```

> 注意：CH 无事务，批量写入用 `InsertAll` 单次提交；`Insert` 返回受影响行数，主键由业务生成；Update/Delete 异步执行，需后台完成后查询结果。

---

## 落地应用案例

框架已在多个生产项目中落地，以下为三个典型应用案例。

### 案例 1：bidResourceServer（标书资源服务）

**项目定位**：标书文档、视频课程、专家咨询的电商型资源服务平台。

**技术特点**：

- 多应用架构：`app/api`（C端接口）、`app/seller`（商家端）、`app/cron`（定时任务）
- 多入口文件：`api.go`、`seller.go`、`cron.go`
- 多数据库混合使用：MySQL（订单/用户）、MongoDB（文档/视频）、Redis（缓存/锁）
- 第三方集成：阿里 OSS、阿里短信、支付宝、微信支付、文兜 API、AI 大模型（豆包）

**目录结构亮点**：

```
bidResourceServer/
├── app/
│   ├── api/              # C端接口应用
│   ├── seller/           # 商家端应用
│   └── cron/             # 定时任务应用
├── pkg/                  # 第三方依赖包
│   ├── Ai/ark/           # AI 大模型集成
│   ├── aliOss/           # 阿里 OSS
│   ├── aliSms/           # 阿里短信
│   └── pay/              # 支付集成（支付宝/微信）
├── api.go                # C端入口
├── seller.go             # 商家端入口
└── cron.go               # 定时任务入口
```

**典型业务场景**：

- 标书文档列表查询、详情、下载、章节同步
- 视频课程播放、收藏、购买
- 订单创建、支付、分佣
- 定时任务：文档章节同步、预览图生成、订单状态同步

### 案例 2：openServer（开放平台服务）

**项目定位**：对外开放的标讯数据 API 平台，提供标讯列表、详情、按企业查询等接口。

**技术特点**：

- ElasticSearch 全文检索：标讯数据存储于 ES，支持高亮、分页、复杂查询
- 多租户鉴权：基于 appId/appSecret 的签名认证，支持接口级权限控制
- 多级限流：基于 Redis 的请求频率限制，按租户配置限流阈值
- 日志中间件：记录所有 API 请求日志，支持统计分析

**中间件组合示例**：

```go
server.POST("/biddingInfo/list",
    http.ToHTTPHandler(r.biddingInfoController.GetList),
    middleware.Auth(),      // 签名认证 + 权限校验
    middleware.RateLimit(), // 限流
    middleware.Logger(),    // 请求日志
)
```

**典型业务场景**：

- 标讯列表查询（支持多条件筛选、高亮）
- 标讯详情查询（含加密详情）
- 按企业名称查询标讯
- 区域数据查询（省/市）
- API 调用统计与日志分析

### 案例 3：ldPartnerServer（合作伙伴服务）

**项目定位**：多站点合作伙伴的分佣结算平台，对接文兜、善达等多个合作方。

**技术特点**：

- 多站点配置：通过 `custom.json` 的 `inSite`/`outSite` 配置管理多站点数据库与密钥
- 跨库查询：动态切换 MongoDB 数据库（`SetDbName`/`SetTablePre`）查询不同站点用户
- 加密通信：RC4 + Base64 编码生成 ucode，实现用户身份加密传递
- 多支付渠道：支付宝（App/H5/PC）、微信支付（JSAPI/小程序/App/H5/PC）
- 对账系统：日/周/月对账，自动同步订单与分佣数据

**多站点配置示例**：

```json
{
  "inSite": {
    "ld":  {"mgDb": "gczb", "pre": ""},
    "gc":  {"mgDb": "gczb_new", "pre": ""},
    "by":  {"mgDb": "biaoYan", "pre": ""},
    "newcc": {"mgDb": "newcc", "pre": ""}
  },
  "outSite": {
    "sd": "3cdb54f9031b75edd981bcd3ad37f9bf",
    "wd": "3cdb54f9031b75edd981bcd3ad19fbdf"
  }
}
```

**典型业务场景**：

- 用户同步：将各站点用户同步到合作伙伴库
- 订单推送：接收合作方订单推送，落库并同步状态
- 支付处理：支持多渠道支付，处理支付回调
- 对账结算：按日/周/月生成对账单，计算分佣金额

### 三个案例的共同模式

通过分析三个落地应用，可提炼出基于 go-Dai 框架的通用开发模式：

1. **统一入口**：使用 `bootstrap.Boot` 一键启动，通过 `EnableServices` 控制服务类型
2. **MVC 严格分层**：Controller 仅做参数校验与响应封装，业务逻辑在 Service，数据操作在 Model
3. **链式数据库操作**：MySQL/MongoDB/ES 均支持链式调用，代码简洁高效
4. **中间件组合**：Auth + RateLimit + Logger 的标准中间件组合
5. **配置驱动**：通过 `custom.json` 管理多站点、多租户的业务配置
6. **pkg 目录复用**：第三方依赖统一放在 `pkg/` 目录，便于跨应用复用

---

## 常见问题

### Q1：如何同时启动多个服务？

使用 `EnableServices` 参数指定多个服务类型：

```go
EnableServices: []bootstrap.ServiceType{
    bootstrap.ServiceTypeHTTP,
    bootstrap.ServiceTypeWS,
    bootstrap.ServiceTypeGRPC,
}
```

### Q2：如何实现 WebSocket 用户绑定？

在控制器中调用 `BindUserID` 方法：

```go
func (c *Controller) Login(ctx netContext.Context) {
    c.WsInit(ctx)
    userId := c.GetPostForm("user_id")
    err := c.BindUserID(userId)
    if err != nil {
        c.Error(400, err.Error())
        return
    }
    c.Success(nil, "登录成功")
}
```

### Q3：如何实现 WebSocket 消息推送？

使用 `SendToUser` 或 `SendToConnID` 方法：

```go
// 给指定用户推送消息
c.SendToUser(userId, "message.notify", map[string]interface{}{
    "content": "您有新的消息",
})

// 给指定连接推送消息
c.SendToConnID(connID, "message.notify", data)
```

### Q4：如何配置数据库连接池？

在 `config/database.json` 中配置：

```json
{
  "mysql": {
    "default": {
      "max_open_conn_num": 100,
      "max_idle_conn_num": 20,
      "conn_max_lifetime": 3600
    }
  }
}
```

框架会根据 CPU 核数自动设置默认值：

- `MaxOpenConnNum` 默认：CPU 核数 × 3
- `MaxIdleConnNum` 默认：CPU 核数 × 2
- `ConnMaxIdleTime` 默认：300 秒
- `ConnMaxLifetime` 默认：1800 秒

### Q5：如何实现优雅停机？

框架内置信号监听（SIGINT/SIGTERM），停机流程：

1. 接收到停机信号
2. 执行业务层清理逻辑（`ShutdownFunc`）
3. 关闭数据库连接
4. 停止 HTTP/WebSocket/gRPC 服务

```go
bootCfg := &bootstrap.BootConfig{
    // ...
    ShutdownFunc: func() {
        // 业务层清理逻辑
        logger.NewService().Stop()
    },
}
```

### Q6：如何实现跨库查询（MongoDB）？

使用 `SetDbName` 和 `SetTablePre` 动态切换数据库：

```go
mgDb.SetDbName("gczb_new").
    SetTablePre("").
    SetTable("member").
    SetWhere(bson.D{{"_id", objectId}}).
    Find(ctx)
```

### Q7：如何实现分布式锁？

使用 Redis 的 `SetNX`：

```go
redisDb, _ := m.GetRedis("default")
cacheKey := redisDb.DbPre + "pay_order_" + orderNo
ok, err := redisDb.Db.SetNX(ctx, cacheKey+"_lock", 1, 2*time.Second).Result()
if !ok {
    return errors.New("请不要频繁提交")
}
defer func() {
    redisDb.Db.Del(ctx, cacheKey+"_lock")
}()
```

### Q8：如何自定义响应格式？

应用层可继承 `BaseController` 后重写 `Success`/`Error` 方法：

```go
type Common struct {
    *base.BaseController
}

func (c *Common) Success(data interface{}, msg ...string) {
    message := "操作成功"
    if len(msg) > 0 && msg[0] != "" {
        message = msg[0]
    }
    ctx, err := c.GetCtx()
    if err != nil {
        c.LogError(err)
        return
    }
    ctx.JSON(200, map[string]interface{}{
        "status": 1,
        "msg":    message,
        "data":   data,
    })
}
```

---

## 文档资源

- **框架开发者手册**：[框架开发者手册.md](file:///d:/golang/project/go-Dai/框架开发者手册.md) - 完整的开发指南，包含框架介绍、核心模块详解、开发指南、最佳实践等
- **代码规范文档**：[代码规范文档.md](file:///d:/golang/project/go-Dai/代码规范文档.md) - 详细的代码规范，包含命名规范、分层规范、数据库操作规范等
- **代码规范文档 2.0**：[代码规范文档_v2.0.md](file:///d:/golang/project/go-Dai/代码规范文档_v2.0.md) - 基于落地应用实践升级的代码规范，补充更多实战示例

---

## License

MIT License
