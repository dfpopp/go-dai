# AGENTS.md — go-Dai 框架 Agent 编码指南

> 基于 go-Dai 框架开发应用项目的强制规则。不适用于修改框架源码。
> 完整细节见 `.trae/rules/` 下各子文档。

***

## 0. 文件操作铁律

- 修改/创建文件：Edit / Write；读取：Read；查找：Glob；搜索：Grep
- **禁止**任何 Windows shell 命令操作文件（`Set-Content`、`echo >`、`sed` 等）
- 改代码前必须先 Read 目标文件

***

## 1. 调用链路与分层

```
请求 → 路由 → Controller(c.Init) → Service(s.Init) → Model(m.Init) → DB
```

错误反向：Model → Service → Controller。三层首行必须 `c.Init(ctx)` / `s.Init()` / `m.Init()`，继承 `*base.BaseController` / `*base.BaseService` / `*base.BaseModel`。

| 层          | 能做                   | 绝不能做                                       |
| ---------- | -------------------- | ------------------------------------------ |
| Controller | 接收参数、调 Service、返回响应  | ❌直接操作 DB ❌跨层调 Model ❌业务 panic              |
| Service    | 业务规则、幂等、事务编排         | ❌依赖协议 Context ❌直调第三方 SDK ❌反向调 Controller   |
| Model      | 链式 DB 操作、CRUD        | ❌拼 SQL 字符串 ❌反向调上层 ❌无 WHERE 的 Update/Delete |
| Middleware | 认证/限流/日志，必须调 next(c) | ❌含业务逻辑 ❌重复注册默认 CORS/Recovery               |

***

## 2. 绝对禁令（红线）

1. ❌ Windows shell 改文件
2. ❌ Controller 直连 DB
3. ❌ SQL 拼接 / `SetField("*")` / 无 WHERE 的 Update/Delete
4. ❌ 业务代码 panic（仅 Boot 失败允许）
5. ❌ Service 直调第三方 SDK（必须封装到 `pkg/`）
6. ❌ 事务内调外部 API
7. ❌ Redis key 无前缀/不设过期（必须 `业务前缀:子键` + TTL）
8. ❌ goroutine 无并发控制/无 recover（必须 channel + defer recover）
9. ❌ DB 操作无 `context.WithTimeout`（必须 WithTimeout + defer cancel）
10. ❌ WS 事件在 main 注册/空导入不写 init（必须 `func init()` 显式注册）
11. ❌ WS 路由直接传控制器方法/WS 取 JSON 用 GetPostForm（必须 ToWSHandler + BindJSON）
12. ❌ gRPC 缺继承/手动 RegisterService/Go error 表达业务错误（必须双继承 + `return &Resp{}, nil`）
13. ❌ 入口文件不放根目录/入口里放业务函数（必须 `package main`，仅含 init/main）
14. ❌ 多应用 `go build` 不带文件名（必须 `go build -o bin/xxx xxx.go`）
15. ❌ import 同名 service/model 包不用别名（必须 usersvc/usermodel）
16. ❌ Model 的 Table 字段硬编码表前缀（必须只写表名本体，前缀由 database.json 的 `pre` 提供，框架 `SetTable` 自动拼接）

***

## 3. 协议差异速查

| 对比项        | HTTP                           | WebSocket                          | gRPC                                    |
| ---------- | ------------------------------ | ---------------------------------- | --------------------------------------- |
| Context 类型 | `netContext.Context`           | `netContext.Context`               | 标准 `context.Context`                    |
| 参数获取       | `GetPostForm`/`GetQuery`       | **`BindJSON(&req)`**               | 直接读 `req.Field`                         |
| 路由包装       | `http.ToHTTPHandler(c.Method)` | `websocket.ToWSHandler(c.Method)`  | `RegisterXXXServiceServer`              |
| 成功码        | `200`                          | `200`                              | `0`                                     |
| 错误响应       | `c.Error(code, msg)`           | `c.Error(code, msg)`               | `return &Resp{Code:code, Msg:msg}, nil` |
| 推送         | —                              | `SendToUser(userID, action, data)` | —                                       |

***

## 4. 数据库引擎速查

|    引擎   | 取连接               | 查询条件                               | 事务                                             | 关键约束                        |
| :-----: | ----------------- | ---------------------------------- | ---------------------------------------------- | --------------------------- |
|  MySQL  | `GetMysqlDb(tag)` | `SetWhere("字段=?", val)`            | `ToBegin()`/`Commit()`/`Rollback()`（无参）        | 禁止 `SetField("*")`，必须 WHERE；Table 不加前缀；非事务路径自动退避重试连接类瞬时错误（事务不重试），增量写禁止开 `retry_on_invalid_conn` |
| MongoDB | `GetMongoDb(tag)` | `SetWhere(bson.D{{"字段", val}})`    | `Begin(ctx)`/`Commit(ctx)`/`Rollback(ctx)`（有参） | 必须索引字段查询                    |
|  Redis  | `GetRedis(tag)`   | 按 key 操作                           | 分布式锁 `SetNX`                                   | key 前缀 + TTL                |
|    ES   | `GetESDb(tag)`    | `SetIndex().Search(ctx, queryStr)` | 无事务                                            | —                           |
|   PG    | `GetPostgresDb(tag)` | `SetWhere("字段=?", val)`（同 MySQL） | `ToBegin()`/`Commit()`/`Rollback()`（无参） | 占位符 `?` 自动转 `$n`；Insert 含 id 字段时 `RETURNING id` 自动返回自增，否则返回 0；Delete 禁止 LIMIT |
|  CH  | `GetClickhouseDb(tag)` | `SetWhere("字段=?", val)`（同 MySQL） | 无事务（无 ToBegin/Commit/Rollback） | Update/Delete 自动转 `ALTER TABLE`（异步 mutation）；Insert 无自增 ID 返回受影响行数 |

***

## 5. 命名速查

| 类型        | 格式            | 示例                                      |
| --------- | ------------- | --------------------------------------- |
| 目录/包名     | 小写+下划线，包名=目录名 | `app/controller/user/` → `package user` |
| 文件名       | 小写+模块名        | `user.go`、`http_middleware.go`          |
| 结构体/函数/常量 | 大驼峰           | `UserController`、`GetUserInfo`          |
| 构造函数      | `NewXxx`      | `NewController()`、`NewService()`        |
| 局部变量      | 小驼峰           | `userId`                                |
| JSON 标签 | 必须和数据库列名一致 | `json:"create_time"` |
| db 标签 | ❌ 禁止使用，框架不读取 | — |
| Redis key | 业务前缀:子键       | `user:token:123`                        |
| WS action | 模块.方法         | `user.get`、`message.send`               |
| HTTP 路径   | 小写+斜线+模块前缀    | `/user/info`                            |

***

## 6. 错误码与错误处理

| 协议   |  成功 | 参数错 | 未认证 | 签名失败 |  禁止 | 不存在 |  限流 | 服务异常 |
| ---- | :-: | :-: | :-: | :--: | :-: | :-: | :-: | :--: |
| HTTP | 200 | 400 | 401 |  402 | 403 | 404 | 429 |  500 |
| WS   | 200 | 400 | 401 |  402 | 403 | 404 | 429 |  500 |
| gRPC |  0  | 400 | 401 |  402 | 403 | 404 | 429 |  500 |

| 层          | 处理方式                                              |
| ---------- | ------------------------------------------------- |
| Model      | `m.LogError(err); return nil, errors.New("友好描述")` |
| Service    | `s.LogError(...); return errors.New("业务错误描述")`    |
| Controller | `c.Error(code, err.Error())`                      |

***

## 7. 项目骨架

```
myproject/
├── main.go                       # 单应用入口（根目录）
├── api.go / admin.go / ws.go / cron.go   # 多应用入口（根目录）
├── app/
│   ├── controller/ / service/ / model/ / router/ / middleware/   # 单应用
│   ├── api/{controller,service,model,router,middleware}/         # 多应用：C端
│   ├── admin/{...}/ | ws/{...,init/ws/listener.go}/ | cron/{service,model}/
├── config/{app,database,custom}.json
├── pkg/                          # 第三方封装
└── runtime/log/
```

- 编译：`go build -o bin/xxx xxx.go`（多应用必须带文件名）
- 入口文件仅含 `package main`、import、`func init()`、`func main()`
- `BootConfig` 必填：`AppName`、`AppConfigPath`、`DatabaseConfigPath`、`CustomConfigPath`、`EnableServices`、`Router`；推荐 `ShutdownFunc: logger.NewService().Stop`

***

## 8. 日志与配置

- 日志：键值对 `s.LogInfo("描述，key:", v)`，禁止字符串拼接
- 敏感信息脱敏（密码/token/身份证/信用卡），回调必须 `LogInfo` 原始参数
- 配置：`config.GetAppConfig(appName)` / `config.GetMysqlConfig()["default"]` / `config.GetCustomConfig().GetConfig()`
- 禁止硬编码任何密钥/密码/连接串

***

## 9. 关键场景红线

| 场景       | 红线                                                                 |
| -------- | ------------------------------------------------------------------ |
| 支付回调     | 不绑认证 → 验签 → 幂等 → 记录原始参数 → 返回约定格式                                   |
| 定时任务     | 独立 cron.go + `app/cron/` + goroutine + channel + recover + 合理间隔    |
| 第三方集成    | 封装 `pkg/<厂商>/` → Service 只调封装层 → 设超时 → 重试/降级                       |
| WS 监听器   | 入口 `init()` 注册 → `app/init/ws/listener.go` → 只做轻量操作                |
| gRPC 控制器 | 双继承 `BaseController` + `UnimplementedXXXServer` → 业务错误进 Msg 返回 nil |

***

## 10. 新增模块检查清单

- [ ] 目录：`app/{controller,service,model}/<模块>/`（多应用加一层 `app/api/...`）
- [ ] 路由：router.go 注册 HTTP/WS/gRPC 路由
- [ ] Controller：继承 `*base.BaseController`，首行 `c.Init(ctx)`，HTTP 用 `ToHTTPHandler`，WS 用 `ToWSHandler` + `BindJSON`
- [ ] Service：继承 `*base.BaseService`，首行 `s.Init()`，不依赖协议 Context
- [ ] Model：继承 `*base.BaseModel`，首行 `m.Init()`，链式操作 + 占位符 + WithTimeout + SetLimit
- [ ] import 别名：同名 service/model 包必须用别名；`gofmt`/`goimports`

***

> 详细规范见 `.trae/rules/` 下各子文档。API 调用以框架源码为准。

