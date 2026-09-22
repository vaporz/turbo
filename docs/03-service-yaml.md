# service.yaml 配置参考

`service.yaml` 是 turbo 唯一的配置文件，它同时决定网关自己怎么跑（`config` 段）和 HTTP 请求怎么被翻译成 RPC（顶层路由与组件段）。这篇按源码逐键说明每个键的作用、类型、缺省值、是否参与热重载，以及写错时会在哪里、以什么形式失败。

## 这个文件什么时候被读

`NewGrpcServer` / `NewThriftServer` 构造时就调用 `NewConfig` 读文件。读取流程是 `Config.loadServiceConfig` -> `Config.loadServiceConfigErr`，内部依次执行 `loadUrlMap`、`loadConfigs`、`loadComponents`、`validate`。

失败分两种场景，行为完全不同：

- 启动阶段：`loadServiceConfig` 用 `panicIf` 把错误升级成 panic，进程直接起不来。
- 热重载阶段：`watchConfig` 用 `loadServiceConfigErr` 拿到错误后只记一行 `turbo: ignoring configuration change, it cannot be loaded: `，整个变更被忽略，正在服务的配置不动。

`config` 段整体由 `Config.loadConfigs` 读入，实现是 `c.GetStringMapString("config")`，所以段内每个值读出来都是字符串。这一点决定了布尔型开关的写法：比较的是字符串，不是 YAML 布尔。

## config 段

### environment

作用：选择日志的形态，是 `production` 还是其它值。

- 类型：字符串。
- 缺省值：未设置时 `Config.Env()` 返回 `""`，`initLogger` 走 else 分支，也就是开发形态：`TextFormatter`、输出到 `os.Stderr`、级别 `debug`、并挂上 `ContextHook`（在每条日志里加 `file` / `func` / `line`）。
- 写成 `production` 时：输出到文件（见 `turbo_log_path`）、`JSONFormatter`、级别 `info`。
- 是否参与热重载：否。`initLogger` 只在 `NewGrpcServer` / `NewThriftServer` 里调用一次。

### turbo_log_path

作用：`production` 下 turbo 自身日志的输出目录，实际写入文件是 `<turbo_log_path>/turbo.log`。

- 类型：字符串（目录路径）。
- 缺省值：空或未设置时用 `os.Getwd()` 当前工作目录；相对路径会被拼成 `<cwd>/<turbo_log_path>`，然后 `path.Clean`。
- 行为：用 `os.MkdirAll(logPath, 0755)` 建目录，用 `os.OpenFile(..., O_CREATE|O_WRONLY|O_APPEND, 0666)` 追加打开。建目录或打开失败都 panic。
- 只在 `environment: production` 时生效；非 production 时该键被完全忽略，日志走 stderr。
- 是否参与热重载：否。

### log_level

> **版本**：v0.6.2 起支持 `config.log_level`。

作用：指定 turbo 自身日志的级别。它只是覆盖 `environment` 给出的默认级别，不改格式、不改输出位置。

- 类型：字符串。
- 取值：交给 logrus 的 `logger.ParseLevel`，接受 `panic`、`fatal`、`error`、`warn`、`warning`、`info`、`debug`、`trace`，大小写不敏感（logrus 内部先 `strings.ToLower`）。
- 缺省值：空，表示由 `environment` 决定（production 为 `info`，其它为 `debug`）。
- 校验：`Config.validate` 会拒绝无法解析的值，错误文本是 `invalid log_level: "verbose", expected one of panic, fatal, error, warn, info, debug, trace`。注意 `warning` 是 logrus 接受的别名，但不在错误文本列举里。
- 是否参与热重载：否，`initLogger` 只跑一次，改了要重启。

和业务自己的顶层 `log_level` 是两个键：turbo 只读 `config.log_level`，顶层那个留给业务代码。

### http_port

作用：HTTP 网关监听端口。`startHTTPServer` 把它拼成 `":" + port` 作为 `http.Server.Addr`。

- 类型：整数，但按字符串读入后用 `strconv.ParseInt(p, 10, 64)` 解析。
- 必填。缺失或空串时 `Config.HTTPPort()` panic，文本是 `[http_port] is required!`。
- 写错的表现：能解析成整数才正常；写成非数字时 `ParseInt` 的 error 只被 `logErrorIf` 记一条日志，返回值是 0，服务器于是在 `:0`（随机端口）上监听，能起来但不在你期望的端口。
- 是否参与热重载：否。地址在 `http.Server` 构造时就固定了。

### file_root_path / package_path

作用：代码生成用的路径。`ServiceRootPath()` 返回 `FileRootPath() + "/" + PackagePath()`，`gen/grpcfields.yaml`、`gen/thriftfields.yaml`（`loadFieldMapping`）和 `turbo generate` 的输出目录都由它算出来。

- 类型：字符串（`file_root_path` 是绝对路径，`package_path` 是包路径，例如 `github.com/you/yourservice`）。
- 校验：`FileRootPath()` 在空值时 panic `'file_root_path' in config file is not set!`，非绝对路径时 panic `fileRootPath MUST be an absolute path, got: ...`；`PackagePath()` 在空值时 panic `'package_path' in config file is not set!`。这两个检查是在真正访问时发生的：运行的 HTTP 网关不会读它们，`turbo generate` / `turbo create` 才会。
- 是否参与热重载：否，而且对运行中的网关完全没有影响。

### grpc_service_name / grpc_service_host / grpc_service_port

作用：gRPC 客户端与服务端的目标。`startGrpcHTTPServerInternal` 用 `GrpcServiceHost()+":"+GrpcServicePort()` 去 `grpc.Dial`，`startGrpcServiceInternal` 用 `GrpcServicePort()` 监听。

- 类型：字符串。`grpc_service_name` 是逗号分隔的列表，`GrpcServiceNames()` 实现是 `strings.Split(names, ",")`，**不做 TrimSpace**，所以 `A, B` 的第二个名字是 `" B"`（带空格）。多个服务名请写成 `A,B`，逗号后不要加空格。
- 缺省值：无。空 host 或空 port 会拼出 `":"`，`grpc.Dial` 与监听会以各自的方式失败。
- 作用范围：`grpc_service_name` 只在代码生成时被读（`generator.go` / `creator.go`），它决定生成出来的 `GrpcClient` 返回的 map 有哪些 key，以及 switcher 里 `if serviceName == "..."` 的分支；运行中的网关不读它。`grpc_service_host` / `grpc_service_port` 在启动时各读一次。
- 是否参与热重载：否。客户端只在启动时 dial 一次（`grpcClient.init` 开头有 `if g.grpcServiceMap != nil { return }`），端口也被监听套接字占住。

### thrift_service_name / thrift_service_host / thrift_service_port

与上一组对称，`ThriftServiceNames()` 同样是逗号切分且不 TrimSpace；`startThriftHTTPServerInternal` 用 `ThriftServiceHost()+":"+ThriftServicePort()` 建 socket，`startThriftServiceInternal` 用 `ThriftServicePort()` 监听。

- 是否参与热重载：否，原因同上。

### json_field_names

> **版本**：v0.6.0 起支持 `config.json_field_names`。

作用：决定 turbo 序列化 RPC 响应时用哪套字段名。`proto` 用 proto 字段名（`owner_openid`、`Int64Value`），`camel` 用 protobuf 定义的 JSON 名（`ownerOpenid`、`int64Value`）。

- 类型：字符串。
- 取值：`proto` 或 `camel`，比较时先 `TrimSpace` 再 `EqualFold`，所以 `  Camel ` 也接受。
- 缺省值：`proto`（`JSONFieldNames()` 里除 `camel` 之外一律回落到 `proto`）。
- 校验：其它值在配置加载阶段就被拒绝，文本是 `invalid json_field_names: "camelCase", expected "proto" or "camel"`。
- 是否参与热重载：是。`writeResponse` 每次请求都从当前 `Config` 指针读这个值。

### filter_proto_json

作用：打开后，turbo 会对 protobuf 序列化出来的 JSON 做一次“补字段”处理，让本来被 `omitempty` 省略的字段也出现在响应里。`Marshaler` 的字段由 `FilterProtoJson()`、`FilterProtoJsonEmitZeroValues()`、`FilterProtoJsonInt64AsNumber()` 三个值填充。

- 类型：布尔（读入后以字符串参与比较）。YAML 把 `true` / `True` / `TRUE` 解析成布尔真，`GetStringMapString` 再经 `cast.ToString` 转成字符串 `"true"`，所以这几种写法都算打开；`yes`、`on`、`1` 在 YAML 里分别是字符串和整数，转出来是 `"yes"`、`"on"`、`"1"`，不算打开。
- 缺省值：关闭。

子项只在 `filter_proto_json` 为 `"true"` 时才有意义，父项关闭时两个子项都返回 false：

- `filter_proto_json_emit_zerovalues`：是否补出零值字段。
- `filter_proto_json_int64_as_number`：int64 是否输出成 JSON 数字。
- 两者缺省都是“开”：父项打开且子项未设置时返回 true；只有显式写成 `"false"` 才关闭（同样是比较字符串 `"false"`）。
- 是否参与热重载：是，和 `json_field_names` 一样在 `writeResponse` 里按当前配置读取。

## 顶层段

### urlmapping

作用：路由表，每行把一条 HTTP 路由映射到一个 RPC 方法。格式见 `04-routing.md`。

- 类型：字符串列表（`GetStringSlice`），每行 `METHOD /path ServiceName MethodName`。
- 解析：`appendMap` 用 `strings.Split(line, " ")` 按单个空格切，直接读 `values[0]`、`values[1]`、`values[2]`，第四列可以没有（`len(values) > 3` 才取，否则方法名是空串）。
- 校验：`Config.validate` 在解析出来的路由表为空时拒绝加载，文本是 `urlmapping is empty, so no route would be served at all (a configuration file read while it is being written looks like this)`。

> **版本**：v0.6.2 起，空 `urlmapping` 会被拒绝。此前空文件是合法 YAML，会加载成一张没有路由的表，每个请求都 404 而进程看起来正常。

- 是否参与热重载：是。

### interceptor / preprocessor / postprocessor / hijacker

作用：把组件按 HTTP 方法和路径模式挂到请求处理流程上。

- 类型：字符串列表，每行 `METHOD /path ComponentName`。`interceptor` 的第三列可以用逗号列多个名字（`A,B`，不能带空格），`loadComponents` 会对它做 `strings.Split(m[2], ",")`；`preprocessor` / `postprocessor` / `hijacker` 则把第三列整串当作一个组件名，所以只能写一个名字，写了逗号会找不到组件。
- 组件必须在代码里用 `Server.RegisterComponent` 注册过，名字大小写敏感。找不到时 `getComponentByName` 返回的错误是 `no such component: <name>, forget to register?`，并被 panic。
- 是否参与热重载：是。

### convertor

作用：为某个类型注册一个通过代码注册的转换函数，让 turbo 知道怎么把请求构造成这个结构体。

- 类型：字符串列表，每行两列 `TypeName ComponentName`。`loadConvertor` 读 `values[0]`、`values[1]`，只有两列。
- 组件类型必须是 `Convertor`，否则 `getComponentByName(s, m[1]).(Convertor)` 的类型断言 panic。
- 是否参与热重载：是。

### errorhandler

作用：指定一个自定义错误处理组件，替换 `defaultErrorHandler`。

- 类型：单个字符串，不是列表。`Config.ErrorHandler()` 实现是 `c.GetString("errorhandler")`。
- 组件类型必须是 `ErrorHandlerFunc`。
- 是否参与热重载：是。

### auth

作用：只给启动路由审计用。turbo 在这个段里只读两个键：

- `auth.interceptors`：被认定为鉴权拦截器的组件名列表。
- `auth.public_routes`：允许没有鉴权的路由，每条写成 `METHOD /path`。

段里其它键 turbo 不读，留给业务自己用。细节和完整示例见 `16-auth-and-route-audit.md`。

- 是否参与热重载：是。`Server.loadComponents` 每次重建都会重新调用 `Config.authConfig()`。

## 热重载 / 需要重启对照表

`watchConfigReload` 启动时会把下面这句话打进日志：

```text
turbo: a configuration change reloads urlmapping, components and filter_proto_json; http_port, grpc_service_port, thrift_service_port, environment, turbo_log_path and log_level need a restart
```

把这句话和实际代码对一遍，得到：

| 改动后自动生效 | 需要重启（或重新生成代码） |
|---|---|
| `urlmapping` | `http_port` |
| `interceptor` / `preprocessor` / `postprocessor` / `hijacker` / `convertor` / `errorhandler` | `grpc_service_host` / `grpc_service_port` |
| `filter_proto_json` 及其两个子项 | `thrift_service_host` / `thrift_service_port` |
| `json_field_names`、`auth` | `environment`、`turbo_log_path`、`log_level` |
|  | `file_root_path`、`package_path`（只影响代码生成） |
|  | `grpc_service_name`、`thrift_service_name`（只影响代码生成） |

两点补充说明，都是从代码读出来的：

- 日志行没有提 `json_field_names` 和 `auth`，但它们在重载范围内：前者在 `writeResponse` 里按当前配置读，后者在 `loadComponents` 重建时读。
- 日志行没有提 `grpc_service_host` / `thrift_service_host`，但客户端只在启动时 dial 一次，改了 host 同样不生效，所以要按“需要重启”对待。

> **版本**：v0.6.1 起，用 `SetCommonInterceptor` 装的全局拦截器能在配置重载后存活。`Server.loadComponents` 重建 `Components` 时会把 `commonInterceptors` 一起带过去。

## 一份注释齐全的最小可用示例

```yaml
config:
  # 决定日志形态：production 写文件 + JSON + info；其它值写 stderr + 文本 + debug
  environment: production

  # 代码生成用：ServiceRootPath = file_root_path + "/" + package_path
  file_root_path: /src
  package_path: github.com/vaporz/turbo/test/testservice

  # production 下写 <turbo_log_path>/turbo.log；留空表示当前工作目录
  turbo_log_path:

  # 只覆盖日志级别，不改格式和输出位置（v0.6.2 起）
  log_level: info

  # 必填，HTTP 网关监听端口
  http_port: 8085

  # 逗号分隔，不 TrimSpace，多个服务名之间不要留空格
  grpc_service_name: TestService,MinionsService
  grpc_service_host: 127.0.0.1
  grpc_service_port: 50065

  thrift_service_name: TestService
  thrift_service_host: 127.0.0.1
  thrift_service_port: 50065

  # 只有值等于字符串 "true" 才打开（true / True / TRUE 经 YAML 解析后都会变成 "true"）
  filter_proto_json: true
  filter_proto_json_emit_zerovalues: true
  filter_proto_json_int64_as_number: true

  # proto 或 camel，默认 proto
  json_field_names: proto

# 每行 METHOD /path ServiceName MethodName，第三列对应 GrpcClient/ThriftClient 的 map key
urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TestService SayHello
  - GET /hello TestService SayHello
  - GET /eat MinionsService Eat

# 每行 METHOD /path ComponentName；多个拦截器写 A,B，逗号后不要空格
interceptor:
  - GET /hello Test1Interceptor
preprocessor:
  - GET /hello preProcessor
postprocessor:
  - GET /hello postProcessor
hijacker:
  - GET /hello hijacker
convertor:
  - CommonValues convertor

# 单个组件名，不是列表
errorhandler: error_handler

# turbo 在这个段里只读这两个键
auth:
  interceptors:
    - Test1Interceptor
  public_routes:
    - GET /eat
```

## 常见写错清单

1. 把 `config.*` 写到顶层。`http_port`、`grpc_service_port`、`filter_proto_json` 都必须放在 `config:` 下面；放到顶层时 turbo 读不到，表现为 `[http_port] is required!` 之类的 panic，或者 `filter_proto_json` 静默不生效。
2. `urlmapping` 行少于三列。`appendMap` 直接下标访问 `values[2]`，行内空白字段不足三个会 panic（被 `loadServiceConfigErr` 的 recover 包成 `invalid configuration <file>: ...`）。恰好三列不 panic，但第四列方法名是空串，生成 switcher 时会落进 `No such method[...]` 分支。
3. 行内用多个空格分隔。`strings.Split(line, " ")` 不合并连续空格，`GET  /hello A B` 会切出空字段，路由的 `url` 变成空串、`ServiceName` 变成 `/hello`，行为完全错位。
4. `interceptor` 写多个名字时留了空格。`GET /hello A, B` 会先被空格切成四列，第三列是 `A,`，第四列 `B` 被丢掉；重载时会去注册表里找 `A` 和空名字，报 `no such component: , forget to register?`。
5. 组件名拼错或没注册。错误文本是 `no such component: <name>, forget to register?`，启动时 panic，热重载时拒绝该次变更。
6. 把 `errorhandler` 写成列表。它读的是 `GetString("errorhandler")`，单个字符串；YAML 列表经 `cast.ToString` 得到空串，`loadComponents` 里 `len(s.Config.ErrorHandler()) > 0` 为假，这个键被静默忽略，自定义错误处理不会生效。
7. `filter_proto_json` 用了别的“真值”。源码比较的是字符串 `"true"`；`yes`、`on`、`1` 转出来不是 `"true"`，不算打开而且是静默的。`True` / `TRUE` 反而可以，因为 YAML 先把它们解析成布尔真。
8. `grpc_service_name: A, B` 留了空格。`strings.Split` 不 TrimSpace，第二个服务名是 `" B"`，生成的 map key 与 `urlmapping` 第三列对不上。

关于键名大小写，源码行为是这样：viper 会把配置里的键统一转成小写（`insensitiviseMap`），所以 `URLMapping:`、`HTTP_PORT:` 也能读到。真正大小写敏感的是值：组件名、服务名、`auth.public_routes` 里的路径都按原样比较。

## 相关阅读

- [04-routing.md](04-routing.md)
- [05-components.md](05-components.md)
- [06-interceptor.md](06-interceptor.md)
- [14-logging.md](14-logging.md)
- [15-hot-reload.md](15-hot-reload.md)
- [16-auth-and-route-audit.md](16-auth-and-route-audit.md)
- [appendix-config-example.md](appendix-config-example.md)
- [appendix-cheatsheet.md](appendix-cheatsheet.md)
