# 附录 C：导出 API 索引

本文列出 turbo 模块中**给使用者**的导出标识符，按用途分组。列表按 v0.6.2 的源码逐个核对，
排除了 `_test.go` 与 `testprotos.pb.go` 这类测试夹具。"使用者"一列的含义：

- **常用**：业务代码或 `InitService` 里会直接写。
- **偶尔**：有明确场景才用。
- **内部/生成代码**：由 turbo 自身或 `turbo generate` 产出的代码调用，一般不用手写。

文档篇名都是相对文件名；标识符的签名以源码为准。

## 1. 服务启动与生命周期

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `Servable` | 服务器抽象：`Service`、`ServerField`、`Stop`、`RegisterComponent` | [05-components.md](05-components.md) | 偶尔 |
| `Initializable` | 启动/停止钩子：`InitService(s Servable) error`、`StopService(s Servable)` | [05-components.md](05-components.md) | 常用 |
| `Server` | 服务器公共字段与方法的载体，内嵌在 `GrpcServer`/`ThriftServer` 里 | [02-getting-started.md](02-getting-started.md) | 常用 |
| `GrpcServer` | gRPC 版服务器，内嵌 `*Server` | [13-grpc-thrift.md](13-grpc-thrift.md) | 常用 |
| `ThriftServer` | Thrift 版服务器，内嵌 `*Server` | [13-grpc-thrift.md](13-grpc-thrift.md) | 常用 |
| `NewGrpcServer(initializer Initializable, configFilePath string) *GrpcServer` | 创建 gRPC 服务器并加载配置、初始化日志 | [02-getting-started.md](02-getting-started.md) | 常用 |
| `NewThriftServer(initializer Initializable, configFilePath string) *ThriftServer` | 创建 Thrift 服务器 | [13-grpc-thrift.md](13-grpc-thrift.md) | 常用 |

`Server` 的导出字段与方法：

| 标识符 | 说明 | 使用者 |
|---|---|---|
| `Server.Config` | 当前生效的 `*Config`，热重载会整体替换 | 偶尔 |
| `Server.Components` | 当前生效的 `*Components` | 偶尔 |
| `Server.Initializer` | 注册进来的 `Initializable` | 偶尔 |
| `Server.RegisterComponent(name string, component interface{})` | 按名字注册组件，配置里的名字就是它 | 常用 |
| `Server.Component(name string) (interface{}, error)` | 按名字取组件，找不到时报 `no such component` | 偶尔 |
| `Server.ServerField() *Server` | 返回自身，供 `Servable` 使用 | 偶尔 |
| `Server.Stop()` | 优雅停机 | 常用 |
| `Server.Service() interface{}` | 占位实现（`GrpcServer`/`ThriftServer` 覆盖它） | 内部 |

`GrpcServer` / `ThriftServer` 的导出方法（`Service`、`ServerField`、`Stop` 与上表同名，语义是各自的客户端/字段）：

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `GrpcServer.Start(clientCreator grpcClientCreator, sw switcher, registerServer func(*grpc.Server))` | 同时启动 gRPC 服务端与 HTTP 网关 | [13-grpc-thrift.md](13-grpc-thrift.md) | 常用 |
| `GrpcServer.StartHTTPServer(clientCreator grpcClientCreator, sw switcher)` | 只启动 HTTP 网关，连已有 gRPC 上游 | [13-grpc-thrift.md](13-grpc-thrift.md) | 常用 |
| `GrpcServer.StartGrpcService(registerServer func(*grpc.Server))` | 只启动 gRPC 服务端 | [13-grpc-thrift.md](13-grpc-thrift.md) | 偶尔 |
| `GrpcServer.Service(serviceName string) interface{}` | 取 gRPC 客户端实例 | [13-grpc-thrift.md](13-grpc-thrift.md) | 偶尔 |
| `ThriftServer.Start(clientCreator thriftClientCreator, sw switcher, registerTProcessor processorRegister)` | 同时启动 Thrift 服务端与 HTTP 网关 | [13-grpc-thrift.md](13-grpc-thrift.md) | 常用 |
| `ThriftServer.StartHTTPServer(clientCreator thriftClientCreator, sw switcher)` | 只启动 HTTP 网关 | [13-grpc-thrift.md](13-grpc-thrift.md) | 常用 |
| `ThriftServer.StartThriftService(registerTProcessor processorRegister)` | 只启动 Thrift 服务端 | [13-grpc-thrift.md](13-grpc-thrift.md) | 偶尔 |
| `ThriftServer.Service(serviceName string) interface{}` | 取 Thrift 客户端实例 | [13-grpc-thrift.md](13-grpc-thrift.md) | 偶尔 |

`Interceptor`、`Preprocessor`、`Postprocessor`、`Hijacker` 还各有一个空的 `ServeHTTP` 方法，
只是为了让它们能当 `http.Handler` 传递，调用它没有任何效果。

`Start` / `StartHTTPServer` 的参数类型（`grpcClientCreator`、`thriftClientCreator`、`switcher`、
`processorRegister`）是**未导出**的，签名里出现只是文档效果：实际传的是生成代码里的
`component.GrpcClient`、`gen.GrpcSwitcher`、`impl.RegisterServer` 这类函数。

## 2. 配置

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `Config` | 配置对象，内嵌 `viper.Viper`（因此也有 Viper 的全部导出方法） | [03-service-yaml.md](03-service-yaml.md) | 偶尔 |
| `Config.File` | 当前配置文件的路径 | [15-hot-reload.md](15-hot-reload.md) | 偶尔 |
| `NewConfig(rpcType, configFilePath string) *Config` | 读取配置；`rpcType` 为 `grpc` 或 `thrift`，同时设置全局 `RpcType` | [03-service-yaml.md](03-service-yaml.md) | 偶尔 |
| `GetWD() string` | 返回进程当前工作目录，读失败时 panic | [18-deployment.md](18-deployment.md) | 偶尔 |
| `RpcType`（var string） | 最近一次 `NewConfig` 设置的 rpc 类型 | [03-service-yaml.md](03-service-yaml.md) | 内部 |

`Config` 的导出方法：

| 标识符 | 说明 |
|---|---|
| `Env()` | 返回 `environment` |
| `LogLevel()` | 返回 `config.log_level`，未设置时为 `""` |
| `ErrorHandler()` | 返回 `errorhandler` 组件名 |
| `FileRootPath()` | 返回 `file_root_path`；为空或非绝对路径时 panic |
| `PackagePath()` | 返回 `package_path`；为空时 panic |
| `ServiceRootPath()` | `FileRootPath()` + `/` + `PackagePath()` |
| `GrpcServiceNames()` / `ThriftServiceNames()` | 按逗号切分服务名列表 |
| `GrpcServiceHost()` / `GrpcServicePort()` / `ThriftServiceHost()` / `ThriftServicePort()` | 上游地址与端口 |
| `HTTPPort()` | 返回 `http_port`；缺失或为空时 panic |
| `JSONFieldNames()` | `proto` 或 `camel` |
| `FilterProtoJson()` / `FilterProtoJsonEmitZeroValues()` / `FilterProtoJsonInt64AsNumber()` | 三个响应补丁开关 |

> **版本**：`Config.LogLevel()` 与 `config.log_level` 自 v0.6.2 起存在。
> `Config.loadXxx` 系列（`loadServiceConfig`、`loadUrlMap`、`loadMappings`、`loadConfigs`、
> `loadComponents`、`loadConvertor`、`loadFieldMapping`）与 `validate`、`authConfig` 都未导出，使用者不用。

## 3. 组件

| 标识符 | 签名或成员 | 说明 | 详见 | 使用者 |
|---|---|---|---|---|
| `Interceptor` | `Before`、`After` 各返回 `error` | 包住请求前后 | [06-interceptor.md](06-interceptor.md) | 常用 |
| `BaseInterceptor` | 空实现的 `Before`/`After` | 嵌入它只覆写需要的方法 | [06-interceptor.md](06-interceptor.md) | 常用 |
| `Interceptors` | `[]Interceptor`，`ServeHTTP` 为空实现 | 一组拦截器 | [06-interceptor.md](06-interceptor.md) | 偶尔 |
| `Preprocessor` | `func(http.ResponseWriter, *http.Request) error` | RPC 调用之前 | [07-preprocessor-postprocessor.md](07-preprocessor-postprocessor.md) | 常用 |
| `Postprocessor` | `func(http.ResponseWriter, *http.Request, interface{}, error) error` | RPC 响应之后 | [07-preprocessor-postprocessor.md](07-preprocessor-postprocessor.md) | 常用 |
| `Hijacker` | `func(http.ResponseWriter, *http.Request)` | 完全接管请求 | [08-hijacker.md](08-hijacker.md) | 偶尔 |
| `Convertor` | `func(r *http.Request) reflect.Value` | 按类型直接构造值 | [09-convertor.md](09-convertor.md) | 偶尔 |
| `ErrorHandlerFunc` | `func(http.ResponseWriter, *http.Request, error)` | 统一错误响应 | [10-errors.md](10-errors.md) | 常用 |

`Components`（通过 `Server.Components` 或 `Servable.ServerField().Components` 拿到）：

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `WithErrorHandler(ErrorHandlerFunc)` | 注册错误处理函数 | [10-errors.md](10-errors.md) | 常用 |
| `SetCommonInterceptor(...Interceptor)` | 安装全局拦截器，配置重载会保留 | [04-routing.md](04-routing.md) | 常用 |
| `Intercept(methods []string, urlPattern string, list ...Interceptor)` | 注册路由级拦截器链 | [06-interceptor.md](06-interceptor.md) | 常用 |
| `SetPreprocessor([]string, string, Preprocessor)` | 注册 preprocessor | [07-preprocessor-postprocessor.md](07-preprocessor-postprocessor.md) | 常用 |
| `SetPostprocessor([]string, string, Postprocessor)` | 注册 postprocessor | [07-preprocessor-postprocessor.md](07-preprocessor-postprocessor.md) | 常用 |
| `SetHijacker([]string, string, Hijacker)` | 注册 hijacker | [08-hijacker.md](08-hijacker.md) | 常用 |
| `SetConvertor(string, Convertor)` | 按 Go 类型名注册 convertor | [09-convertor.md](09-convertor.md) | 常用 |
| `Reset()` | 清空全部组件映射 | [17-testing.md](17-testing.md) | 偶尔 |
| `CommonInterceptors()` | 取全局拦截器 | [04-routing.md](04-routing.md) | 内部 |
| `Interceptors(*http.Request)` | 取该请求的路由级拦截器 | [04-routing.md](04-routing.md) | 内部 |
| `Preprocessor` / `Postprocessor` / `Hijacker` | 按请求取组件 | [04-routing.md](04-routing.md) | 内部 |
| `Convertor(theType string)` | 按类型名取 convertor | [09-convertor.md](09-convertor.md) | 内部 |

## 4. 参数绑定与请求上下文

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `InjectParam(req *http.Request, key, value string)` | 记录服务端确认过的值；就地改写 `req`，优先级最高 | [11-binding.md](11-binding.md) | 常用 |
| `InjectedValue(fieldName string, req *http.Request) (string, bool)` | 读取注入值，也兼容直接放进 context 的字符串 | [11-binding.md](11-binding.md) | 常用 |
| `BuildStructErr(s Servable, theType reflect.Type, theValue reflect.Value, req *http.Request) error` | 把请求里的值绑定到结构体，遇到不可用值时报错 | [11-binding.md](11-binding.md) | 内部/生成代码 |
| `BuildStruct(...)` | 同上的旧版：只记日志、字段留零值 | [11-binding.md](11-binding.md) | 内部（已废弃） |
| `BuildRequest(s Servable, v proto.Message, req *http.Request) error` | 生成代码用来构造 gRPC 请求消息 | [11-binding.md](11-binding.md) | 内部/生成代码 |
| `BuildArgs(...) ([]reflect.Value, error)` | 生成代码用来构造 Thrift 参数列表 | [11-binding.md](11-binding.md) | 内部/生成代码 |
| `BuildThriftRequest(...) ([]reflect.Value, error)` | 生成代码用来构造 Thrift 请求 | [11-binding.md](11-binding.md) | 内部/生成代码 |

## 5. gRPC metadata 与调用选项

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `CallOptions`（var） | 生成代码调用的函数变量，可整体覆盖以定制 `grpc.CallOption` | [13-grpc-thrift.md](13-grpc-thrift.md) | 偶尔 |
| `WithCallOptions(req, header, trailer, peer)` | 把 header/trailer/peer 放进请求 context | [13-grpc-thrift.md](13-grpc-thrift.md) | 内部/生成代码 |
| `GrpcMetadataHeader(ctx) *metadata.MD` | 取 RPC 响应 header | [13-grpc-thrift.md](13-grpc-thrift.md) | 偶尔 |
| `GrpcMetadataTrailer(ctx) *metadata.MD` | 取 RPC 响应 trailer | [13-grpc-thrift.md](13-grpc-thrift.md) | 偶尔 |
| `GrpcMetadataPeer(ctx) *peer.Peer` | 取 RPC 对端信息 | [13-grpc-thrift.md](13-grpc-thrift.md) | 偶尔 |

## 6. 错误

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `Errorf(status int, format string, args ...interface{}) error` | 生成带 HTTP 状态码的 error | [10-errors.md](10-errors.md) | 常用 |
| `WithStatus(err error, status int) error` | 附加状态码；nil 保持 nil，已有状态码不覆盖 | [10-errors.md](10-errors.md) | 常用 |
| `StatusOf(err error) int` | 取状态码，无则 0，会穿透 `%w` 包装 | [10-errors.md](10-errors.md) | 常用 |

> **版本**：`Errorf` / `WithStatus` / `StatusOf` 自 v0.6.0 起可用。

## 7. JSON 序列化

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `Marshaler` | JSON 序列化器；字段 `FilterProtoJson`、`EmitZeroValues`、`Int64AsNumber`、`UseJSONNames` | [14-logging.md](14-logging.md) | 偶尔 |
| `Marshaler.JSON(v interface{}) ([]byte, error)` | proto 消息走 jsonpb（可加补丁），其它走 `encoding/json` | [14-logging.md](14-logging.md) | 偶尔 |
| `Marshaler.FilterJsonWithStruct(jsonBytes []byte, structObj interface{}) ([]byte, error)` | protobuf 补丁：补零值、int64 转数字、补 nil 指针 | [14-logging.md](14-logging.md) | 偶尔 |
| `IsCamelCase(name string) bool` | 是否 CamelCase | [12-code-generation.md](12-code-generation.md) | 偶尔 |
| `IsNotCamelCase(name string) bool` | 是否不是 CamelCase | [12-code-generation.md](12-code-generation.md) | 偶尔 |
| `ToSnakeCase(str string) string` | camelCase 转 snake_case | [11-binding.md](11-binding.md) | 偶尔 |

## 8. 日志

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `SetOutput(out io.Writer)` | 运行时改 turbo 日志输出目标，并把格式改成 TextFormatter | [14-logging.md](14-logging.md) | 偶尔 |
| `ContextHook` | logrus hook，给每条日志加 `file`、`func`、`line`；只在非 production 环境安装 | [14-logging.md](14-logging.md) | 偶尔 |
| `ContextHook.Levels()` / `ContextHook.Fire(*logger.Entry) error` | logrus hook 接口实现（`logger` 是源码里对 logrus 的别名） | [14-logging.md](14-logging.md) | 内部 |
| `SortKeysFirst(first ...string) func([]string)` | 生成只重排、不丢字段的 logrus 排序函数 | [14-logging.md](14-logging.md) | 偶尔 |
| `CheckSortingFunc(sortKeys func([]string), keys ...string) error` | 在测试里校验排序函数只是重排 | [17-testing.md](17-testing.md) | 偶尔 |

## 9. 代码生成

| 标识符 | 说明 | 详见 | 使用者 |
|---|---|---|---|
| `Creator` | 新建工程；字段 `RpcType`、`PkgPath`、`FileRootPath` | [12-code-generation.md](12-code-generation.md) | 偶尔 |
| `Creator.CreateProject(serviceName string, force bool)` | 生成可运行的 HTTP + gRPC/Thrift 工程 | [12-code-generation.md](12-code-generation.md) | 偶尔 |
| `Generator` | 生成 stub 与 switcher；字段 `RpcType`、`PkgPath`、`ConfigFileName`、`Options`、`FilePaths` | [12-code-generation.md](12-code-generation.md) | 偶尔 |
| `Generator.Generate()` | 按 `RpcType` 依次生成 stub、字段映射与 switcher | [12-code-generation.md](12-code-generation.md) | 偶尔 |
| `Generator.GenerateProtobufStub()` | 调 `protoc --go_out=plugins=grpc` | [12-code-generation.md](12-code-generation.md) | 内部 |
| `Generator.GenerateGrpcSwitcher()` | 生成 `gen/grpcswitcher.go` | [12-code-generation.md](12-code-generation.md) | 内部 |
| `Generator.GenerateThriftStub()` | 调 `thrift --gen go` | [12-code-generation.md](12-code-generation.md) | 内部 |
| `Generator.GenerateBuildThriftParameters()` | 生成 `gen/thrift/build.go` 并运行，产出字段映射 | [12-code-generation.md](12-code-generation.md) | 内部 |
| `Generator.GenerateThriftSwitcher()` | 生成 `gen/thriftswitcher.go` | [12-code-generation.md](12-code-generation.md) | 内部 |
| `ValidateIncludePaths(paths []string) error` | 校验 `-I` 是目录而不是文件 | [12-code-generation.md](12-code-generation.md) | 偶尔 |

## 10. 命令行包 `turbo/cmd`

| 标识符 | 说明 | 详见 |
|---|---|---|
| `Execute() error` | 执行根命令；由 `turbo/main.go` 调用 | [12-code-generation.md](12-code-generation.md) |
| `RootCmd`（var） | cobra 根命令，`Version` 为 `v0.6.2` | [12-code-generation.md](12-code-generation.md) |
| `RpcType`（var string） | `-r/--rpctype` 绑定到的变量 | [12-code-generation.md](12-code-generation.md) |
| `FilePaths`（var []string） | `-I/--include-path` 绑定到的变量 | [12-code-generation.md](12-code-generation.md) |
| `FileRootPath`（var string） | `turbo create -p/--rootpath` 绑定到的变量 | [12-code-generation.md](12-code-generation.md) |

## 11. 生成代码里的标识符

这些不在 turbo 包里，而是 `turbo generate` 写进 `<package_path>/gen` 的产物：

| 标识符 | 说明 | 详见 |
|---|---|---|
| `gen.GrpcSwitcher` | gRPC 版运行时分发函数，传给 `StartHTTPServer` / `Start` | [12-code-generation.md](12-code-generation.md) |
| `gen.ThriftSwitcher` | Thrift 版运行时分发函数 | [12-code-generation.md](12-code-generation.md) |
| `gen.buildStructArg`（Thrift） | Thrift switcher 内部按类型名构造参数 | [12-code-generation.md](12-code-generation.md) |

`protoc-gen-buildfields` 是 `turbo generate` 调用的 protoc 插件（`--buildfields_out`），
它把 `*Request` 消息写成 `gen/grpcfields.yaml`；它是 `package main`，没有供业务代码调用的导出 API。

## 12. 明确不导出的内部实现

以下名字在源码里存在但不能被使用者引用，索引里不必查：`RpcType` 之外的内部常量与键名
（`jsonFieldNamesProto`、`urlServiceMaps`、`authInterceptorsKey` 等）、`statusError`、
`defaultErrorHandler`、`defaultInitializer`、`switcher`、`Components` 的小写方法
（`setCommonInterceptor`、`intercept`、`setComponent` 等）、`Config` 的 `loadXxx` 与 `validate`。
直接用 `go doc github.com/vaporz/turbo` 可以看到编译器认可的完整导出列表。

## 相关阅读

- [01-overview.md](01-overview.md) —— 这些 API 在请求生命周期里的位置
- [05-components.md](05-components.md) —— 组件类型与注册接口
- [11-binding.md](11-binding.md) —— `InjectParam` / `InjectedValue` 的用法
- [10-errors.md](10-errors.md) —— `Errorf` / `WithStatus` / `StatusOf` 的语义
- [14-logging.md](14-logging.md) —— `SetOutput` / `SortKeysFirst` / `CheckSortingFunc`
- [13-grpc-thrift.md](13-grpc-thrift.md) —— `CallOptions` 与 metadata 相关 API
