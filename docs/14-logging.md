# turbo 自己的日志

这篇讲 turbo 框架自己打出的日志：谁决定格式、输出和目标级别，怎么在测试里接管它，怎么控制字段顺序，以及启动、路由、审计、重载、绑定冲突这几类日志行分别是什么意思。

> **版本**：v0.6.2 起，`config.log_level` 可以单独指定 turbo 自己的日志级别；启动时会逐条打印路由表（`route: ...`）和一行路由审计汇总（`route audit: ...`）；没有匹配到路由的请求会打印 `404 no route`。

## 谁决定格式、输出和默认级别

turbo 没有自己的日志实现，它用的是 logrus 的 standard logger（`log.go`）：

```go
var log *logger.Logger
// ...
log = logger.StandardLogger()
```

这意味着 `log` 和 logrus 的包级函数指向同一个 logger，你在服务里用 `logrus.SetFormatter` 改的就是 turbo 在用的那个。

真正的开关是 `config.environment`（`log.go` 的 `initLogger`）。它一次决定三件事：格式、输出位置、默认级别。

| `environment` | formatter | 输出 | 默认级别 | 额外 hook |
|---|---|---|---|---|
| `production` | `logger.JSONFormatter` | 文件 `<turbo_log_path>/turbo.log` | `info` | 不加 |
| 其它值或未设置 | `logger.TextFormatter` | `os.Stderr` | `debug` | `ContextHook`（补 `file` / `func` / `line` 字段） |

注意 `production` 分支不会加 `ContextHook`，所以 JSON 日志里没有 `file`、`func`、`line` 这三个字段；文本日志有。

### turbo_log_path 与文件位置

`setupLoggerFile`（`log.go`）的行为：

- 取 `config.turbo_log_path`，为空（或只有空白）时用当前工作目录 `os.Getwd()`。
- 不是绝对路径时拼成 `<工作目录>/<turbo_log_path>`，再做 `path.Clean`。
- 用 `os.MkdirAll(logPath, 0755)` 建目录，用 `os.OpenFile(logPath+"/turbo.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)` 打开文件。

所以文件名固定是 `turbo.log`，多个实例写到同一个目录时会追加进同一个文件。另外源码里的 TODO 写得很清楚：`os.Getwd()` 返回的是执行文件时所在的目录，不是可执行文件所在的目录。用 systemd 或容器部署时一定要写绝对路径，否则日志会落在进程的工作目录里。

还有一个细节：判空用的是 `strings.TrimSpace`，但真正拼路径时用的是原值，所以 `turbo_log_path` 前后的空格不会被清掉，会变成路径的一部分。

## config.log_level：只挑级别

`config.log_level` 让服务在不改变格式和输出的前提下换一个级别。它读的是 `config` 段下面那个键（`config.go` 的 `LogLevel()`，注释里也说明它和“服务自己顶层 `log_level`”是两回事，服务那份归服务自己解释）。

```yaml
config:
  environment: production
  turbo_log_path: /var/log/yourservice
  log_level: warn
```

- 取值：`panic`、`fatal`、`error`、`warn`、`info`、`debug`、`trace`。校验失败时的报错原文是 `invalid log_level: %q, expected one of panic, fatal, error, warn, info, debug, trace`。
- 由于底层用的是 logrus 的 `ParseLevel`，同义写法 `warning` 也能通过，值前后的空白会被 `LogLevel()` 的 `TrimSpace` 去掉（`log_level_test.go` 里就有 `"  Info "` 这样的用例）。
- 缺省：不写这个键时，级别完全由 `environment` 决定（`production` 是 `info`，其它是 `debug`）。
- 只改级别：`initLogger` 只在 `environment` 分支里设置 formatter 和输出，`config.log_level` 只会走到 `logger.SetLevel`，格式和输出不动。
- 非法值会被拒绝：启动时 `NewGrpcServer` / `NewThriftServer` 里 `loadServiceConfig()` 会 panic；运行中改成一个非法值，重载会失败并保留正在生效的配置。
- 不参与热重载：`watchConfigReload` 启动时打印的那行明确把 `log_level` 归到“需要重启”的一组。改完要重启进程。

## 在测试里接管输出：turbo.SetOutput

`log.go` 里用来在运行时接管输出的公开函数是 `SetOutput`：

```go
// SetOutput sets output at runtime
func SetOutput(out io.Writer) {
	log.Out = out
	log.Formatter = &logger.TextFormatter{}
}
```

两点必须记住：

1. 它改的是全局 standard logger，所以影响进程里所有使用 logrus 的代码。
2. 它顺手把 formatter 换成默认 `TextFormatter`。即使你的服务在 `production` 下用 JSON，调用之后输出也是文本。

测试里的标准用法，来自 `test/integration_test.go`：

```go
logged := &syncBuffer{}
turbo.SetOutput(logged)
defer turbo.SetOutput(os.Stdout)
```

`syncBuffer` 是那个测试文件里自己写的、带 `sync.Mutex` 的 `bytes.Buffer`，因为请求处理器在自己的 goroutine 里写日志。你的测试里也要用带锁的 buffer，或者让断言只发生在所有 goroutine 结束之后。

如果你想断言启动时打印的路由表，必须在 `StartHTTPServer`（或 `Start`）之前就调用 `SetOutput`，因为路由表是在那一步打印的。

## 字段排序：SortingFunc、SortKeysFirst 和 CheckSortingFunc

`log_sorting.go` 解决的是 logrus 的一个坑：**只有 `TextFormatter` 有 `SortingFunc`**。`JSONFormatter` 序列化的是 map，字段顺序由 `encoding/json` 决定，也不会丢字段。

`TextFormatter` 只会写 `SortingFunc` 留在 `[]string` 里的名字，所以一个“重写名字”而不是“重排名字”的函数会把它覆盖掉的字段删掉。源码注释给出的两个例子：

```text
// 函数收到的名字
amount device_code order_id user_id
// keys[0] = "level"; keys[1] = "msg"   —— 重写
level msg order_id user_id amount         // device_code 没了
// keys[i] = "" for i >= 2              —— 截断
amount device_code ="<nil>" ="<nil>"      // 字段悄悄变成 nil
```

`turbo.SortKeysFirst` 只会重排：先按你点名的顺序，其余按字母序，结果永远是输入的一个排列，不可能丢字段或改名。

给一个真实的 logrus formatter 配置示例。注意要在 `NewGrpcServer` / `NewThriftServer` 之后再设置，因为 `initLogger` 会覆盖 formatter；`turbo.SetOutput` 也会覆盖它。

```go
import (
	logger "github.com/sirupsen/logrus"
	"github.com/vaporz/turbo"
)

func setupLogFormat() {
	logger.SetFormatter(&logger.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
		SortingFunc: turbo.SortKeysFirst(
			"level", "msg", "time", "file", "func", "line",
		),
	})
}
```

`turbo.CheckSortingFunc` 是给你在自己的测试里检查的，它不是包装器（包装器救不回来：logrus 读的是它传进去的那个切片，被删掉的名字在包装器里补不回来）：

```go
func TestLogFieldOrder(t *testing.T) {
	mySorter := func(keys []string) { /* 你自己写的排序 */ }
	if err := turbo.CheckSortingFunc(mySorter, "level", "msg", "user_id", "amount"); err != nil {
		t.Fatal(err)
	}
}
```

它报告的是“这个函数是不是输入名字的一个排列”。传 `nil` 直接通过，因为 logrus 会自己按字母序排。报错文本以 `turbo: this SortingFunc is not a permutation of the names it was given` 开头，并提示改用 `turbo.SortKeysFirst`。

## 读懂 turbo 的日志行

下面每一条都来自源码，级别按 logrus 的调用方式标注。

### 启动与停止

| 级别 | 日志文本 | 出处 |
|---|---|---|
| Info | `Starting Turbo...` | `GrpcServer.Start` / `ThriftServer.Start` |
| Info | `Starting GRPC Service...` | `startGrpcServiceInternal` |
| Info | `GRPC Service started` | `startGrpcServiceInternal` |
| Info | `Starting Thrift Service at :%s...` | `startThriftServiceInternal` |
| Info | `Thrift Service started` | `startThriftServiceInternal` |
| Info | `Starting HTTP Server...` | `startGrpcHTTPServerInternal` / `startThriftHTTPServerInternal` |
| Info | `HTTP Server started` | `startHTTPServer` |
| Info | `Stop() invoked, Service is stopping...` | `GrpcServer.Stop`（Thrift 的 `Stop` 不打印这行） |
| Info | `Http Server stopped` | `stop` |
| Info | `Grpc Server stopped` | `stop` |
| Info | `Thrift Server stopped` | `stop` |
| Info | `HTTP Server failed to serve: %v` | `startHTTPServer`（`log.Printf`，Info 级） |
| Info | `GRPC Service failed to serve: %v` | `startGrpcServiceInternal`（`log.Printf`，Info 级） |
| Panic | `grpc connection not initiated!` | `GrpcServer.Service`（还没启动 gRPC 客户端时取服务实例） |
| Panic | `thrift connection not initiated!` | `ThriftServer.Service` |

### 配置装载

`loadComponents` 每配一个组件打一行 Info：

- `interceptor:` 后面跟该行映射的 `[4]string`
- `preprocessor:` / `postprocessor:` / `hijacker:` / `convertor:` 同理
- `errorhandler:` 后面跟 `config.ErrorHandler()`

`watchConfigReload` 在启动时打印一行 Info，提醒哪些键重载、哪些键要重启：

```text
turbo: a configuration change reloads urlmapping, components and filter_proto_json; http_port, grpc_service_port, thrift_service_port, environment, turbo_log_path and log_level need a restart
```

### 路由表与 404

`router`（`runtime.go`）在每次装载路由时逐条打印：

```text
route: GET /hello -> TestService.SayHello
```

最后一行是汇总：

```text
turbo: 2 route(s) registered
```

没有匹配到路由的请求由 `notFoundHandler` 记录，级别是 Error（HTTP 响应仍然是标准的 `404 page not found`）：

```text
turbo: 404 no route for GET /no/such/path, host=localhost:8080, remote=127.0.0.1:53124, user-agent="curl/8.5.0"
```

query 被故意排除在外，因为它可能带 token 或签名；`test/integration_test.go` 有断言确认 `token=secret` 不会出现在日志里。

### 路由审计

`auditRoutes`（`audit.go`）每次装载配置都会跑。逐条明细是 Debug，一行汇总是 Info：

```text
route audit: GET /hello -> TestService.SayHello [auth=common:[] + route:[TestInterceptor], authenticated]
route audit: GET /healthz -> TestService.SayHello [public, auth=common:[] + route:[]]
route audit: 2 route(s), 1 authenticated, 1 public, 0 unprotected
```

当配置里没有声明 `auth.interceptors` 时，汇总会补一句 `(no auth.interceptors declared, so the audit only reports)`，表示审计只报告不拦截。异常情况还有三种：

- Warn：一条路由被多条 interceptor 声明匹配，`route audit: %s %s is matched by %d interceptor declarations %v, only the first one runs`（只有第一条生效）。
- Warn：`route audit: ... [auth=..., cannot be verified: a common interceptor is not registered under a name]`（公共拦截器是匿名注册的，没法核对）。
- Error：`route audit: ... [UNPROTECTED: ...]`，随后汇总里的 `unprotected` 计数非零，最终装配置时报错 `refusing this configuration: ...`。

### 热重载

| 级别 | 日志文本 | 含义 |
|---|---|---|
| Info | `Reloading configuration...` | 收到变更并开始重载 |
| Info | `Configuration reloaded` | 重载成功，新路由表已生效 |
| Error | `turbo: configuration reload failed, keeping the running configuration: <err>` | 变更能读到，但装不出来（例如组件没注册），保留旧配置继续服务 |
| Error | `turbo: ignoring configuration change, it cannot be loaded: <err>` | 连读都读不了（YAML 坏了、空文件、`urlmapping` 为空），直接忽略这次变更 |

### 绑定冲突

`binding.go` 在注入值覆盖了别的来源时打 Warn：

```text
binding: YourName <- injected "from-server", overriding the query/form value "from-query"
binding: YourName <- injected "from-server", overriding the request body value
```

### 其它

| 级别 | 日志文本 | 出处 |
|---|---|---|
| Error | `error in Before(): <err>` | `doBefore` |
| Error | `turbo: error in After(): <err>` | `doAfter` |
| Error | `<err>`（原始错误文本） | `defaultErrorHandler`，自定义 error handler 会取代它 |
| Info | `value is invalid, please check grpc-fieldmapping` | `BuildStructErr` |
| Info | `turbo: generating with %s` | `checkToolchain`（代码生成路径） |
| Warn | `turbo: protoc-gen-go reports %s. ...` | `checkToolchain`，装的是新版插件时提醒 |

## 该把哪些日志放在什么级别

结合上面的分布，可以这样安排：

- `info`（只留汇总）：保留启动/停止、组件装载、路由表、`route audit` 汇总、`Reloading configuration...` / `Configuration reloaded`。生产环境默认就是这个级别，`test/integration_test.go` 的 `TestConfiguredLogLevelIsUsed` 也确认 `log_level: warn` 时连路由审计汇总都不写。
- `debug`（排查单条路由）：打开后能看到每条路由的链（`route audit: ... [auth=...]`）。一条路由一行，路由多的时候很吵，所以默认放在 Debug。
- `warn`：注入值覆盖查询参数或请求体这类“配置或调用方式可疑，但服务还在正常工作”的情况。
- `error`：404、重载失败、绑定失败、拦截器报错、业务错误。这些是“需要有人看一眼”的行。

只保留汇总：

```yaml
config:
  environment: production
  log_level: info
```

彻底安静：`config.log_level` 的最高一级是 `panic`，而 turbo 自己会用 `log.Panic` 输出 `grpc connection not initiated!` 这类行，所以配置层面做不到完全无声。要真正静默，在代码里把输出丢掉：

```go
turbo.SetOutput(io.Discard)
```

代价是 formatter 会被换成默认 `TextFormatter`，所以这只适合测试或诊断场景。

## 相关阅读

- [03-service-yaml.md](03-service-yaml.md)：`config` 段每个键的含义
- [16-auth-and-route-audit.md](16-auth-and-route-audit.md)：`auth.interceptors` / `auth.public_routes` 与审计的关系
- [15-hot-reload.md](15-hot-reload.md)：哪些键重载、哪些键要重启
- [19-troubleshooting.md](19-troubleshooting.md)：按症状查日志
- [appendix-config-example.md](appendix-config-example.md)：完整配置示例
