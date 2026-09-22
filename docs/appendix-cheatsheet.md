# 附录 B：速查表

所有内容按 v0.6.2 的源码整理；每个标识符都可以在仓库里 grep 到。
配置键的完整说明见 [03-service-yaml.md](03-service-yaml.md)，组件语义见 [05-components.md](05-components.md)，
绑定细节见 [11-binding.md](11-binding.md)，错误见 [10-errors.md](10-errors.md)。

## 1. 配置键速查

下表键都在 `config:` 段内（`Config.loadConfigs` 用 `GetStringMapString("config")` 读取）。
"热重载"一列里，"是"表示改文件即生效，"否"表示需要重启进程。

| 键 | 作用 | 缺省 | 热重载 |
|---|---|---|---|
| `environment` | 只有 `production` 会改日志格式、输出目标与默认级别 | 空（走非 production 分支） | 否 |
| `file_root_path` | 包根目录，必须绝对路径；与 `package_path` 拼成 `ServiceRootPath` | 必填，访问值为空时 panic | 否（仅代码生成用） |
| `package_path` | 相对 `file_root_path` 的包路径 | 必填，为空时 `PackagePath()` panic | 否（仅代码生成用） |
| `turbo_log_path` | turbo 日志目录，固定写 `turbo.log` | 空，即进程启动时的工作目录 | 否 |
| `log_level` | turbo 自身日志级别；`panic`/`fatal`/`error`/`warn`/`info`/`debug`/`trace` | 空，由 `environment` 决定 | 否 |
| `http_port` | 网关对外 HTTP 端口 | 必填，`HTTPPort()` 为空时 panic | 否 |
| `grpc_service_name` | gRPC 上游服务名，逗号分隔 | 空 | 否（运行时由 clientCreator 决定） |
| `grpc_service_host` | gRPC 上游地址 | 空 | 否 |
| `grpc_service_port` | gRPC 上游端口 | 空 | 否 |
| `thrift_service_name` | Thrift 上游服务名，逗号分隔 | 空 | 否（运行时由 clientCreator 决定） |
| `thrift_service_host` | Thrift 上游地址 | 空 | 否 |
| `thrift_service_port` | Thrift 上游端口 | 空 | 否 |
| `filter_proto_json` | 是否启用 protobuf 响应补丁 | `false` | 是 |
| `filter_proto_json_emit_zerovalues` | 补零值字段；仅当 `filter_proto_json: true` 时判断 | 该场景下 `true` | 是 |
| `filter_proto_json_int64_as_number` | int64 输出为数字而非字符串；同上 | 该场景下 `true` | 是 |
| `json_field_names` | JSON 键拼写：`proto` 或 `camel`，别的值加载时报错 | `proto` | 是 |

组件与路由段（`Config.loadComponents` / `loadUrlMap`）：

| 段 | 行的格式 | 热重载 |
|---|---|---|
| `urlmapping` | `METHOD[,METHOD...] PATH ServiceName MethodName`（4 段） | 是 |
| `interceptor` | `METHOD[,METHOD...] URL_PATTERN Name[,Name...]`（3 段） | 是 |
| `preprocessor` | `METHOD[,METHOD...] URL_PATTERN Name`（3 段） | 是 |
| `postprocessor` | `METHOD[,METHOD...] URL_PATTERN Name`（3 段） | 是 |
| `hijacker` | `METHOD[,METHOD...] URL_PATTERN Name`（3 段） | 是 |
| `convertor` | `GoTypeName Name`（2 段） | 是 |
| `errorhandler` | 组件名字符串（标量，不是列表） | 是 |
| `auth.interceptors` | 鉴权拦截器名列表 | 是 |
| `auth.public_routes` | `"METHOD /path"` 列表 | 是 |

> **版本**：`config.log_level` 自 v0.6.2 起可用。`environment` 仍然决定格式（`production` 为 JSON）
> 与输出目标（`turbo_log_path`），`log_level` 只覆盖级别。

> **版本**：`auth.interceptors` 自 v0.6.0 起会强制校验；v0.6.2 起逐条审计明细降到 debug，
> info 只保留一行汇总。

## 2. 组件类型速查

| 类型 | 签名 | 什么时候用 |
|---|---|---|
| `Interceptor`（接口） | `Before(http.ResponseWriter, *http.Request) error`、`After(...) error` | 需要包住整个请求（鉴权、注入、计时） |
| `BaseInterceptor`（结构体） | 空实现 `Before`/`After` | 嵌入它，只覆写需要的方法 |
| `Interceptors`（`[]Interceptor`） | `ServeHTTP` 为空实现 | 需要把一组拦截器当整体传递时 |
| `Preprocessor` | `func(http.ResponseWriter, *http.Request) error` | 调用 RPC 之前的校验/改写 |
| `Postprocessor` | `func(http.ResponseWriter, *http.Request, interface{}, error) error` | 拿到 RPC 响应后、写 JSON 之前 |
| `Hijacker` | `func(http.ResponseWriter, *http.Request)` | 完全接管请求，不走 RPC、不做 JSON 序列化 |
| `Convertor` | `func(r *http.Request) reflect.Value` | 由代码直接构造某类型的值，跳过参数绑定 |
| `ErrorHandlerFunc` | `func(http.ResponseWriter, *http.Request, error)` | 统一错误响应 |

组件注册与查找（`Components` 的方法；`RegisterComponent` 注册的实例要与 `SetCommonInterceptor` 传入的是同一个，审计才能把全局拦截器对上名字）：

| 方法 | 作用 |
|---|---|
| `SetCommonInterceptor(...Interceptor)` | 安装全局拦截器，每个请求先跑；配置重载会保留 |
| `Intercept([]string, string, ...Interceptor)` | 按方法与 URL 模式注册路由级拦截器 |
| `SetPreprocessor` / `SetPostprocessor` / `SetHijacker` | 注册对应组件 |
| `SetConvertor(string, Convertor)` | 按 Go 类型名注册 Convertor |
| `WithErrorHandler(ErrorHandlerFunc)` | 注册错误处理函数 |
| `CommonInterceptors` / `Interceptors` / `Preprocessor` / `Postprocessor` / `Hijacker` / `Convertor` | 按请求查找（生成代码与内部用） |
| `Reset()` | 清空所有组件映射 |

`Server.RegisterComponent(name, component)` 按名字注册（配置里的组件名就是它）；
`Server.Component(name)` 按名字取回，找不到时返回 `no such component: <name>, forget to register?`。
函数类型的组件必须先转成具名类型，例如 `turbo.Preprocessor(myFunc)`。

## 3. 参数绑定优先级与拼写归一

优先级（从高到低）：

```text
injected > path > body > query/form
```

| 来源 | 读取方式 | 谁写的 |
|---|---|---|
| injected | `InjectParam` 的 store，或请求 context 里的字符串 | 服务端自己（拦截器等） |
| path | `mux.Vars(req)` 里的路由变量 | 路由匹配 |
| body | JSON body（`jsonpb.Unmarshaler`，或 Thrift 按参数名） | 客户端 |
| query/form | `req.Form`（查询串 + urlencoded 表单） | 客户端 |

JSON 请求的应用顺序是：先 body，再用 query/form 补 body 没提到的字段，再 path 覆盖，
最后 injected 覆盖一切。表单请求走 `findValue`，同样按上表优先级。

拼写归一规则：

- `normalizeKey(k) = strings.ToLower(strings.ReplaceAll(k, "_", ""))`，因此
  `deviceCode`、`device_code`、`DeviceCode`、`DEVICE_CODE` 是同一个参数。
- 字段探测顺序 `lookupKeys` 是 `[Go 字段名, 全小写, ToSnakeCase]`；
  同名参数有多种拼写时，先按这三种精确拼写查，再用归一比较，避免结果受 map 迭代顺序影响。
- `parseRequestForm` 会把表单里的全大写键合并到小写键，并把路由变量以小写键放进 `req.Form`，路由值排在前面。
- 响应侧：`json_field_names: proto` 写 proto 字段名，`camel` 写 protobuf 定义的 JSON 名。

> **版本**：绑定失败不再静默留零值、注入值不再被请求覆盖、JSON 请求也会读 URL，均是 v0.6.0 起的行为。

## 4. 状态码与错误速查

| API | 作用 |
|---|---|
| `turbo.Errorf(status, format, args...)` | 生成一个带 HTTP 状态码的 error |
| `turbo.WithStatus(err, status)` | 给 err 附加状态码；err 为 nil 时返回 nil，已有状态码时保留原值 |
| `turbo.StatusOf(err)` | 取回状态码，没有则返回 0；会穿透 `%w` 包装 |

默认 error handler（`defaultErrorHandler`）：把 `err.Error()` 以 error 级别写日志，
状态码为 `StatusOf` 的结果，为 0 时用 `http.StatusInternalServerError`，响应体是 `err.Error()`。

| 触发点 | 状态码 | 说明 |
|---|---|---|
| 绑定 query/form 或 path 参数失败 | 400 | 消息形如 `turbo: cannot bind X from query/form parameter: ...` |
| 绑定注入值失败 | 500 | 同一消息里 source 是 `injected value`：这是服务端的错 |
| JSON body 解析失败 | 400 | `turbo: failed to BuildRequest for json api, request body: N bytes, error: ...` |
| Thrift JSON body 解析失败/出现未知参数名 | 400 | 未知键会被拒绝，不会静默忽略 |
| 未匹配任何路由 | 404 | `notFoundHandler`，同时打一条 error 日志 |
| HTTP 方法不匹配 | 405 | 由 gorilla/mux 返回 |
| 业务自定义 | 任意 | 用 `turbo.Errorf` / `turbo.WithStatus` |

> **版本**：v0.6.2 起错误消息不再回显请求里的参数值：被引用的值替换为 `<redacted>`，
> 只保留字段名、来源和失败原因（如 `invalid syntax`）。

## 5. CLI 速查

根命令 `turbo` 的版本由 `RootCmd.Version` 决定（v0.6.2），`turbo --version` 的输出里带有 `v0.6.2`。

`turbo create`（别名 `c`）：

| 参数 | 简写 | 缺省 | 含义 |
|---|---|---|---|
| `package_path ServiceName` | 位置参数 | 无 | 前两个位置参数；`ServiceName` 必须是 CamelCase，否则报错 |
| `--rpctype` | `-r` | `grpc` | `grpc` 或 `thrift`，其它值报 `invalid value for -r` |
| `--force` | `-f` | `false` | 覆盖已有文件，不再交互式确认删除目录 |
| `--rootpath` | `-p` | `.` | 新包生成到哪个目录下（会转成绝对路径） |

`turbo generate`（别名 `g`）：

| 参数 | 简写 | 缺省 | 含义 |
|---|---|---|---|
| `package_path` | 位置参数 | 无 | 第一个位置参数，缺失时报 `Usage: generate [package_path] ...` |
| `--rpctype` | `-r` | 空，必填 | `grpc` 或 `thrift` |
| `--include-path` | `-I` | 空数组 | 可重复；传的是**目录**（含 `.proto`/`.thrift` 的目录），传文件会被 `ValidateIncludePaths` 拒绝 |

`-I` 传文件名时 `turbo generate` 会直接返回一句人话，而不是让 protoc 报错。
`turbo generate -r grpc` 需要 `protoc`、`protoc-gen-go`、`protoc-gen-buildfields` 在 PATH 中；
thrift 只需要 `thrift`。

## 6. 日志行速查

| 真实文本（msg） | 级别 | 含义 |
|---|---|---|
| `Starting Turbo...` | info | `Start` 开头 |
| `Starting GRPC Service...` / `GRPC Service started` | info | gRPC 服务端 |
| `Starting HTTP Server...` / `HTTP Server started` | info | HTTP 网关 |
| `[grpc]connecting addr:<host:port>` | info | gRPC 客户端建连 |
| `connecting thrift addr: <host:port>` | debug | Thrift 客户端建连 |
| `interceptor:[...]` / `preprocessor:[...]` / `postprocessor:[...]` / `hijacker:[...]` / `convertor:[...]` | info | 从配置装载的每条组件声明 |
| `errorhandler:<name>` | info | 从配置装载的错误处理函数 |
| `route: <METHODS> <path> -> <Service>.<Method>` | info | 每条注册的路由 |
| `turbo: N route(s) registered` | info | 路由总数（等于 urlmapping 行数） |
| `route audit: N route(s), A authenticated, P public, U unprotected` | info | 路由审计汇总 |
| `route audit: ... [public, auth=...]` / `[auth=..., authenticated]` | debug | 逐条审计明细 |
| `route audit: ... [UNPROTECTED: ...]` | error | 该路由没有任何鉴权拦截器 |
| `route audit: ... is matched by N interceptor declarations ..., only the first one runs` | warn | 多条声明命中同一路由 |
| `binding: X <- injected "...", overriding the query/form value "..."` | warn | 注入值覆盖了 query/form |
| `binding: X <- injected "...", overriding the request body value` | warn | 注入值覆盖了 body |
| `turbo: 404 no route for <METHOD> <path>, host=..., remote=..., user-agent=...` | error | 没匹配到路由（不含查询串） |
| `error in Before(): ...` / `turbo: error in After(): ...` | error | 拦截器返回错误 |
| `turbo: encounter error in preprocessor for <url>, error: ...` | error（经 errorhandler 输出） | preprocessor 失败（包装后仍保留状态码） |
| `turbo: a configuration change reloads ... need a restart` | info | 启动时提示热重载范围 |
| `Reloading configuration...` / `Configuration reloaded` | info | 热重载成功 |
| `turbo: configuration reload failed, keeping the running configuration: ...` | error | 重载失败，沿用旧配置 |
| `turbo: ignoring configuration change, it cannot be loaded: ...` | error | 新配置无法解析，忽略本次变更 |
| `Http Server stopped` / `Grpc Server stopped` / `Thrift Server stopped` | info | 停机 |
| `turbo: generating with <protoc 版本>` | info | 代码生成 |
| `turbo: protoc-gen-go reports ...` | warn | 检测到新版 protoc-gen-go |

## 7. 常用片段

写一个鉴权拦截器并注入身份：

```go
type AuthInterceptor struct{ turbo.BaseInterceptor }

func (i *AuthInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	uid, err := verify(req.Header.Get("X-Auth-Token"))
	if err != nil {
		return turbo.Errorf(http.StatusUnauthorized, "bad token")
	}
	turbo.InjectParam(req, "userId", uid) // injected 优先级最高
	return nil
}
```

在请求里读取注入值并绑定：

```go
uid, ok := turbo.InjectedValue("userId", req)
if !ok {
	return turbo.Errorf(http.StatusUnauthorized, "no user")
}
req_ := &proto.GetProfileRequest{UserId: uid}
```

安装全局拦截器（必须在服务器启动前；配置重载会保留）：

```go
s.RegisterComponent("LogInterceptor", logInterceptor) // 同一个实例
s.ServerField().Components.SetCommonInterceptor(logInterceptor)
```

自定义 errorhandler：

```go
func myErrorHandler(resp http.ResponseWriter, req *http.Request, err error) {
	status := turbo.StatusOf(err)
	if status == 0 {
		status = http.StatusInternalServerError
	}
	http.Error(resp, err.Error(), status)
}
s.RegisterComponent("myErrorHandler", turbo.ErrorHandlerFunc(myErrorHandler))
```

取 gRPC metadata（在 postprocessor 或 `After` 里，RPC 已经返回）：

```go
header := turbo.GrpcMetadataHeader(req.Context())
trailer := turbo.GrpcMetadataTrailer(req.Context())
peer := turbo.GrpcMetadataPeer(req.Context())
fmt.Println("header:", (*header)["headerval"], "peer:", peer.Addr)
```

设置 turbo 日志级别：配置里写 `config.log_level: warn`（v0.6.2 起，需要重启）；
运行时要改输出可用 `turbo.SetOutput(os.Stdout)`（同时把格式改成 TextFormatter）。
turbo 的 logger 就是 logrus 的 standard logger，需要更细的控制可以调用
`logger.SetLevel(logger.WarnLevel)`（`github.com/sirupsen/logrus`）。

写一个 hijacker：

```go
func PaymentNotifyHijacker(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	fmt.Fprint(resp, `{"result":"ok"}`)
}
s.RegisterComponent("PaymentNotifyHijacker", turbo.Hijacker(PaymentNotifyHijacker))
```

## 相关阅读

- [README.md](README.md) —— 文档索引与阅读路线
- [03-service-yaml.md](03-service-yaml.md) —— 配置键的完整说明
- [11-binding.md](11-binding.md) —— 绑定优先级与拼写归一的推导
- [10-errors.md](10-errors.md) —— 状态码与错误消息
- [14-logging.md](14-logging.md) —— 日志行与级别
- [19-troubleshooting.md](19-troubleshooting.md) —— 按症状查
