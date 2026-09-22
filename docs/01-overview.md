# Turbo 总览

这篇讲清三件事：Turbo 在你系统里的位置、它启动时做了什么、一个 HTTP 请求在它内部经过哪些步骤。
读完你就有了后面所有细节的坐标。

## 1. Turbo 解决什么问题

假设你已经有（或准备写）一个 gRPC / Thrift 服务。RPC 的调用方只能是"会写 RPC 客户端的程序"，
而现实里还有一堆只会发 HTTP 的调用方：浏览器、微信小程序、第三方支付/企业微信/对象存储的回调、
`curl`、运维脚本。

Turbo 的做法是**不碰你的 RPC 服务**，而是加一层可配置的网关：

```
浏览器 / 小程序 / 回调 / curl
        │  HTTP (form / query / path / JSON)
        ▼
┌─────────────────────────────────────────────┐
│ Turbo HTTP 层                               │
│   urlmapping 路由 → 组件链 → 参数绑定        │
└───────────────┬─────────────────────────────┘
                │  RPC（gRPC 或 Thrift，真实网络调用）
                ▼
┌─────────────────────────────────────────────┐
│ 你的 RPC 服务（方法实现仍然只有一份）        │
└─────────────────────────────────────────────┘
```

要点：

- **路由与调用规则写在配置里**（`service.yaml` 的 `urlmapping`），加一个 HTTP 接口往往只改 proto/thrift 与一行配置；
- **配置支持热重载**，改路由、改组件不需要重启进程；
- **横切逻辑用组件**（拦截器/前后处理器/劫持器/转换器），与业务实现分开；
- HTTP 层与 RPC 服务可以**同一个进程**跑，也可以分成两个进程（见第 2 节）。

## 2. 三种部署形态

`turbo create` 生成的项目里同时有这三个入口，选哪个取决于你要独立部署还是要省一个进程：

| 入口 | 内容 | 什么时候用 |
|---|---|---|
| `main.go` | RPC 服务 + HTTP 网关，同一进程 | 最省事；服务本身没有独立扩容需求时首选 |
| `grpcservice/<name>.go`（thrift 为 `thriftservice/<name>.go`） | 只启动 RPC 服务 | RPC 服务要独立部署/独立扩缩容时 |
| `grpcapi/<name>api.go`（thrift 为 `thriftapi/<name>api.go`） | 只启动 HTTP 网关（通过 RPC 连到别处的服务） | 网关要多实例、RPC 服务在别处时 |

三个入口用的是同一份 `service.yaml`；单独跑网关时，配置里的 `grpc_service_host` / `grpc_service_port`
指向真正的 RPC 服务地址。

## 3. 启动顺序

以 gRPC 的单进程形态 `s.Start(...)` 为例（`grpcserver.go`、`server.go`）：

1. `NewGrpcServer(initializer, configFilePath)`：读配置并**校验**（配置不合法直接 `panic`，快速失败）；
   同时初始化 logger（级别由 `environment` 与 `config.log_level` 决定）。
2. `Start(...)`：
   1. `initializer.InitService(s)` —— **注册组件**的地方（组件名与 `service.yaml` 里的名字对应）；
   2. 启动 RPC 服务：监听 `grpc_service_port`，`registerServer` 注册服务实现，并打开 gRPC reflection；
   3. 启动 HTTP 服务：连接 RPC（`grpc_service_host:grpc_service_port`）、构造路由表
      （`urlmapping` → gorilla/mux router）、**做一次启动路由审计**，然后 `ListenAndServe`；
   4. `watchConfigReload(s)` —— 监听配置文件变化，之后每次变更都会重建路由表与组件。

Thrift 的形态多一步细节：`ThriftServer.Start` 在启动 RPC 服务之后固定 `time.Sleep(time.Second)` 再起
HTTP 层；Thrift 服务端用的是 `TMultiplexedProcessor`（按服务名注册），客户端用
`TBinaryProtocolFactoryDefault` + `NewTMultiplexedProtocol`（见 [13-grpc-thrift.md](13-grpc-thrift.md)）。

> **配置不合法时**：启动阶段直接 panic（进程起不来）；热重载阶段则记录 error 并**保留正在生效的配置**。
> 详见 [15-hot-reload.md](15-hot-reload.md)。

## 4. 一次请求的完整生命周期

下面是 `runtime.go` 里 `handler()` 的真实顺序，编号对应源码位置。假设请求命中了
`GET /api/v1/devices/{device_code}`：

1. **路由匹配**：`currentHandler` 取出当前 router（读锁保护，热重载时会被整体替换），
   gorilla/mux 按注册顺序找到第一条匹配的路由，进入 `handler(s, serviceName, methodName)`。
2. **固定组件快照**：`copyComponentsPtr` 把当前的 `*Components` 放进 `req.Context()`，
   保证一个请求从开始到结束用的是**同一份**组件（热重载不会中途换掉它）。
3. **解析表单**：`parseRequestForm` 解析 query/form（原始 body 在解析后仍可读）。
4. **组装拦截器链**：`getInterceptors` = **全局拦截器（`SetCommonInterceptor`）＋ 该路由声明的拦截器**，
   顺序即数组顺序。
5. **`Before` 链**：`doBefore` 依次调用；**任何一个 `Before` 返回 error 就截断**后面的拦截器，
   进入第 8 步的错误处理。
6. **`doRequest`**：
   1. **劫持器**：如果该路由挂了 hijacker，调用它并由它自己写响应，**到此结束**（不走 RPC、不走前后处理器）；
   2. **前处理器**：`doPreprocessor`，出错 → 错误处理；
   3. **RPC 调用**：`switcherFunc(...)`（即生成的 `gen.GrpcSwitcher` / `gen.ThriftSwitcher`）里
      **绑定参数**（见 [11-binding.md](11-binding.md)）并发起真实 RPC，出错 → 错误处理；
   4. **后处理器**：`doPostprocessor` 拿到响应与错误，出错 → 错误处理；
   5. **写响应**：`writeResponse` 把 RPC 响应序列化成 JSON（受 `filter_proto_json*`、`json_field_names` 影响）。
7. **`After` 链**：`doAfter` 只对**第 5 步成功的那些**拦截器调用（失败点及其之后的不会执行）。
8. **错误处理**：`errorHandlerFunc()` —— 默认实现用 `turbo.StatusOf(err)` 决定 HTTP 状态码，
   没有状态码时用 500；`errorhandler: <组件名>` 可以换成你自己的。

两个常被问到的推论：

- **鉴权拦截器失败时，排在它后面的拦截器不会执行**（第 5 步的截断）。所以"访问日志"这类拦截器要么放全局，
  要么放在鉴权之前；`SetCommonInterceptor` 装的全局拦截器天然排在最前。
- **hijacker 一旦命中就短路**（第 6.1 步），所以需要 JSON 响应的接口不要挂 hijacker；反过来，
  必须返回**非 JSON**（纯文本、图片、HTML）的接口只能靠 hijacker。

## 5. 术语表

| 术语 | 含义 |
|---|---|
| `service.yaml` | Turbo 的配置：路由表、组件声明、端口、日志等 |
| `urlmapping` | 一行一条路由：`METHOD /path ServiceName MethodName` |
| route / 路由 | 一条 `urlmapping`；一条路由可以有多个 HTTP 方法 |
| component / 组件 | 拦截器、前处理器、后处理器、劫持器、转换器、错误处理器的统称；单例、并发调用 |
| global / common interceptor | `SetCommonInterceptor` 装的拦截器，对所有路由生效，排在路由级拦截器之前 |
| switcher | 生成代码 `gen.GrpcSwitcher` / `gen.ThriftSwitcher`，按服务名+方法名绑定参数并调用 RPC |
| client | `GrpcClient` / `ThriftClient` 返回的 map：服务名 → RPC 客户端实例 |
| binding / 绑定 | 把 HTTP 请求里的值填进 RPC 请求消息（或参数列表） |
| injected | 由服务端代码通过 `turbo.InjectParam` 放进请求的值，优先级最高 |
| hijack / 劫持 | 由 hijacker 完全接管响应，不再走 RPC |
| route audit | 启动/重载时的路由审计，按 `auth.interceptors` / `auth.public_routes` 判断有没有漏配鉴权 |

## 6. 支持矩阵

| 维度 | 支持情况 |
|---|---|
| RPC 类型 | gRPC、Thrift（二选一，由 `-r grpc\|thrift` 与服务配置决定） |
| 请求编码 | `application/x-www-form-urlencoded`（query/form）、`application/json`、路径参数 |
| 响应编码 | JSON（protobuf/thrift 结构 → JSON），或由 hijacker 完全自定义 |
| 多服务 | 一个网关可以同时代理多个 RPC 服务（`*_service_name` 逗号分隔 + client map） |
| 热重载 | 路由表、组件、`errorhandler`、`filter_proto_json`、`json_field_names`、`auth`；端口与日志设置需要重启 |
| 最低 Go 版本 | 1.27.1（见 [18-deployment.md](18-deployment.md)） |

## 7. Turbo 不做什么

明确边界可以省下很多猜测：

- **不做鉴权**：Turbo 只负责"谁来调用"这件事留给你的拦截器；它提供的是**声明哪些拦截器算鉴权**、
  以及"漏配就拒绝启动"的审计（[16-auth-and-route-audit.md](16-auth-and-route-audit.md)）。
- **不做健康检查**：`/hello` 这类探活接口是你自己写在 `urlmapping` 里的普通路由。
- **不做服务发现**：RPC 地址来自配置，`grpcClient` 里也留了注释说明这一点。
- **不生成业务实现**：`turbo create` 只生成骨架与示例方法。
- **不做请求内容日志**：错误消息里刻意不包含请求体与参数值（[10-errors.md](10-errors.md)）。

## 相关阅读

- [02-getting-started.md](02-getting-started.md) —— 动手跑起来
- [05-components.md](05-components.md) —— 组件与生命周期细节
- [11-binding.md](11-binding.md) —— 参数到底怎么绑定
- [15-hot-reload.md](15-hot-reload.md) —— 热重载的边界
