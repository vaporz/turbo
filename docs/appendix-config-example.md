# 附录 A：一份注释齐全的 service.yaml

本文是一份**完整、有效、可复制**的 `service.yaml`，覆盖 `config` 段全部键、
五类组件声明、`urlmapping` 的多种写法，以及 `auth` 鉴权审计段。配套的最小
`components.go` 紧随其后，配置里出现的每个组件名都能在那里找到注册代码。

它不是模板：请从 [03-service-yaml.md](03-service-yaml.md) 了解每个键的推导过程，
从 [02-getting-started.md](02-getting-started.md) 了解 `turbo create` 生成的骨架长什么样。

> **版本**：本文按 v0.6.2 编写。`config.log_level` 与"审计明细降到 debug、info 只留一行汇总"是 v0.6.2 起的行为；
> `auth.interceptors` / `auth.public_routes` 的强制校验自 v0.6.0 起存在。

## 1. service.yaml

```yaml
# =============================================================================
# service.yaml（turbo v0.6.2）
# 键名大小写不敏感；config 段之外的组件段都是"URL 模式 + 组件名"的列表。
# 行内分隔符必须是**单个空格**：解析用 strings.Split(line, " ")，多空格会产生空字段。
# =============================================================================

config:
  # 运行环境。只有 "production" 会：日志改 JSON、写到 turbo_log_path/turbo.log、级别默认 info。
  # 其它任何值都走：TextFormatter + stderr + debug + ContextHook（打印 file/func/line）。
  # 本键只在 NewGrpcServer/NewThriftServer 里被读取，热重载不生效，需要重启。
  environment: production

  # 包根目录，必须是**绝对路径**。与 package_path 用 "/" 拼成 ServiceRootPath。
  # 代码生成（turbo generate、loadFieldMapping）依赖它；为空或相对路径时
  # Config.FileRootPath() 会 panic："fileRootPath MUST be an absolute path"。
  file_root_path: /opt/turbo

  # 相对 file_root_path 的包路径，也就是生成工程所在目录。为空时 Config.PackagePath() panic。
  package_path: example.com/turbo/orderservice

  # turbo 自身日志目录，仅在 environment: production 时使用。
  # 留空 -> 用进程启动时的工作目录；相对路径 -> 拼到工作目录后面；目录不存在会自动创建。
  # 文件名固定为 turbo.log。本键需要重启才生效。
  turbo_log_path: /var/log/turbo

  # turbo 自身日志级别：panic/fatal/error/warn/info/debug/trace，非法值在加载配置时报错。
  # 留空 -> 由 environment 决定（production 为 info，其它为 debug）。
  # 只改 turbo 的日志级别，不改输出目标与格式（那是 environment 决定的）。需要重启。
  log_level: info

  # 网关对外的 HTTP 监听端口，必填。缺失或为空时 Config.HTTPPort() panic："[http_port] is required!"。
  # 需要重启才生效。
  http_port: 8081

  # gRPC 上游服务名，逗号分隔（逗号后不要加空格）。名字必须与 clientCreator 返回的 map 键一致。
  # 运行时由 clientCreator 决定服务实例，本键主要供 turbo create/generate 使用。
  grpc_service_name: HealthService,SearchService,UserService,OrderService,PaymentService
  # gRPC 上游地址与端口。连接在 StartHTTPServer/Start 时建立，改动需要重启。
  grpc_service_host: 127.0.0.1
  grpc_service_port: 50061

  # thrift 上游服务名、地址与端口；只有 NewThriftServer(...) 这一组会被用到。
  thrift_service_name: HealthService,SearchService,UserService,OrderService,PaymentService
  thrift_service_host: 127.0.0.1
  thrift_service_port: 50062

  # protobuf 响应补丁总开关，缺省 false。开启后会补零值、把 int64 从字符串改成数字。
  filter_proto_json: true
  # 仅当 filter_proto_json: true 时参与判断；缺省 true。false 表示不补零值字段。
  filter_proto_json_emit_zerovalues: true
  # 仅当 filter_proto_json: true 时参与判断；缺省 true。false 表示 int64 仍输出字符串。
  filter_proto_json_int64_as_number: true

  # proto JSON 的键拼写：proto（owner_openid，缺省）或 camel（ownerOpenid）。
  # 其它值在加载时报错，不会静默回退。
  json_field_names: proto

urlmapping:
  # 格式：METHOD[,METHOD...] PATH ServiceName MethodName
  # 方法列表用逗号分隔且不能有空格；PATH 里可以带 {名字} 或 {名字:正则} 路径参数。
  - GET /health HealthService Health
  - GET,POST /search SearchService Search
  - GET /users/{userId:[0-9]+} UserService GetUser
  - POST /users UserService CreateUser
  - PUT /users/{userId:[0-9]+} UserService UpdateProfile
  - GET /orders/{orderId:[0-9a-fA-F-]+} OrderService GetOrder
  - POST /payments/notify PaymentService Notify
  - GET /profile UserService GetProfile

interceptor:
  # 格式：METHOD[,METHOD...] URL_PATTERN Name[,Name...]
  # 同一路由被多条声明命中时，只有**第一条**生效；审计会为此打一条 warn。
  # 组件实例被所有请求共享、并发调用：请求状态放 req.Context()，别放结构体字段。
  - GET,POST /search AuthInterceptor
  - GET /users/{userId:[0-9]+} AuthInterceptor
  - POST /users AuthInterceptor
  - PUT /users/{userId:[0-9]+} AuthInterceptor
  - GET /orders/{orderId:[0-9a-fA-F-]+} AuthInterceptor
  - GET /profile AuthInterceptor

preprocessor:
  # 在真正发起 RPC 之前运行；返回 error 会交给 errorhandler。
  - PUT /users/{userId:[0-9]+} ValidateProfile

postprocessor:
  # 拿到 RPC 响应之后、写回 JSON 之前运行；可以改写响应或改写 error。
  - PUT /users/{userId:[0-9]+} AuditResponse

hijacker:
  # 命中后完全接管：turbo 不再跑 preprocessor、不再调用 RPC、不跑 postprocessor、不做 JSON 序列化。
  - POST /payments/notify PaymentNotifyHijacker

convertor:
  # 格式：GoTypeName ComponentName
  # 按 Go 类型名匹配（请求消息本身，或它内部嵌套字段的类型）；命中后该值由 Convertor 直接构造。
  - GetProfileRequest ProfileRequestConvertor

# 自定义错误处理函数。注意它与上面的组件段不同：这是一个标量字符串，不是列表。
errorhandler: turboErrorHandler

auth:
  # 声明哪些拦截器算"鉴权拦截器"。一旦声明，审计会强制执行：
  # 某条路由的有效拦截器链（全局拦截器 + 路由自己声明的）里不含任何一个，
  # 且不在 public_routes 里，就拒绝启动，热重载时也拒绝这次变更。
  # 留空或整段不写时，审计只报告、不拒绝。
  interceptors:
    - AuthInterceptor
  # 允许"没有鉴权拦截器"的路由，格式 "METHOD /path"，方法名大小写不敏感、多余空格会被归一。
  # 被 hijacker 接管的路由 turbo 无法判断其安全性，按需放进这里。
  public_routes:
    - GET /health
    - POST /payments/notify
```

## 2. 配套的 components.go

```go
// 生成工程里 components.go 的位置：<package_path>/grpcapi/component/components.go
package component

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/vaporz/turbo"

	// turbo generate 产出的 proto 包；包名由 option go_package 决定。
	"example.com/turbo/orderservice/gen/proto"
)

// AuthInterceptor 是需要鉴权的路由上运行的拦截器。
// 组件是单例、被并发调用：这里不保存任何请求级字段。
type AuthInterceptor struct {
	turbo.BaseInterceptor
}

func (i *AuthInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	token := req.Header.Get("X-Auth-Token")
	if token == "" {
		// 带状态码的错误：默认 errorhandler 会用 401 回答，而不是一律 500。
		return turbo.Errorf(http.StatusUnauthorized, "missing X-Auth-Token")
	}
	// verifyToken 是你的验签逻辑，这里用前缀代替。
	userID := strings.TrimPrefix(token, "token-")
	if userID == token {
		return turbo.Errorf(http.StatusForbidden, "invalid token")
	}
	// 注入值优先级最高（injected > path > body > query/form），客户端无法覆盖。
	// InjectParam 就地改写 req，之后必须继续使用同一个 *http.Request。
	turbo.InjectParam(req, "userId", userID)
	return nil
}

// LogInterceptor 是全局拦截器：每个请求都跑，先于路由自己声明的拦截器。
type LogInterceptor struct {
	turbo.BaseInterceptor
}

func (i *LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	fmt.Println("-->", req.Method, req.URL.Path)
	return nil
}

func (i *LogInterceptor) After(resp http.ResponseWriter, req *http.Request) error {
	fmt.Println("<--", req.Method, req.URL.Path)
	return nil
}

// ValidateProfile 是 preprocessor：RPC 调用之前运行。
func ValidateProfile(resp http.ResponseWriter, req *http.Request) error {
	if req.Header.Get("Content-Type") == "" {
		return turbo.Errorf(http.StatusBadRequest, "Content-Type is required")
	}
	return nil
}

// AuditResponse 是 postprocessor：RPC 响应之后、写回 JSON 之前运行。
func AuditResponse(resp http.ResponseWriter, req *http.Request, serviceResponse interface{}, err error) error {
	if err != nil {
		return nil // 交给 errorhandler 处理 RPC 错误
	}
	fmt.Printf("rpc response: %+v\n", serviceResponse)
	return nil
}

// PaymentNotifyHijacker 完全接管 POST /payments/notify：响应由这里写出。
func PaymentNotifyHijacker(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	fmt.Fprint(resp, `{"result":"ok"}`)
}

// ProfileRequestConvertor 对应 convertor 段的 GetProfileRequest。
// 必须返回指针（*GetProfileRequest），turbo 会取 .Elem() 赋给请求结构体。
func ProfileRequestConvertor(req *http.Request) reflect.Value {
	userID, _ := turbo.InjectedValue("userId", req)
	return reflect.ValueOf(&proto.GetProfileRequest{UserId: userID})
}

// turboErrorHandler 是 errorhandler 段指向的函数。
func turboErrorHandler(resp http.ResponseWriter, req *http.Request, err error) {
	status := turbo.StatusOf(err)
	if status == 0 {
		status = http.StatusInternalServerError
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(status)
	fmt.Fprintf(resp, `{"error":%q}`, err.Error())
}

// ServiceInitializer 在服务启动前把所有组件按名字注册进 turbo。
type ServiceInitializer struct{}

// InitService 由 turbo 在 Start/StartHTTPServer/StartGrpcService 里调用，
// 早于配置里的组件被解析，所以这里注册的名字才能被 service.yaml 找到。
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	// service.yaml 里出现过的每个组件名都必须在这里注册，
	// 否则 loadComponents 会 panic："no such component: xxx, forget to register?"
	log := &LogInterceptor{}
	s.RegisterComponent("LogInterceptor", log)
	s.RegisterComponent("AuthInterceptor", &AuthInterceptor{})

	// 函数类型的组件必须转换成 turbo 的具名类型再注册，
	// 否则 loadComponents 里的类型断言会失败。
	s.RegisterComponent("ValidateProfile", turbo.Preprocessor(ValidateProfile))
	s.RegisterComponent("AuditResponse", turbo.Postprocessor(AuditResponse))
	s.RegisterComponent("PaymentNotifyHijacker", turbo.Hijacker(PaymentNotifyHijacker))
	s.RegisterComponent("ProfileRequestConvertor", turbo.Convertor(ProfileRequestConvertor))
	s.RegisterComponent("turboErrorHandler", turbo.ErrorHandlerFunc(turboErrorHandler))

	// 全局拦截器来自代码而不是配置：每个请求上先跑，路由级拦截器跟在后面。
	// 必须在服务器启动前安装，并且注册成同一个实例，审计才能把全局拦截器对上名字。
	// 配置热重载会保留它（v0.6.1 起）。
	s.ServerField().Components.SetCommonInterceptor(log)
	return nil
}

// StopService 由 turbo 在所有服务器停止之后调用。
func (i *ServiceInitializer) StopService(s turbo.Servable) {}
```

## 3. 这份配置启动后应该看到什么

`environment: production` 时 turbo 用 logrus 的 JSONFormatter，把日志写进
`<turbo_log_path>/turbo.log`（这里是 `/var/log/turbo/turbo.log`），级别为 info，
每条是一个 JSON 对象，形如：

```json
{"level":"info","msg":"Starting Turbo...","time":"2026-09-23T10:00:00+08:00"}
```

下面是各条 `msg` 的真实文本，按出现顺序列出（生产环境每条都包在上述 JSON 里；
development 环境是 TextFormatter，还会带 `file=`、`func=`、`line=` 字段）：

```text
Starting Turbo...
Starting GRPC Service...
GRPC Service started
Starting HTTP Server...
[grpc]connecting addr:127.0.0.1:50061
interceptor:[GET,POST /search AuthInterceptor ]
interceptor:[GET /users/{userId:[0-9]+} AuthInterceptor ]
interceptor:[POST /users AuthInterceptor ]
interceptor:[PUT /users/{userId:[0-9]+} AuthInterceptor ]
interceptor:[GET /orders/{orderId:[0-9a-fA-F-]+} AuthInterceptor ]
interceptor:[GET /profile AuthInterceptor ]
preprocessor:[PUT /users/{userId:[0-9]+} ValidateProfile ]
postprocessor:[PUT /users/{userId:[0-9]+} AuditResponse ]
hijacker:[POST /payments/notify PaymentNotifyHijacker ]
convertor:[GetProfileRequest ProfileRequestConvertor  ]
errorhandler:turboErrorHandler
route audit: 9 route(s), 7 authenticated, 2 public, 0 unprotected
route: GET /health -> HealthService.Health
route: GET,POST /search -> SearchService.Search
route: GET /users/{userId:[0-9]+} -> UserService.GetUser
route: POST /users -> UserService.CreateUser
route: PUT /users/{userId:[0-9]+} -> UserService.UpdateProfile
route: GET /orders/{orderId:[0-9a-fA-F-]+} -> OrderService.GetOrder
route: POST /payments/notify -> PaymentService.Notify
route: GET /profile -> UserService.GetProfile
turbo: 8 route(s) registered
HTTP Server started
turbo: a configuration change reloads urlmapping, components and filter_proto_json; http_port, grpc_service_port, thrift_service_port, environment, turbo_log_path and log_level need a restart
```

组件那几行是把配置解析出的 `[4]string` 整段打印出来，末尾空字段会留下空格（convertor 行有两个），这是正常的。

关于审计那一行（`audit.go` 的真实格式）：

- `9 route(s)` 是按"HTTP 方法 × urlmapping 行"展开后的条数：`GET,POST /search` 算 2 条，其余 7 行各算 1 条。
- `7 authenticated` 是有效拦截器链里含 `AuthInterceptor` 的条数。
- `2 public` 是 `auth.public_routes` 命中的条数（`GET /health`、`POST /payments/notify`）。
- `0 unprotected` 表示审计没有拒绝这份配置。
- 改成 `development`（或 `config.log_level: debug`）后还会多出逐条明细，例如 `route audit: GET /health -> HealthService.Health [public, auth=common:[LogInterceptor] + route:[]]` 和 `[auth=common:[LogInterceptor] + route:[AuthInterceptor], authenticated]`；info 级别下没有这些行。

## 4. 三个最容易踩的点

1. **`environment: production` 会同时改格式、目标和级别**：JSON、写文件、info；只想在终端看 debug 就保持 `development`。
2. **`file_root_path` 必须是绝对路径**：它和 `package_path` 拼成代码生成的根目录，相对路径会让 `Config.FileRootPath()` panic。
3. **配置里的组件名必须在 `InitService` 注册**：漏一个就是启动时的 `no such component` panic；函数类型还要转成 `turbo.Preprocessor` 等具名类型。

组件各自的语义见 [05-components.md](05-components.md)，鉴权审计的完整规则见
[16-auth-and-route-audit.md](16-auth-and-route-audit.md)，热重载范围见 [15-hot-reload.md](15-hot-reload.md)。

## 相关阅读

- [03-service-yaml.md](03-service-yaml.md) —— 每个配置键的推导与失败表现
- [05-components.md](05-components.md) —— 组件签名、注册与执行顺序
- [06-interceptor.md](06-interceptor.md) —— 上面那个鉴权拦截器的完整写法与坑
- [16-auth-and-route-audit.md](16-auth-and-route-audit.md) —— `auth.interceptors` / `auth.public_routes` 的规则
- [15-hot-reload.md](15-hot-reload.md) —— 这份配置里哪些改动不需要重启
