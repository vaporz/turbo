# 排查手册

这篇按“症状 → 可能原因 → 怎么确认 → 怎么修”组织，覆盖路由、鉴权、绑定、启动 panic、热重载、日志、代码生成、端口与生命周期这些最常见的故障。每一条里的日志文本都来自 turbo 源码，可以直接在日志里搜。

> **版本**：下面提到的 `config.log_level`、路由审计（`route audit: ...`）、`404 no route`、配置稳定等待与空配置拒绝，都是 v0.6.2 起的行为。

## 怎么用这张手册

1. 先确定请求到底有没有到这个进程。turbo 对没匹配上的请求会打 Error 级日志：

   ```text
   turbo: 404 no route for GET /no/such/path, host=localhost:8080, remote=127.0.0.1:53124, user-agent="curl/8.5.0"
   ```

   日志里有这行，说明请求到了、但路由没匹配；没有这行，说明请求根本没进来（端口、实例、前置代理的问题）。

2. 再看启动时打印的路由表，确认你以为存在的那条路由真的注册了：

   ```text
   route: GET /hello -> TestService.SayHello
   turbo: 2 route(s) registered
   ```

3. 还看不出原因时，把级别调到 `debug`，看路由审计逐条明细。

## 所有路由都 404

| 症状 | 可能原因 | 怎么确认 | 怎么修 |
|---|---|---|---|
| 每个路径都 404，进程还活着 | `curl` 打到了别的端口或别的实例 | 在被访问的那个实例日志里找 `404 no route`；同时确认 `HTTP Server started` 这行出现过 | 用配置里的 `config.http_port` 对应端口；多实例时确认负载均衡指向的是哪一台 |
| 每个路径都 404 | 配置被读到空表 | 日志里有 `turbo: ignoring configuration change, it cannot be loaded: ... urlmapping is empty ...` | 一次性写完整份配置；确认 `/hello` 这类路由在 `urlmapping` 下 |
| 每个路径都 404 | `urlmapping` 没被解析出来 | 启动日志里一条 `route: ...` 都没有 | 检查 YAML 缩进：`urlmapping:` 下的每一项以 `- ` 开头，与 `config:` 同级；key 是 `urlmapping` |
| 每个路径都 404 | 请求打到了 HTTP 端口之外（例如 gRPC 端口） | gRPC 端口不会回 HTTP 404，连接表现不同 | 分清 `config.http_port` 与 `config.grpc_service_port` |
| 每个路径都 404 | 前面有反向代理改写了路径 | 直接对实例端口 `curl` 一次同一路径 | 在代理侧修正 path rewrite |

注意：如果配置本身有问题，进程通常根本起不来（见下面的“启动直接 panic”），而不是服务全 404。全 404 更常见的原因是打错了地方。

## 某一条路由 404

| 症状 | 可能原因 | 怎么确认 | 怎么修 |
|---|---|---|---|
| 只有某条路径 404 | 路径拼写或大小写不符 | 对比 `route: ...` 行和 `404 no route` 行里的精确路径 | 改成配置里的写法 |
| 只有某条路径 404 | 路径段数不对 | 例如 `/hello/{your_Name}` 只匹配 `/hello/x`，不匹配 `/hello/a/b` 或 `/hello/` | 按模板的段数调用，或新增一条更宽的映射 |
| 只有某条路径 404 | 末尾斜杠差异 | `/hello/` 和 `/hello` 是两个路径（`pattern.go` 明确说“不做任何规范化”） | 统一写法，或两条都声明 |
| 只有某条路径 404 | 被更早注册的更宽路由吃掉 | `route: ...` 按 `urlmapping` 顺序打印，mux 也按注册顺序匹配 | 把更精确的路由放到更宽的前面 |
| 只有某条路径 404，但方法换了就通 | 方法不匹配 | 方法不对通常返回 405 而不是 404（`test/integration_test.go` 里对 `POST /hello/testtest` 的断言就是 405） | 在 `urlmapping` 第一列补上该方法，用逗号分隔，例如 `GET,POST` |
| 路径匹配了但返回 500 | service 或 method 名写错 | 响应体是 `No such service[...]` 或 `No such method[...]`（生成代码里的默认分支） | 对齐生成代码里的 service / method 名，重新 `turbo generate` |
| 组件（拦截器、处理器）没生效 | 组件声明的匹配规则不是路由 | 打开 `debug` 看 `route audit` 明细里的 `auth=common:[...] + route:[...]`，`route:[...]` 为空说明没有声明匹配上 | 检查组件声明的 pattern；`/*` 才是“所有路径”，`/` 只是根路径 |

组件声明的匹配规则是 turbo 自己的模式语言（`pattern.go`）：`/hello` 只匹配自己，`/hello/{name}` 多一段，`/hello/*` 覆盖自身及以下，`/*` 覆盖全部。同一路由被多条声明命中时，**只有第一条生效**，审计会打一行 Warn：

```text
route audit: GET /hello is matched by 2 interceptor declarations [[A] [B]], only the first one runs
```

## 401 与 403

turbo 本身不做鉴权，状态码来自服务自己的拦截器。排查方向是“拦截器有没有跑”和“错误有没有带状态”。

| 症状 | 可能原因 | 怎么确认 | 怎么修 |
|---|---|---|---|
| 期望 401，实际 500 | 拦截器返回了不带状态的错误 | 日志里 `error in Before(): ...`，响应体是原始错误文本 | 用 `turbo.Errorf(http.StatusUnauthorized, ...)` 或 `turbo.WithStatus(err, 401)`；`StatusOf` 为 0 时默认处理函数回 500 |
| 期望 401，实际 200 | 鉴权拦截器没匹配到这条路由 | `debug` 级别下看 `route audit: GET /x -> Svc.M [auth=..., authenticated]`，没有你的拦截器名 | 补 `interceptor:` 声明；或把它注册成公共拦截器（`SetCommonInterceptor`） |
| 期望 401，启动就被拒 | 声明了 `auth.interceptors`，但有条路由没有鉴权拦截器 | 启动报错 `refusing this configuration: N route(s) would be served without any declared auth interceptor (list a route under auth.public_routes if it is meant to be public):` | 给该路由加鉴权拦截器，或把 `GET /path` 写进 `auth.public_routes` |
| 公共拦截器“没生效” | 它没在配置里出现，但审计说无法核对 | Warn：`route audit: ... [auth=..., cannot be verified: a common interceptor is not registered under a name]` | 用 `RegisterComponent("名字", ...)` 注册，名字要和 `auth.interceptors` 里的一致 |
| 403 出现在不需要鉴权的路径上 | 这条路径没有写进 `auth.public_routes` | 看 `route audit` 明细里有没有 `[public, ...]` 标记 | 把该路由加到 `auth.public_routes`，写法是 `GET /healthz` |
| 拦截器注入的值没落到字段上 | 字段名拼写对不上，或值被别的来源覆盖 | 用 `InjectParam` 的名字和结构体字段名比对 | 三种拼写都会命中：`FieldName`、`fieldname`、`field_name`；优先级是 `injected > path > body > query/form` |
| 拦截器里用 `req.Form.Set` 没效果 | JSON 请求的绑定路径不读 `req.Form` | 对同一条路由分别发 form 和 JSON 请求对比 | 改用 `turbo.InjectParam(req, key, value)`，它是官方替代方案；调用后继续用同一个 `*http.Request` |

## 400 且消息里是 cannot bind

消息格式是：

```text
turbo: cannot bind <字段名> from <来源>: <原因>
```

`<来源>` 只有三种取值：

| 来源 | 含义 | HTTP 状态 |
|---|---|---|
| `query/form parameter` | 来自 query 或表单 | 400 |
| `path parameter` | 来自路由变量 | 400 |
| `injected value` | 服务自己注入的值 | 500（这是服务的错，不是调用方的） |

一个真实例子：

```text
turbo: cannot bind Int64Value from query/form parameter: strconv.ParseInt: parsing "<redacted>": invalid syntax
```

怎么读：

- 字段名告诉你是哪个参数，来源告诉你是哪一类调用方式，原因告诉你为什么转不过去（`invalid syntax`、`cannot unmarshal`、`not supported kind` 之类）。
- 原始值看不到，是因为这段文本既返回给调用方、也会写进服务日志，而参数里可能有 token、验证码或签名。所以值本身被替换掉了。
- `<redacted>` 就是那个占位符。strconv 引号里的值会变成 `"<redacted>"`；`encoding/json` 报错里的数字会变成 `number <redacted>`（源码 `binding.go` 的 `withoutRequestValues`）。
- 缺少参数不是错误：只有“参数存在但用不了”才会报。字段留成零值、响应 200，才是老行为。

JSON 请求体整体坏掉的报错不同，它不会引用具体字段：

```text
turbo: failed to BuildRequest for json api, request body: 6 bytes, error: invalid character 'a' looking for beginning of object key string
```

Thrift 的 JSON 体按参数名绑定，写错键会报 `not fields of the argument Request` 之类的消息，并列出可用的参数名。

## 启动直接 panic

turbo 在启动路径上大量使用 panic（`panicIf`）。输出的第一行以 `panic: ` 开头，后面就是错误本身的文本，栈从第二行开始。常见的有：

| panic 文本 | 原因 | 怎么修 |
|---|---|---|
| `urlmapping is empty, so no route would be served at all (a configuration file read while it is being written looks like this)` | 配置里没有 `urlmapping`，或文件为空 | 补上至少一条路由；确认文件写完整 |
| `invalid json_field_names: "camelCase", expected "proto" or "camel"` | 值不是这两个之一 | 改成 `proto` 或 `camel`，或删掉这个键 |
| `invalid log_level: "verbose", expected one of panic, fatal, error, warn, info, debug, trace` | 值 logrus 解析不了 | 用合法级别；注意同义的 `warning` 也能通过 |
| `no such component: NoSuchInterceptor, forget to register?` | 配置里引用的组件没注册 | 在 `Initializer.InitService` 里 `RegisterComponent("NoSuchInterceptor", ...)`，名字完全一致 |
| `refusing this configuration: 1 route(s) would be served without any declared auth interceptor ...` | 声明了 `auth.interceptors`，但某路由没有任何已声明的鉴权拦截器 | 补拦截器声明，或把路由写进 `auth.public_routes` |
| `[http_port] is required!` | `config.http_port` 缺失或为空 | 填一个端口 |
| `index out of range [2] with length 1`（`appendMap`） | `urlmapping` 的一行字段太少（少于“方法 路径 服务名”三段） | 每行至少写三段：`GET /hello YourService SayHello` |
| `listen tcp :50061: bind: address already in use` | gRPC 端口被占 | 换端口，或结束占用进程 |
| `fileRootPath MUST be an absolute path, got: ...` | `file_root_path` 是相对路径 | 改成绝对路径；注意这个键只在 `turbo create` / `turbo generate` 时被读取 |
| `'file_root_path' in config file is not set!` | 生成代码时缺这个键 | 补上绝对路径 |
| `<业务自己的错误>` | 业务必填项缺失（例如服务自定义的 `jwt_secret`） | turbo 源码里没有 `jwt_secret` 这个键，turbo 不做这类校验；报错来自服务自己的 `InitService` 或拦截器，按业务代码的提示修 |

另外 `environment` 是精确匹配的：`c.Env()` 直接返回配置里的字符串，不 `TrimSpace`、不忽略大小写。写成 `Production` 或带尾随空格都算“非 production”，日志会变成 stderr 上的文本格式而不是 JSON 文件，这一点在排查“日志文件怎么没生成”时很容易误导。

## 热重载没生效

先确认这行出现过：

```text
Reloading configuration...
```

| 症状 | 可能原因 | 怎么确认 | 怎么修 |
|---|---|---|---|
| 改了配置，什么反应都没有 | 改的键本来就要重启 | 启动时的提示行把 `http_port`、`grpc_service_port`、`thrift_service_port`、`environment`、`turbo_log_path`、`log_level` 列在“需要重启”一侧 | 重启进程 |
| 没有任何重载日志 | 文件系统没触发事件 | 改一次配置文件再看；WSL 的 DrvFs 挂载、网络盘、容器里挂载的 ConfigMap 都可能丢事件 | 把配置放在本机 Linux 文件系统上；容器里改配置后重启 Pod |
| 有 `ignoring configuration change, it cannot be loaded:` | 配置读不出来（YAML 坏了、文件为空） | 后面的错误文本会说明原因，常见的是 `urlmapping is empty` | 修好文件后重新保存一次 |
| 有 `configuration reload failed, keeping the running configuration:` | 配置读得出来但装不上，通常是组件名没注册 | 错误文本里是 `no such component: ...` 或有路由未鉴权 | 注册组件、补鉴权声明或 `public_routes` |
| 一直停在旧路由 | 服务只调了 `StartGrpcService`，没起 HTTP | 日志里没有 `HTTP Server started` | 用 `Start` 或 `StartHTTPServer` 启动 HTTP，watcher 是在那里装的 |
| `Stop()` 之后改配置没反应 | 这是设计行为 | 日志里不再出现 `Reloading configuration...` | 正常；已停止的 server 不再处理变更 |

成功的标志是 `Configuration reloaded`。不确定时用轮询而不是固定 sleep 去验证，慢文件系统上切换可能比你以为的慢。

## 日志里没有想要的行

| 症状 | 可能原因 | 怎么确认 | 怎么修 |
|---|---|---|---|
| 什么都没有 | 级别太高 | `config.log_level` 或 `environment` 决定的默认级别 | 临时设 `log_level: debug` |
| 只想看每条路由的审计，看不到 | 逐条审计是 Debug，汇总是 Info | 汇总行 `route audit: N route(s), ...` 在，逐条不在 | 设 `log_level: debug` |
| `production` 下 stderr 什么都没有 | 日志写进了文件 | 看 `config.turbo_log_path` 目录下的 `turbo.log` | 正常行为；日志级别仍是 Info |
| 改了 `log_level` 没变化 | 这个键不参与热重载 | 启动提示行把它列为需要重启 | 重启进程 |
| 日志格式变了 | 调用了 `turbo.SetOutput` | 它会同时把 formatter 换成 `TextFormatter` | 只应在测试里用，用完 `defer turbo.SetOutput(os.Stdout)` 还原 |
| 字段少了一个 | 自定义 `SortingFunc` 在重写名字 | 用 `turbo.CheckSortingFunc` 检查 | 改用 `turbo.SortKeysFirst` |

完整的日志行清单见 [14-logging.md](14-logging.md)。

## turbo generate 失败

| 报错 | 原因 | 怎么修 |
|---|---|---|
| `turbo: protoc is not in PATH, install it before generating (...)` | 缺工具 | 装 protoc；gRPC 还需要 `protoc-gen-go` 和 `protoc-gen-buildfields`，Thrift 需要 `thrift` |
| `--go_out: protoc-gen-go: plugins are not supported; use 'protoc --go-grpc_out=...'` | PATH 里第一个 `protoc-gen-go` 是新版，不接受 `plugins=grpc` | README 的做法：`go install github.com/golang/protobuf/protoc-gen-go@v1.5.1`，再 `export PATH="$GOPATH/bin:$PATH"` |
| `turbo: protoc-gen-go reports ...`（Warn，不是失败） | 检测到新版插件 | 先按提示继续；protoc 真的拒绝时 turbo 会把上面那条修复方法加进错误里 |
| `turbo: -I /path/service.proto is a file, but -I takes the directory that contains your .proto or .thrift files; pass /path instead` | `-I` 传了文件 | 传目录 |
| `turbo: -I /path cannot be read: ...` | 目录不存在或没权限 | 确认路径存在；建议始终用绝对路径（`-I` 的 flag 帮助里也写了 absolute path） |
| `can not find service.yaml in any:` | 配置文件名或位置不对 | `-I` 指向的目录里要有 `service.yaml`，或者按 CLI 约定放在能找到的位置 |
| `missing rpctype (-r)` / `invalid rpctype` | `-r` 缺失或不是 `grpc` / `thrift` | 补 `-r grpc` 或 `-r thrift` |
| `missing .proto file path (-I)` | gRPC 生成时没给 `-I` | 加上 `-I /abs/path/to/protos` |
| `Usage: generate [package_path] -r [grpc\|thrift] -I (absolute_paths_to_proto\|thrift_files)` | 位置参数缺失 | 第一个位置参数是 `package_path` |
| `[X] is not a CamelCase string`（`turbo create`） | 服务名不是 CamelCase | 用 `YourService` 这种写法 |
| `invalid value for -r, should be grpc or thrift`（`turbo create`） | `-r` 值非法 | 改成 `grpc` 或 `thrift` |

生成前先看这行确认工具链版本（后面跟的是 `protoc --version` 的输出）：

```text
turbo: generating with libprotoc 3.x.y
```

## 端口占用与连接未初始化

| 症状 | 可能原因 | 怎么确认 | 怎么修 |
|---|---|---|---|
| 启动 panic，报 `bind: address already in use` | gRPC 或 Thrift 端口被占 | panic 文本里的端口号 | 换端口，或停掉占用进程 |
| 进程起来了但 HTTP 完全不通 | HTTP 端口被占时 `ListenAndServe` 的错误只写日志，不会让进程退出 | 日志里找 `HTTP Server failed to serve: listen tcp :8081: bind: address already in use` | 换端口；把这条日志纳入监控 |
| `grpc connection not initiated!`（Panic 级） | 在 HTTP 服务启动之前调用了 `s.Service("...")` | 调用栈指向 `GrpcServer.Service` | 先 `StartGrpcService` + `StartHTTPServer`，再取服务实例 |
| `thrift connection not initiated!`（Panic 级） | 同上，Thrift 版本 | 调用栈指向 `ThriftServer.Service` | 同上 |
| 后端 RPC 报连接错误 | 后端进程没起，或 `grpc_service_host` / `grpc_service_port` 写错 | 请求返回的错误文本来自底层 gRPC | 先用 `grpc_service_host:grpc_service_port` 能连通，再启 turbo |

## 长连接与超时

turbo 自己设置的超时只有一处，其余都交给底层默认值：

| 项 | turbo 的行为 | 影响 |
|---|---|---|
| gRPC 出站调用的 context | 生成的 switcher 把 `req.Context()` 传给 RPC，`turbo.CallOptions` 默认只返回 `grpc.Header` / `grpc.Trailer` / `grpc.Peer` | 默认没有 deadline；可以在拦截器里给请求换一个带超时的 context，或覆盖 `CallOptions` 加 `grpc.WaitForReady` 之类的选项 |
| Thrift 出站调用的 context | 生成的 switcher 用 `ctx := context.Background()`，不传请求 context | Thrift 调用拿不到请求级取消，也没有 deadline |
| HTTP 服务端超时 | `http.Server` 只设置了 `Addr` 和 `Handler`，没有 `ReadTimeout` / `WriteTimeout` / `IdleTimeout` | 慢客户端可以一直占着连接 |
| 关闭 HTTP | `httpServer.Shutdown` 用 5 秒超时的 context | 超时后 `Shutdown` 返回错误，但代码没有处理，进程继续 |
| 关闭 gRPC | `GracefulStop()`，没有 deadline | 会等在途 RPC 结束；卡住的 RPC 会拖住关闭 |
| 关闭 Thrift | `thriftServer.Stop()` | 直接停 |

给 gRPC 调用加超时可以放在公共拦截器里，因为生成代码取的是请求的 context：

```go
func (i *timeoutInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	_ = cancel // 真实项目里要把 cancel 存起来，在 After 里调用
	*req = *req.WithContext(ctx)
	return nil
}
```

`turbo.CallOptions` 也是公开变量，可以覆盖它来加 gRPC 调用选项，但它只影响 `...grpc.CallOption`，不能替代上面的 context 超时。

### impl 里返回的错误一律变成 500

**症状**：RPC 方法实现里 `return nil, turbo.Errorf(http.StatusForbidden, "...")`，客户端却收到
`500 {"code":500,"msg":"内部服务器错误"}`。

**原因**：HTTP 层与实现之间是一次真实 RPC，RPC 会把 error 序列化成 status，`turbo.StatusOf`
拿不到状态码，于是按 500 处理。

**修法**：实现里用响应消息自己的 `code` / `msg` 字段表达业务错误；需要真实 HTTP 状态码的判定
（401/403）放到拦截器里做。机制见 [10-errors.md](10-errors.md) 的「在哪里能用 `turbo.Errorf`」。

## 相关阅读

- [14-logging.md](14-logging.md)：每条日志的文本与级别
- [15-hot-reload.md](15-hot-reload.md)：重载覆盖范围
- [16-auth-and-route-audit.md](16-auth-and-route-audit.md)：`auth.interceptors` 与路由审计
- [11-binding.md](11-binding.md)：绑定来源与优先级
- [12-code-generation.md](12-code-generation.md)：`turbo create` / `turbo generate` 的参数
- [17-testing.md](17-testing.md)：怎么为上面这些行为写回归测试
