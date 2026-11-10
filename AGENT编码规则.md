# go-Dai Agent 编码规则

> 本文件是 Agent（AI 编码助手）在基于 go-Dai 框架**开发新应用项目 / 修改项目 / 新增功能模块**时必须遵守的强制规则。
> **不是用来修改 go-Dai 框架源码的**——框架源码改动不适用本规则。
> 目标标准：Agent 按本文档写出的代码，除复杂业务逻辑外，**结构、目录、API 调用、分层应基本不出错、可直接编译运行**。
> 依据：[代码规范文档_v2.0.md](file:///d:/golang/project/go-Dai/代码规范文档_v2.0.md)（与框架源码核实）。
> 约定：「**必须**」=不可违反；「**禁止**」=做了即视为错误。
> 组织：按真实开发流程——**项目骨架 → 配置 → 入口 → 路由 → 控制器 → Service → Model**，每章独立、带可直接参照的实例。

---

## 0. 文件操作铁律（最高优先级，违反即拒收）

1. **修改/创建文件必须用 Edit / Write，禁止任何 Windows 系统 shell 命令操作文件内容。**
   - 禁止：`Set-Content`、`Add-Content`、`Out-File`、`echo >`、`echo >>`、`[System.IO.File]::WriteAllText`、`>`、`>>`、`copy`、`Move-Item` 改写、`sed`/`awk` 等。
   - 允许：Edit 精确替换、Write 新建/整体覆盖、Read 读取。
2. **读取用 Read**（禁 `cat`/`Get-Content`/`type`）；**查找用 Glob**（禁 `Get-ChildItem`/`dir`/`find`）；**检索用 Grep**（禁 `Select-String`/`findstr`/`rg`/`grep`）。
3. **改代码前必须先 Read 目标文件**，确认上下文与缩进后再 Edit；禁止盲改。
4. **禁止一次性大段覆盖已有业务文件**（除非用户要求整体重写），优先小步 Edit。
5. 新建文件路径必须符合第 2 节目录规范。
6. **禁止提交/修改 `runtime/log/`、`vendor/`、配置中的敏感信息。**

---

## 1. 边界速查表

### 1.1 必须做
- 改文件前先 Read；改文件只用 Edit / Write。
- 严格 MVC 单向：Controller → Service → Model → DB；错误反向 Model → Service → Controller。
- Controller / Service / Model 三层方法**首行必须** `c.Init(ctx)` / `s.Init()` / `m.Init()`。
- 三层分别继承 `*base.BaseController` / `*base.BaseService` / `*base.BaseModel`。
- HTTP 路由处理器必须用 `http.ToHTTPHandler(c.Method)` 包装；WS 路由必须用 `websocket.ToWSHandler(c.Method)` 包装。
- WS 控制器取 JSON 参数用 `c.BindJSON(&req)`（data 是 JSON）。
- 所有 DB 操作必须 `context.WithTimeout` + `defer cancel()`。
- WHERE 必须占位符 `?`；更新/删除必须带 WHERE。
- 多表写入必须事务：MySQL `ToBegin()`，MongoDB `Begin(ctx)`。
- 认证接口绑 `middleware.Auth()`；回调接口不绑。
- 支付回调：记录原始参数 → 验签 → 幂等 → 返回第三方约定格式。
- 类型断言必须 `ok` 双返回值。
- goroutine 内必须 `recover()`；并发必须 channel 控制。
- **WS 服务的入口文件必须自带 `func init()` 注册连接事件监听器**（见 4.2）。
- gRPC 控制器必须同时继承 `*base.BaseController` 与 `UnimplementedXXXServer`；gRPC 路由用生成的 `RegisterXXXServiceServer`。
- 入口 `BootConfig` 必须填齐 `AppName`/`AppConfigPath`/`DatabaseConfigPath`/`EnableServices`/`Router`。
- **入口文件必须放项目根目录**：单应用 `main.go`，多应用 `api.go`/`admin.go`/`ws.go`/`cron.go`，编译用 `go build -o bin/xxx xxx.go`（见 2.3）。
- **controller 同时 import service/<模块> 与 model/<模块>（同名包）时必须用 import 别名**（见 6.1）。

### 1.2 坚决不能做
- **禁止**用 Windows shell 命令改文件。
- **禁止** Controller 直接操作数据库（`GetMysqlDb` 等）。
- **禁止** Service 调 Controller、Model 调 Service（反向调用）。
- **禁止** SQL 字符串拼接；**禁止**全表查询/`SetField("*")`/无 WHERE 的 Update/Delete。
- **禁止**事务内调外部 API；**禁止**业务代码 `panic`（仅启动阶段可用）。
- **禁止**把底层错误直接返回客户端；**禁止**硬编码配置。
- **禁止** Service 层直接调第三方 SDK（必须封装到 `pkg/`）。
- **禁止**回调接口绑认证中间件；**禁止**日志记录敏感信息。
- **禁止** Redis key 不设过期时间/不带业务前缀。
- **禁止**无限制 goroutine、`time.Sleep(0)` 死循环。
- **禁止**直接操作 WS 连接 map（必须走连接管理器）。
- **禁止**在 WS `OnConnect`/`OnDisconnect` 里做 DB 查询/调外部 API。
- **禁止**在 `main()` 里注册 WS 事件监听（必须入口文件 `init()`）。
- **禁止** WS 路由直接传控制器方法（必须 `ToWSHandler` 包装）。
- **禁止** gRPC 控制器用 `netContext.Context` 或返回 Go `error` 表达业务错误。
- **禁止**手动 `RegisterService` 注册 gRPC。
- **禁止**多应用时用 `go build`（不带文件名）或 `go build ./...` 编译根目录——多个 `func main()` 会报错；单应用 `main.go` 可直接 `go build`。
- **禁止**入口文件中放业务函数/类型定义——入口只含 `package main`、import、`func init()`、`func main()`。
- **禁止**手动拼格式，必须 `gofmt`/`goimports`。

---

## 2. 项目骨架与多应用目录结构

### 2.1 单应用项目骨架

**单应用项目 `app/` 下直接放 MVC 各层，不需要额外子目录**：

```
myproject/
├── main.go                      # 入口（package main，根目录，单应用通用）
├── app/
│   ├── controller/<模块>/<模块>.go
│   ├── service/<模块>/<模块>.go
│   ├── model/<模块>/<模块>.go
│   ├── router/router.go
│   ├── middleware/http_middleware.go
│   └── init/ws/listener.go     # 仅 WS 服务
├── config/
│   ├── app.json
│   ├── database.json
│   └── custom.json
├── pkg/                         # 第三方封装
├── proto/                       # 仅 gRPC
├── gen/                         # 仅 gRPC 生成代码
├── runtime/log/                 # 日志
├── go.mod
└── go.sum
```

### 2.2 多应用项目骨架（api / ws / cron 同一项目下）

一个项目可同时含 HTTP 应用、WS 应用、定时任务应用，**各自独立入口、独立编译，共享 config/pkg/runtime**：

```
myproject/
├── api.go                       # HTTP 入口（package main，根目录）
├── admin.go                     # 管理后台入口（package main，根目录）
├── ws.go                        # WS 入口（package main，含 init 监听器注册）
├── cron.go                      # 定时任务入口（package main）
├── app/
│   ├── api/                     # C 端 HTTP 应用
│   │   ├── controller/
│   │   ├── service/
│   │   ├── model/
│   │   ├── router/router.go
│   │   └── middleware/
│   ├── admin/                   # 管理后台 HTTP 应用
│   │   ├── controller/
│   │   ├── service/
│   │   ├── model/
│   │   ├── router/router.go
│   │   └── middleware/
│   ├── ws/                      # 独立 WebSocket 应用
│   │   ├── controller/
│   │   ├── service/
│   │   ├── model/
│   │   ├── router/router.go
│   │   ├── middleware/
│   │   └── init/ws/listener.go
│   └── cron/                    # 定时任务应用
│       ├── service/
│       └── model/
├── config/                      # 全项目共享
│   ├── app.json
│   ├── database.json
│   └── custom.json
├── pkg/                         # 全项目共享第三方封装
├── runtime/log/                 # 全项目共享
├── go.mod                       # 全项目唯一
└── go.sum
```

### 2.3 入口文件位置规则（关键）

**入口文件必须放项目根目录**（与 `go.mod` 同级）：

| 场景 | 入口文件名 | 说明 |
|------|-----------|------|
| 单应用 | `main.go` | 通用入口，可直接 `go build` |
| 多应用 - HTTP/gRPC | `api.go` | C 端接口服务 |
| 多应用 - 管理后台 | `admin.go` | 后台 HTTP 服务 |
| 多应用 - WebSocket | `ws.go` | 含 `func init()` 注册监听器 |
| 多应用 - 定时任务 | `cron.go` | 用 `BootCron` 启动 |

规则：

- 所有入口文件均为 `package main`，各自含独立的 `func main()`。
- **单应用**：根目录只有 `main.go`，可直接 `go build` 或 `go build -o bin/app`。
- **多应用**：多个入口文件共存于根目录是允许的——Go 单文件编译模式 `go build file.go` 只编译指定文件，不会因其它入口的 `func main()` 冲突。
- **多应用时禁止** `go build`（不带文件名）或 `go build ./...`——多个 `func main()` 会报错。
- **禁止**在入口文件中放置除 `package main` 声明、import、`func init()`、`func main()` 之外的其它内容。

### 2.4 module 路径与编译命令

- `go.mod` 的 `module` 名作为所有内部 import 前缀，示例统一用 `myproject`，实际替换为项目真实 module 名。
- **编译命令**：

```bash
# 单应用（main.go 可直接 go build）
go build -o bin/app
go build -o bin/app main.go

# 多应用（必须带入口文件名）
go build -o bin/api   api.go
go build -o bin/admin  admin.go
go build -o bin/ws     ws.go
go build -o bin/cron   cron.go
```

- 跨平台编译同理：`GOOS=linux GOARCH=amd64 go build -o bin/api api.go`。
- **多应用时禁止** `go build`（不带文件名）或 `go build ./...`——多个 `func main()` 会编译失败。

### 2.5 共享与隔离规则

- **共享**：`config/`、`pkg/`、`runtime/`、`go.mod` 全项目共用。
- **隔离**：各应用的 `controller/service/model/router/middleware` 互相独立，禁止跨应用直接 import 对方 service/model（如需共用逻辑，提取到 `pkg/` 或专门共享层）。
- **cron 应用**：只建 `service/` 和 `model/`（无 controller/router）。

---

## 3. 配置文件规范（完整样例）

### 3.1 app.json

**顶层是「应用名 → AppConfig」的 map**（框架 `LoadAppConfig` 解析为 `map[string]*AppConfig`）。多应用就在此加多个 key。`Boot` 时按 `AppName` 取对应配置。

```json
{
  "api": {
    "name": "api",
    "env": "dev",
    "http": {
      "addr": ":8080",
      "read_timeout": 30,
      "write_timeout": 30,
      "idle_timeout": 60,
      "max_header_bytes": 1048576,
      "ssl": false
    },
    "websocket": {
      "addr": ":8081",
      "read_timeout": 60,
      "write_timeout": 60,
      "path": "/ws",
      "origin": "*",
      "handshake_timeout": 3,
      "max_message_size": 1048576,
      "max_connections": 1000,
      "ssl": false
    },
    "grpc": {
      "addr": ":50051",
      "max_recv_msg_size": 4194304,
      "max_send_msg_size": 4194304,
      "keepalive_time": 300,
      "keepalive_timeout": 60,
      "max_concurrent_streams": 256,
      "timeout": 30,
      "ssl": false
    },
    "logger": { "path": "runtime/log" }
  },
  "cron": {
    "name": "cron",
    "env": "dev",
    "logger": { "path": "runtime/log" }
  }
}
```

> 应用只启用部分协议时，未用协议的配置段可省略（如 cron 无 http/websocket/grpc 段）。

### 3.2 database.json

```json
{
  "mysql": {
    "default": {
      "host": "127.0.0.1",
      "port": "3306",
      "user": "root",
      "pwd": "root",
      "dbname": "myproject",
      "charset": "utf8mb4",
      "pre": "tp_",
      "max_open_conn_num": 100,
      "max_idle_conn_num": 10,
      "conn_max_idleTime": 300,
      "conn_max_lifetime": 1800
    }
  },
  "mongodb": {
    "default": {
      "host": "127.0.0.1",
      "port": "27017",
      "user": "root",
      "pwd": "root",
      "dbname": "myproject",
      "pre": "",
      "charset": "",
      "max_pool_size": 100,
      "min_pool_size": 5,
      "max_conn_idle_time": 300,
      "timeout": 10
    }
  },
  "redis": {
    "default": {
      "host": "127.0.0.1",
      "port": "6379",
      "pwd": "",
      "pre": "",
      "db_index": 0,
      "pool_size": 100,
      "min_idle_conns": 5,
      "max_conn_lifetime": 300,
      "idle_timeout": 60,
      "read_timeout": 3,
      "write_timeout": 3,
      "timeout": 3,
      "max_retries": 3,
      "min_retry_backoff": 8,
      "max_retry_backoff": 512
    }
  },
  "es": {
    "default": {
      "host": "127.0.0.1",
      "port": "9200",
      "user": "",
      "pwd": "",
      "pre": "",
      "gzip_status": true,
      "max_idle_conn_num": 100,
      "max_idle_conn_num_per_host": 10,
      "idle_conn_timeout": 90,
      "max_conn_num_per_host": 100,
      "timeout": 10,
      "keep_alive": 30,
      "response_header_timeout": 10,
      "tls_handshake_timeout": 10
    }
  }
}
```

> 只需保留实际使用的引擎段。`pre` 为表名/索引前缀，框架 `SetTable` 会自动拼接。

### 3.3 custom.json 与读取

业务自定义配置，JSON 任意结构：

```json
{
  "outSite": {
    "app001": { "appId": "app001", "appSecret": "secretxxx", "rateLimit": 100 }
  },
  "sms": { "signName": "myproject", "timeout": 10 }
}
```

读取：

```go
import "github.com/dfpopp/go-dai/config"

customConfig := config.GetCustomConfig().GetConfig()
appSecret := customConfig.Get("outSite").Get("app001").Get("appSecret").String()
rateLimit := customConfig.Get("outSite").Get("app001").Get("rateLimit").Int()
```

---

## 4. 入口文件规范

入口为 `package main`，含 `func main()`。5 个必填字段缺一不可（`Boot` 会校验）：`AppName`、`AppConfigPath`、`DatabaseConfigPath`、`EnableServices`、`Router`。启动失败允许 `panic`（唯一允许业务 panic 的场景）。`Boot` 返回后主协程需 `select{}` 阻塞。

### 4.1 HTTP 入口（单应用）

```go
// main.go（项目根目录）
package main

import (
    "myproject/app/router"

    "github.com/dfpopp/go-dai/bootstrap"
)

func main() {
    bootCfg := &bootstrap.BootConfig{
        AppName:            "api",
        AppConfigPath:      "config/app.json",
        DatabaseConfigPath: "config/database.json",
        CustomConfigPath:   "config/custom.json",
        EnableServices:     []bootstrap.ServiceType{bootstrap.ServiceTypeHTTP},
        Router:             router.NewRouter(),
        ShutdownFunc:       func() { /* 资源清理 */ },
    }
    _, err := bootstrap.Boot(bootCfg)
    if err != nil {
        panic("应用启动失败: " + err.Error())
    }
    select {}
}
```

> 单应用 import 路径为 `myproject/app/router`（`app/` 下直接放 router）。多应用为 `myproject/app/api/router`（多一层应用目录）。

### 4.2 WebSocket 入口（重点：必须自带 init 注册监听器）

**WS 入口文件必须直接编写 `func init()`，在其中注册连接事件监听器。光空导入监听器包不会执行业务逻辑，必须显式调用构造函数并 `RegisterHandler`。`init()` 在 `main` 之前执行，此时 WS 服务尚未启动，注册安全有效。**

```go
// ws.go（项目根目录，多应用）
// 单应用 WS 直接在 main.go 中加 init() 即可
package main

import (
    "myproject/app/ws/init/ws"    // 多应用
    // "myproject/app/init/ws"    // 单应用
    "myproject/app/ws/router"     // 多应用
    // "myproject/app/router"     // 单应用

    "github.com/dfpopp/go-dai/bootstrap"
    "github.com/dfpopp/go-dai/websocket"
)

// init 注册 WS 连接事件监听器（必须在 main 之前执行）
func init() {
    connManager := websocket.GetGlobalConnManager()
    eventMgr := connManager.GetEventManager()
    customListener := ws.NewCustomConnListener()
    eventMgr.RegisterHandler(customListener)
}

func main() {
    bootCfg := &bootstrap.BootConfig{
        AppName:            "ws",
        AppConfigPath:      "config/app.json",
        DatabaseConfigPath: "config/database.json",
        CustomConfigPath:   "config/custom.json",
        EnableServices:     []bootstrap.ServiceType{bootstrap.ServiceTypeWS},
        Router:             router.NewRouter(),
    }
    _, err := bootstrap.Boot(bootCfg)
    if err != nil {
        panic("应用启动失败: " + err.Error())
    }
    select {}
}
```

> 多协议混合（HTTP+WS+gRPC）时，`EnableServices` 列多项，`init()` 监听器注册规则不变。

### 4.3 gRPC 入口

单应用用 `main.go`，多应用用 `api.go`，除 `EnableServices` 改为 `bootstrap.ServiceTypeGRPC` 外其余与 HTTP 入口一致。

```go
// main.go 或 api.go（仅 gRPC 场景）
package main

import (
    "myproject/app/router"       // 单应用
    // "myproject/app/api/router" // 多应用

    "github.com/dfpopp/go-dai/bootstrap"
)

func main() {
    bootCfg := &bootstrap.BootConfig{
        AppName:            "api",
        AppConfigPath:      "config/app.json",
        DatabaseConfigPath: "config/database.json",
        CustomConfigPath:   "config/custom.json",
        EnableServices:     []bootstrap.ServiceType{bootstrap.ServiceTypeGRPC},
        Router:             router.NewRouter(),
    }
    _, err := bootstrap.Boot(bootCfg)
    if err != nil {
        panic("应用启动失败: " + err.Error())
    }
    select {}
}
```

### 4.4 定时任务入口

用 `bootstrap.BootCron`（不校验 `EnableServices`/`Router`），启动后业务自行起 goroutine（见第 12 节）。单应用在 `main.go` 中启动，多应用用独立的 `cron.go`。

```go
// cron.go（多应用）或 main.go（单应用 cron 项目）
package main

import (
    "myproject/app/cron/service"

    "github.com/dfpopp/go-dai/bootstrap"
)

func main() {
    bootCfg := &bootstrap.BootConfig{
        AppName:            "cron",
        AppConfigPath:      "config/app.json",
        DatabaseConfigPath: "config/database.json",
        CustomConfigPath:   "config/custom.json",
    }
    if err := bootstrap.BootCron(bootCfg); err != nil {
        panic("应用启动失败: " + err.Error())
    }
    service.StartAllTasks()
    select {}
}
```

### 4.5 WS 连接事件监听器实现

- 文件固定位置：单应用 `app/init/ws/listener.go`，多应用 `app/<应用>/init/ws/listener.go`。
- 实现 `websocket.ConnectionEventHandler`（`OnConnect`/`OnDisconnect`）。
- 命名用业务名（`CustomConnListener`），禁止 `Listener`/`Handler` 无意义名。
- 只做轻量操作（记日志、读连接属性），**禁止** DB 查询/调外部 API。
- 连接属性绑定用 `SetConnAttr`，**禁止**绑定 `password`/`token`。

```go
// app/init/ws/listener.go（单应用）
// app/ws/init/ws/listener.go（多应用）
package ws

import (
    "github.com/dfpopp/go-dai/logger"
    "github.com/dfpopp/go-dai/websocket"
)

// CustomConnListener WS 连接事件监听器
type CustomConnListener struct{}

// NewCustomConnListener 构造监听器实例
func NewCustomConnListener() *CustomConnListener {
    return &CustomConnListener{}
}

// OnConnect 连接建立时触发
func (l *CustomConnListener) OnConnect(connInfo *websocket.ConnInfo) {
    logger.Info("用户上线", "connID", connInfo.ConnID, "clientIP", connInfo.ClientIP)
}

// OnDisconnect 连接关闭时触发
func (l *CustomConnListener) OnDisconnect(connInfo *websocket.ConnInfo, reason string) {
    logger.Info("用户下线", "connID", connInfo.ConnID, "reason", reason)
}
```

---

## 5. 路由规范

路由集中注册在 `app/<应用>/router/router.go`，实现 `base.BaseRouter` 三个方法。不用的协议嵌入 `base.DefaultBaseRouter` 复用空实现。

### 5.1 路由骨架

```go
// app/router/router.go（单应用）
// app/api/router/router.go（多应用）
package router

import (
    "myproject/app/controller/order"   // 单应用
    "myproject/app/controller/pay"
    "myproject/app/controller/user"
    "myproject/app/middleware"
    // "myproject/app/api/controller/order" // 多应用
    // "myproject/app/api/controller/pay"
    // "myproject/app/api/controller/user"
    // "myproject/app/api/middleware"

    "github.com/dfpopp/go-dai/base"
    "github.com/dfpopp/go-dai/grpc"
    "github.com/dfpopp/go-dai/http"
    "github.com/dfpopp/go-dai/websocket"
)

// ApiRouter 应用路由
// 实现 base.BaseRouter 接口
type ApiRouter struct {
    base.DefaultBaseRouter // 嵌入默认空实现，仅重写需要的协议

    userController  *user.Controller
    orderController *order.Controller
    payController   *pay.Controller
}

// NewRouter 创建路由实例
func NewRouter() *ApiRouter {
    return &ApiRouter{
        userController:  user.NewController(),
        orderController: order.NewController(),
        payController:   pay.NewController(),
    }
}
```

### 5.2 HTTP 路由

- 路径小写 + 斜线，模块前缀：`/user/info`。
- **处理器必须用 `http.ToHTTPHandler(r.ctrl.Method)` 包装**（控制器方法签名 `func(netContext.Context)`，需适配为 `http.HandlerFunc`）。
- 认证接口追加 `middleware.Auth()`；回调接口不追加。

```go
// RegisterHTTPRoutes 注册 HTTP 路由
func (r *ApiRouter) RegisterHTTPRoutes(server *http.Server) {
    // ===== 用户模块 =====
    server.POST("/user/info", http.ToHTTPHandler(r.userController.GetInfo), middleware.Auth())
    server.POST("/user/list", http.ToHTTPHandler(r.userController.GetList), middleware.Auth())

    // ===== 支付回调（无需认证）=====
    server.POST("/pay/alipayPush", http.ToHTTPHandler(r.payController.AlipayPush))
}
```

### 5.3 WebSocket 路由

- action 命名 `模块.方法`：`user.get`、`message.send`。
- **处理器必须用 `websocket.ToWSHandler(r.ctrl.Method)` 包装**（控制器方法签名 `func(netContext.Context)`，需适配为 `websocket.HandlerFunc`）。直接传控制器方法会编译错误。

```go
// RegisterWSRoutes 注册 WebSocket 路由
func (r *ApiRouter) RegisterWSRoutes(server *websocket.Server) {
    server.Register("user.get", websocket.ToWSHandler(r.userController.HandleGet))
    server.Register("message.send", websocket.ToWSHandler(r.userController.HandleSendMessage))
}
```

### 5.4 gRPC 路由

- 必须用 protoc 生成的 `RegisterXXXServiceServer`。
- 传入 `server.GrpcServer`（框架 `*grpc.Server` 的内部字段）。
- **禁止**用 `server.RegisterService(&xxx.ServiceDesc, ...)` 手动注册。

```go
// RegisterGRPCRoutes 注册 gRPC 路由
func (r *ApiRouter) RegisterGRPCRoutes(server *grpc.Server) {
    user.RegisterUserServiceServer(server.GrpcServer, r.userController)
    order.RegisterOrderServiceServer(server.GrpcServer, r.orderController)
}
```

---

## 6. 控制器规范

> **import 别名坑**：当 controller 同时 import `service/<模块>` 与 `model/<模块>`（两个包同名，如都叫 `user`）时，**必须用 import 别名**，否则编译冲突。service 通常需引用 model，同样要注意。下方示例用 `usersvc`/`usermodel` 别名示范。

### 6.1 HTTP 控制器

- 继承 `*base.BaseController`；方法首行 `c.Init(ctx)`。
- 参数：`c.GetPostForm` / `c.GetQuery` / `c.GetPostFormAll` / `c.GetRequestInfo().GetHeader` / `ctx.GetRequestInfo().GetClientIP()`。
- 响应：`c.Success(data)` / `c.DataSuccess(list, count)` / `c.Error(code, msg)` / `c.RespText("success")`。
- 基础校验失败立即 `return`。
- **禁止** Controller 内出现 `GetMysqlDb`/`SetTable`/`Find` 等 DB 操作；**禁止**跨层调 Model。

```go
// app/controller/user/user.go（单应用）
// app/api/controller/user/user.go（多应用）
package user

import (
    usersvc "myproject/app/service/user"       // 单应用
    // usersvc "myproject/app/api/service/user" // 多应用

    "github.com/dfpopp/go-dai/base"
    "github.com/dfpopp/go-dai/netContext"
)

// Controller 用户控制器
// 负责处理用户相关 HTTP 请求
type Controller struct {
    *base.BaseController
    userService *usersvc.Service
}

// NewController 构造控制器实例
func NewController() *Controller {
    return &Controller{
        BaseController: &base.BaseController{},
        userService:    usersvc.NewService(),
    }
}

// GetInfo 获取用户信息
//
// 参数：
//   userId - 用户ID
//
// 返回：
//   用户信息
func (c *Controller) GetInfo(ctx netContext.Context) {
    c.Init(ctx)

    userId := c.GetPostForm("userId")
    if userId == "" {
        c.Error(400, "用户ID不能为空")
        return
    }
    info, err := c.userService.GetUserById(userId)
    if err != nil {
        c.Error(400, err.Error())
        return
    }
    c.Success(info)
}

// GetList 获取用户列表
func (c *Controller) GetList(ctx netContext.Context) {
    c.Init(ctx)

    page := c.GetPostForm("page")
    list, count, err := c.userService.GetUserList(page)
    if err != nil {
        c.Error(500, err.Error())
        return
    }
    c.DataSuccess(list, count)
}
```

### 6.2 WebSocket 控制器

- 继承 `*base.BaseController`；方法首行 `c.Init(ctx)`。
- **WS 消息体为 `{action, request_id, data}`，`data` 是 JSON。取参优先用 `c.BindJSON(&req)` 绑定结构体**；`c.GetPostForm` 仅适用于 data 为表单格式 `key=value` 的场景。
- 推送必须用三参方法：`SendToUser(userID, action, data)` / `SendToConnID` / `SendToUsers` / `SendToConnIDs`。action 格式 `模块.方法`，data 任意类型（框架自动序列化并填 `request_id`）。
- **禁止**直接操作连接 map。

```go
// app/ws/controller/user/user.go
package user

import (
    usersvc "myproject/app/ws/service/user"

    "github.com/dfpopp/go-dai/base"
    "github.com/dfpopp/go-dai/netContext"
)

// getUserReq user.get 消息参数
type getUserReq struct {
    UserId string `json:"userId"`
}

// sendMessageReq message.send 消息参数
type sendMessageReq struct {
    TargetUserID string `json:"targetUserId"`
    FromUserID   string `json:"fromUserId"`
    Content      string `json:"content"`
}

type Controller struct {
    *base.BaseController
    userService *usersvc.Service
}

func NewController() *Controller {
    return &Controller{
        BaseController: &base.BaseController{},
        userService:    usersvc.NewService(),
    }
}

// HandleGet 处理 user.get 消息
func (c *Controller) HandleGet(ctx netContext.Context) {
    c.Init(ctx)

    var req getUserReq
    if err := c.BindJSON(&req); err != nil { // WS data 是 JSON，用 BindJSON
        c.Error(400, "参数解析失败")
        return
    }
    if req.UserId == "" {
        c.Error(400, "用户ID不能为空")
        return
    }
    info, err := c.userService.GetUserById(req.UserId)
    if err != nil {
        c.Error(400, err.Error())
        return
    }
    c.Success(info)
}

// HandleSendMessage 处理 message.send 消息：推送给目标用户
func (c *Controller) HandleSendMessage(ctx netContext.Context) {
    c.Init(ctx)

    var req sendMessageReq
    if err := c.BindJSON(&req); err != nil {
        c.Error(400, "参数解析失败")
        return
    }
    if req.TargetUserID == "" || req.Content == "" {
        c.Error(400, "目标用户或内容不能为空")
        return
    }

    if err := c.SendToUser(req.TargetUserID, "message.recv", map[string]string{
        "from":    req.FromUserID,
        "content": req.Content,
    }); err != nil {
        c.LogError("消息推送失败，targetUserID:", req.TargetUserID, "error:", err)
        c.Error(500, "消息推送失败")
        return
    }
    c.Success(nil, "发送成功")
}
```

### 6.3 gRPC 控制器

- 必须同时继承 `*base.BaseController` 与 `xxx.UnimplementedXXXServer`（缺一不可）。
- 方法签名 `func(ctx context.Context, req *XxxRequest) (*XxxResponse, error)`，用标准 `context.Context`，**禁止** `netContext.Context`。
- 参数直接读 `req.Field`，**禁止** `GetPostForm`。
- 业务错误放进响应 `Msg` 字段并返回 `nil` error：`return &Resp{Code:400, Msg:"..."}, nil`。**禁止**用 `return nil, errors.New(...)` 表达业务错误。
- 成功码 gRPC 用 `0`，HTTP 用 `200`（协议差异，非笔误）。

```go
// app/controller/user/user.go（gRPC 部分，单应用）
// app/api/controller/user/user.go（gRPC 部分，多应用）
import (
    "context"

    guser "myproject/gen/user" // protoc 生成代码
)

type Controller struct {
    *base.BaseController
    guser.UnimplementedUserServiceServer // gRPC 未实现基类（必须继承）
    userService *usersvc.Service
}

// GetUser gRPC 方法
func (c *Controller) GetUser(ctx context.Context, req *guser.UserRequest) (*guser.UserResponse, error) {
    if req.Id <= 0 {
        return &guser.UserResponse{Code: 400, Msg: "用户ID非法"}, nil
    }
    // gRPC 的 req.Id 是 int32，Service 的 GetUserById 入参是 string，需做类型转换
    info, err := c.userService.GetUserById(strconv.FormatInt(int64(req.Id), 10))
    if err != nil {
        return &guser.UserResponse{Code: 500, Msg: err.Error()}, nil
    }
    return &guser.UserResponse{Code: 0, Msg: "操作成功", Data: info}, nil
}
```

> 上例 `strconv.FormatInt` 用于把 gRPC 的 int32 入参转成 Service 需要的 string；按实际 Service 方法签名选择转换方式。gRPC 控制器完整 import 需含 `"context"`、`"strconv"`、`guser "myproject/gen/user"` 及 `usersvc`。

---

## 7. Service 规范

- 继承 `*base.BaseService`；方法首行 `s.Init()`。
- 业务规则校验、幂等检查、事务编排都在这里；多协议复用核心逻辑必须放 Service。
- 错误：`s.LogError(err)` 记录后返回友好 `errors.New("...")`，**禁止**把底层 err 直接抛给 Controller。
- **禁止**依赖协议上下文（`netContext.Context`）——Service 应协议无关；**禁止**直接调第三方 SDK（必须调 `pkg/` 封装层）。

```go
// app/service/user/user.go（单应用）
// app/api/service/user/user.go（多应用）
package service

import (
    "errors"

    usermodel "myproject/app/model/user"       // 单应用
    // usermodel "myproject/app/api/model/user" // 多应用

    "github.com/dfpopp/go-dai/base"
)

// Service 用户服务
// 封装用户核心业务逻辑，HTTP/WS/gRPC 共用
type Service struct {
    *base.BaseService
    userModel *usermodel.Model
}

// NewService 构造服务实例
func NewService() *Service {
    return &Service{
        BaseService: &base.BaseService{},
        userModel:   usermodel.NewModel(),
    }
}

// GetUserById 根据用户ID查询用户
func (s *Service) GetUserById(userId string) (*usermodel.User, error) {
    s.Init()

    if userId == "" {
        return nil, errors.New("用户ID不能为空")
    }
    info, err := s.userModel.GetOne("id,username,email", "id=?", []interface{}{userId})
    if err != nil {
        s.LogError("用户查询失败，userId:", userId, "error:", err)
        return nil, errors.New("用户查询失败")
    }
    if info == nil {
        return nil, errors.New("用户不存在")
    }
    return info, nil
}
```

支付回调幂等示例：

```go
// AlipayPush 支付宝回调业务处理（含幂等）
func (s *Service) AlipayPush(params map[string]string) error {
    s.Init()

    orderStatus := ""
    switch params["trade_status"] {
    case "TRADE_SUCCESS", "TRADE_FINISHED":
        orderStatus = "Y"
    case "TRADE_CLOSED":
        orderStatus = "F"
    default:
        return errors.New("未知的支付状态：" + params["trade_status"])
    }

    orderSn := params["out_trade_no"]
    orderInfo, err := s.orderModel.GetOne("", "order_no=?", []interface{}{orderSn})
    if err != nil {
        return errors.New("订单查询失败")
    }
    if orderInfo == nil {
        return errors.New("订单不存在：" + orderSn)
    }

    // 幂等检查：已处理的订单直接返回
    if orderInfo.PayStatus == "Y" {
        return nil
    }
    return s.orderModel.UpdatePayStatus(orderSn, orderStatus)
}
```

---

## 8. Model 规范

- 继承 `*base.BaseModel`；方法首行 `m.Init()`。
- 用链式方法操作 DB，**禁止**拼 SQL 字符串。
- 结构体含 `DbTag` / `Table` 字段并注释；错误 `m.LogError(err)` + 返回友好信息。

### 8.1 MySQL Model

- 查询：必须 `SetField`（禁止 `*`）、必须带 WHERE + `SetLimit`、必须 `context.WithTimeout`。
- 批量插入必须事务。
- 更新/删除必须带 WHERE，禁止更新主键，重要数据软删除。
- 事务：`db.ToBegin()`（无参）→ `defer` 判 `err` 回滚 → `Commit()`。**禁止**事务内调外部 API。

```go
// app/model/user/user.go（单应用）
// app/api/model/user/user.go（多应用）
package model

import (
    "context"
    "encoding/json"
    "errors"
    "time"

    "github.com/dfpopp/go-dai/base"
)

// User 用户实体
type User struct {
    ID         int64  `json:"id" db:"id"`                  // 用户ID（主键）
    Username   string `json:"username" db:"username"`      // 用户名
    Email      string `json:"email" db:"email"`            // 邮箱
    Status     int    `json:"status" db:"status"`          // 状态：0-禁用，1-正常
    CreateTime string `json:"createTime" db:"create_time"` // 创建时间
}

// Model 用户数据模型
type Model struct {
    *base.BaseModel
    DbTag string // 数据库连接标识
    Table string // 数据表名称
}

// NewModel 构造模型实例
func NewModel() *Model {
    return &Model{BaseModel: &base.BaseModel{}, DbTag: "default", Table: "users"}
}

// GetOne 查询单条用户
//
// 参数：
//   field       - 查询字段
//   whereTpl    - WHERE 模板（占位符 ?）
//   whereValues - WHERE 参数值
func (m *Model) GetOne(field, whereTpl string, whereValues []interface{}) (*User, error) {
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
        db.SetWhere(whereTpl, whereValues...)
    }
    db.SetLimit(0, 1)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    retStr, err := db.Find(ctx)
    if err != nil {
        m.LogError(err)
        return nil, errors.New("数据查询失败")
    }
    if retStr == "" {
        return nil, nil
    }
    var u User
    if err := json.Unmarshal([]byte(retStr), &u); err != nil {
        return nil, errors.New("数据解析失败")
    }
    return &u, nil
}

// UpdatePayStatus 更新支付状态（带 WHERE + 事务）
func (m *Model) UpdatePayStatus(orderNo, status string) error {
    m.Init()

    db, err := m.GetMysqlDb(m.DbTag)
    if err != nil {
        return err
    }
    db.SetTable(m.Table).SetWhere("order_no=?", orderNo)

    if err := db.ToBegin(); err != nil { // MySQL 事务无参
        return err
    }
    defer func() {
        if err != nil {
            db.Rollback()
        }
    }()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err = db.Update(ctx, map[string]interface{}{"pay_status": status})
    if err != nil {
        return errors.New("支付状态更新失败")
    }
    return db.Commit()
}
```

### 8.2 MongoDB Model

- 必须用索引字段查询。
- 事务用 `Begin(ctx)` / `Commit(ctx)` / `Rollback(ctx)`，`defer` 回滚。

```go
// app/model/log/log.go（单应用）
// app/api/model/log/log.go（多应用）
package model

import (
    "context"
    "encoding/json"
    "errors"
    "time"

    "go.mongodb.org/mongo-driver/bson"

    "github.com/dfpopp/go-dai/base"
)

// Log 日志实体
type Log struct {
    UserID string `json:"userId" bson:"user_id"` // 用户ID（索引字段）
    Action string `json:"action" bson:"action"`  // 操作动作
    Time   string `json:"time" bson:"time"`      // 操作时间
}

type Model struct {
    *base.BaseModel
    DbTag string
    Table string
}

func NewModel() *Model {
    return &Model{BaseModel: &base.BaseModel{}, DbTag: "default", Table: "logs"}
}

// FindByUser 按用户ID查询日志（必须用索引字段）
func (m *Model) FindByUser(userId string) ([]Log, error) {
    m.Init()

    db, err := m.GetMongoDb(m.DbTag)
    if err != nil {
        return nil, err
    }
    db.SetTable(m.Table).SetWhere(bson.D{{"user_id", userId}}).SetLimit(0, 100)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    str, err := db.FindAll(ctx).ToString()
    if err != nil {
        m.LogError(err)
        return nil, errors.New("日志查询失败")
    }
    var list []Log
    _ = json.Unmarshal([]byte(str), &list)
    return list, nil
}
```

### 8.3 Redis Model

- key 必须业务前缀 + 冒号：`user:info:123`。**禁止**无前缀、用 `_`/`-` 分隔。
- **必须**设过期时间，禁止 `Set(key, val, 0)`。
- 分布式锁用 `SetNX` + `defer Del`。

```go
// app/model/cache/cache.go（单应用）
// app/api/model/cache/cache.go（多应用）
package model

import (
    "errors"
    "time"

    "github.com/dfpopp/go-dai/base"
)

type Model struct {
    *base.BaseModel
    DbTag string
}

func NewModel() *Model {
    return &Model{BaseModel: &base.BaseModel{}, DbTag: "default"}
}

// SetUserToken 缓存用户 token（必须设过期时间）
func (m *Model) SetUserToken(userId, token string) error {
    m.Init()
    rdb, err := m.GetRedis(m.DbTag)
    if err != nil {
        return err
    }
    return rdb.Set("user:token:"+userId, token, 3600) // 业务前缀 + 冒号 + 过期秒
}

// TryLock 分布式锁防重复提交
func (m *Model) TryLock(orderNum string) (bool, error) {
    m.Init()
    rdb, err := m.GetRedis(m.DbTag)
    if err != nil {
        return false, err
    }
    locked, err := rdb.SetNX("pay:lock:"+orderNum, 1, 10*time.Second)
    if err != nil || !locked {
        return false, errors.New("操作正在处理，请勿重复提交")
    }
    return true, nil
}
```

> Redis 具体方法签名（`Set`/`SetNX`/`Del` 等）以框架 `redisDb` 包源码为准。

### 8.4 ElasticSearch Model

```go
// app/model/doc/doc.go（单应用）
// app/api/model/doc/doc.go（多应用）
package model

import (
    "context"
    "time"

    "github.com/dfpopp/go-dai/base"
)

type Model struct {
    *base.BaseModel
    DbTag string
    Index string
}

func NewModel() *Model {
    return &Model{BaseModel: &base.BaseModel{}, DbTag: "default", Index: "docs"}
}

// Search 全文检索
func (m *Model) Search(keyword string) (string, error) {
    m.Init()
    es, err := m.GetESDb(m.DbTag)
    if err != nil {
        return "", err
    }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    return es.SetIndex(m.Index).Search(ctx, m.StringToSearchFulltextStr(keyword))
}
```

> ES 具体方法签名以框架 `elasticSearch` 包源码为准。

---

## 9. gRPC proto 规范

### 9.1 proto 文件

- 放 `proto/` 目录，文件名用服务名（`user.proto`），禁止 `_service`/`_pb` 后缀、大写。
- service/method 大驼峰；message 以 `Request`/`Response` 结尾；字段小驼峰。

```protobuf
// proto/user.proto
syntax = "proto3";

package user;
option go_package = "myproject/gen/user";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}

message GetUserRequest {
  int32 id = 1;
}

message GetUserResponse {
  int32 code = 1;
  string msg = 2;
  UserData data = 3;
}

message UserData {
  int32 id = 1;
  string username = 2;
  string email = 3;
}
```

### 9.2 生成命令

```bash
protoc --go_out=./gen/user --go_opt=paths=source_relative \
       --go-grpc_out=./gen/user --go-grpc_opt=paths=source_relative \
       proto/user.proto
```

### 9.3 生成代码目录

```
gen/user/user.pb.go
gen/user/user_grpc.pb.go
```

> proto 修改后必须立即生成代码，禁止带病运行。

---

## 10. 中间件规范

- 命名：功能名大驼峰（`Auth`/`RateLimit`/`Logger`/`CORS`——CORS 全大写）。
- 返回 `http.MiddlewareFunc`，洋葱模型，**必须**调用 `next(c)`。
- 不含业务逻辑；类型断言用 `ok` 模式；错误响应用统一 `{code,msg}`。
- 认证中间件必须校验时间戳（5 分钟防重放）+ 签名。
- 框架已默认注册 `CORS()` 与 `Recovery()`，业务不要重复注册。

```go
// app/middleware/http_middleware.go（单应用）
// app/api/middleware/http_middleware.go（多应用）
package middleware

import (
    "math"
    "strconv"
    "time"

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
            // 时间戳校验（防重放，5 分钟有效期）
            requestTime, _ := strconv.ParseInt(t, 10, 64)
            if math.Abs(float64(time.Now().Unix()-requestTime)) > 300 {
                c.JSON(200, map[string]interface{}{"code": 401, "msg": "请求已过期"})
                return
            }
            if sign == "" {
                c.JSON(200, map[string]interface{}{"code": 402, "msg": "签名验证失败"})
                return
            }
            next(c) // 必须调用
        }
    }
}
```

---

## 11. 并发与安全规范

- goroutine 必须用 channel 控制并发数（`make(chan int, N)`），**禁止**无限制创建。
- goroutine 内必须 `recover()`。
- 共享变量必须加锁；channel 用完正确关闭。
- 高并发支付等场景用 Redis 分布式锁防重复提交。
- 类型断言必须 `ok` 模式。
- **禁止**在 goroutine 间共享 Controller/Service 实例（框架按 goroutine ID 隔离上下文）。

```go
channelNum := make(chan int, 10) // 并发数=10
defer close(channelNum)
for _, item := range items {
    channelNum <- 1
    go func(it Item) {
        defer func() {
            <-channelNum
            if r := recover(); r != nil {
                logger.Error(fmt.Errorf("任务异常: %v", r))
            }
        }()
        process(it)
    }(item)
}
```

---

## 12. 定时任务规范

- 独立入口根目录 `cron.go` + `app/cron/{service,model}`，与 API 隔离。
- 每个任务独立 goroutine + `sync.WaitGroup`；单任务异常不得拖垮其他任务。
- 任务间隔：实时 100ms~1s、普通 10s、统计 1min~1h；**禁止** `time.Sleep(0)` 死循环。
- 错误必须记录，**禁止**静默吞掉。

```go
// app/cron/service/doc_sync.go
package service

import (
    "fmt"
    "sync"
    "time"
)

// StartAllTasks 启动所有定时任务
func StartAllTasks() {
    var wg sync.WaitGroup
    wg.Add(1)

    go func() {
        defer wg.Done()
        channelNum := make(chan int, 1) // 并发=1
        defer close(channelNum)
        for {
            channelNum <- 1
            go func() {
                defer func() {
                    <-channelNum
                    if r := recover(); r != nil {
                        fmt.Println("任务异常：", r)
                    }
                }()
                if err := SyncChapterTask(); err != nil {
                    fmt.Println(err.Error())
                }
            }()
            time.Sleep(10 * time.Second)
        }
    }()

    wg.Wait()
}
```

---

## 13. 第三方集成规范

- 第三方代码统一放 `pkg/`，按厂商分模块（`pkg/pay/alipay/`、`pkg/aliSms/`）。
- Service 层只调封装层简洁接口，**禁止**直接调 SDK。
- 第三方接口必须设超时、捕获网络异常、关键业务有重试/降级。
- 支付回调必须验签 + 幂等。

```go
// pkg/pay/alipay/alipay.go
package alipay

// VerifySign 验签封装
func VerifySign(params map[string]string, sign string) (bool, error) { /* ... */ return true, nil }

// app/api/service/pay/pay.go
func (s *Service) AlipayPush(params map[string]string) error {
    s.Init()
    ret, err := alipay.VerifySign(params, params["sign"]) // 调封装层
    if err != nil || !ret {
        return errors.New("签名验证失败")
    }
    // 幂等业务处理...
    return nil
}
```

---

## 14. 错误处理 / 日志 / 配置

### 14.1 错误码
- HTTP：200 成功 / 400 参数 / 401 未认证 / 402 签名失败 / 403 禁止 / 404 不存在 / 429 限流 / 500 服务异常。
- gRPC：0 成功，错误码与 HTTP 一致。
- 错误从 Model → Service → Controller 单向传递，**禁止**跨层。

### 14.2 日志
- `LogInfo` 业务流程、`LogWarn` 降级、`LogError` 异常。
- 第三方回调必须 `LogInfo` 原始参数。
- 敏感信息必须脱敏（密码/token/身份证/信用卡）。
- 格式：`s.LogInfo("描述，key1:", v1, "key2:", v2)`，禁止字符串拼接式日志。

### 14.3 配置读取
- 应用配置 `config.GetAppConfig(appName)`；数据库 `config.GetMysqlConfig()["default"]` 等；自定义见 3.3。
- **禁止**硬编码任何密钥/密码/连接串。

---

## 15. 命名 / 注释 / 格式

- 文件/包/目录：小写 + 下划线，包名 = 目录名。
- 结构体/函数/常量：大驼峰；构造 `NewXxx`；局部变量小驼峰。
- JSON 标签小驼峰，DB 字段蛇形。
- 接口：单方法 `er` 结尾，**禁止** `I` 前缀 / `Interface` 后缀。
- 公开函数必须注释（功能/参数/返回）；结构体与重要字段必须注释。
- 左括号不换行、禁止分号、单行 ≤120 字符。
- **禁止**手动调格式，必须 `gofmt`/`goimports`；导入按「标准库→框架库→第三方→内部包」分组。

---

## 16. 反模式速查（命中即改）

Controller 直连 DB / SQL 拼接 / 类型断言不用 ok / 业务 panic / 支付回调无幂等 / goroutine 无并发控制 / goroutine 无 recover / 硬编码配置 / Service 直调第三方 SDK / 认证无时间戳校验 / DB 无超时 / 回调绑 Auth / 底层错误直返客户端 / 定时任务无间隔循环 / **WS 事件在 main 注册或仅空导入不写 init** / **WS 路由不用 ToWSHandler 包装** / **WS 取 JSON 参数用 GetPostForm** / gRPC 控制器缺继承 / gRPC 手动 RegisterService / **入口文件不放根目录** / **单应用 app/ 下多一层 api/ 目录** / **go build 不带文件名编译根目录多入口** / **入口文件中放业务函数** / **import 同名包不用别名** / 用 shell 命令改文件。

---

## 17. 新增功能模块检查清单

新增一个业务模块（如订单）时，Agent 按此清单逐项创建/检查：

- [ ] 目录：单应用 `controller/order/order.go`、`service/order/order.go`、`model/order/order.go`（均在 `app/` 下）。多应用加一层应用目录 `app/api/controller/order/order.go`。
- [ ] 路由：在 `router/router.go` 的 `ApiRouter` 加 `orderController` 字段，`NewRouter` 初始化，`RegisterHTTPRoutes`/`RegisterWSRoutes`/`RegisterGRPCRoutes` 注册。
- [ ] Controller：继承 `*base.BaseController`，首行 `c.Init(ctx)`，HTTP 用 `ToHTTPHandler`、WS 用 `ToWSHandler` 包装；WS 取参 `BindJSON`。
- [ ] Service：继承 `*base.BaseService`，首行 `s.Init()`，错误 `LogError` + 友好返回，不依赖协议上下文。
- [ ] Model：继承 `*base.BaseModel`，首行 `m.Init()`，链式操作 + 占位符 + `WithTimeout` + `SetLimit`。
- [ ] import：controller/service 同时引用同名 service/model 包时用别名。
- [ ] 命名/注释/格式符合第 15 节，跑 `gofmt`/`goimports`。
- [ ] 配置：如需自定义配置，写入 `config/custom.json` 并用 `GetCustomConfig` 读取。
- [ ] gRPC（如涉及）：proto 放 `proto/`，生成到 `gen/<服务>/`，控制器双继承，路由用 `RegisterXXXServiceServer`。
- [ ] WS（如涉及）：入口文件 `init()` 注册监听器，监听器放 `init/ws/listener.go`。
- [ ] 不碰 `runtime/log/`、`vendor/`、框架源码。

---

*本规则与 [代码规范文档_v2.0.md](file:///d:/golang/project/go-Dai/代码规范文档_v2.0.md) 配套；API 调用以 go-Dai 框架源码为准。*
