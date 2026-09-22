# 错误与 status code

这篇讲一个失败如何变成 HTTP 响应：错误怎么带上状态码，turbo 自己产生的错误分别带哪个码，默认 errorhandler 长什么样，怎么换成 JSON 错误响应，以及为什么返回给调用方（也写进日志）的消息里不会出现请求内容。

## 三个错误 API

源码在 `errors.go`。

```go
func Errorf(status int, format string, args ...interface{}) error
func WithStatus(err error, status int) error
func StatusOf(err error) int
```

`turbo.Errorf` 返回一个既带消息、又带状态码的错误，`Error()` 是格式化后的消息，状态码交给 `turbo.StatusOf` 读：

```go
return turbo.Errorf(http.StatusForbidden, "user %d may not do that", id)
```

内部的 `*statusError` 只有 `Error()` 和 `Unwrap()`，所以它能直接塞进 `fmt.Errorf("...: %w", err)` 的包装链。

`turbo.WithStatus` 给普通错误补上状态码，三个边界行为来自源码：`err == nil` 时返回 `nil`；错误已经带状态码时原样返回，不覆盖作者的判断；错误消息不变。

`turbo.StatusOf` 用 `errors.As` 穿透包装读状态码，**没有状态码时返回 0**，`err == nil` 时也返回 0。常见结果：

- `StatusOf(Errorf(403, "no"))` 是 `403`
- `StatusOf(errors.New("boom"))` 是 `0`
- `StatusOf(WithStatus(errors.New("boom"), 400))` 是 `400`
- `StatusOf(WithStatus(Errorf(404, "gone"), 400))` 是 `404`，已有的码优先
- `StatusOf(nil)` 是 `0`，`WithStatus(nil, 400)` 是 `nil`

## turbo 自己产生的错误带什么状态码

### 绑定失败按来源给码

`bindingErrorFor`（`binding.go`）给「参数存在但没法用」定码，规则是看这个值是谁给的：

| 值的来源 | 状态码 | 消息里的来源字样 |
| --- | --- | --- |
| `turbo.InjectParam` 注入的值 | 500 | `injected value` |
| path 参数 | 400 | `path parameter` |
| query / form 参数 | 400 | `query/form parameter` |

服务端自己注入的值绑不上是服务端的错，所以是 500；其余是调用方给的，所以是 400。消息形状固定为 `turbo: cannot bind <字段名> from <来源>: <原因>`。

### body 无法解析：400

JSON body 走 `BuildRequest`（`runtime.go`），`jsonpb` 解析失败时带 `http.StatusBadRequest`，文案是：

```
turbo: failed to BuildRequest for json api, request body: <字节数> bytes, error: <解析原因>
```

Thrift 一侧由 `bindThriftArgsFromJSON` 负责，body 不是 JSON 对象、命名了不存在的参数、或某个参数读不出来，都是 400，文案见 11-binding.md。

### 404：没有匹配到任何路由

`router()` 给 mux 挂了 `notFoundHandler()`（`runtime.go`）。响应仍然是 `http.NotFound`，但会先写一条日志，并且**故意不记 query**（query 里可能有 token 或签名）：

```
turbo: 404 no route for GET /no/such/path, host=..., remote=..., user-agent="..."
```

任何没有带上状态码的错误，最终都由默认 errorhandler 变成 500。

## 默认 errorhandler

没有配置 `errorhandler` 时，`Components.errorHandlerFunc()`（`component.go`）返回 `defaultErrorHandler`（`errors.go`），它做三件事：`log.Error(err.Error())`、把 `StatusOf(err)` 为 0 的情况改成 `http.StatusInternalServerError`、用 `http.Error(resp, err.Error(), status)` 回复。

所以响应体就是 `err.Error()` 加一个换行，`Content-Type` 是 `text/plain; charset=utf-8`，状态码是错误自带的那个。消息会同时出现在响应体和日志里。

## 自定义 errorhandler

`errorhandler` 是 `service.yaml` 的**顶层**键（不在 `config:` 下面），值是注册好的组件名：

```yaml
urlmapping:
  - GET /hello OrderService GetOrder

errorhandler: JSONErrorHandler
```

加载时 `server.go` 的 `loadComponents()` 会做 `c.WithErrorHandler(getComponentByName(s, s.Config.ErrorHandler()).(ErrorHandlerFunc))`。组件必须在启动前用 `RegisterComponent` 注册过，否则 `getComponentByName` 会 panic（`no such component: <名字>, forget to register?`）：启动时报错会让进程退出，热重载时报错会保留正在运行的配置。

`ErrorHandlerFunc`（`component.go`）的签名是 `func(http.ResponseWriter, *http.Request, error)`。一个真实可用的 JSON 错误响应：

```go
package component

import (
	"encoding/json"
	"net/http"

	"github.com/vaporz/turbo"
)

// JSONErrorHandler 把错误渲染成 {"code":...,"msg":...}，优先使用出错者声明的状态码。
func JSONErrorHandler(resp http.ResponseWriter, req *http.Request, err error) {
	status := turbo.StatusOf(err)
	if status == 0 {
		status = http.StatusInternalServerError
	}
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp.WriteHeader(status)
	_ = json.NewEncoder(resp).Encode(map[string]interface{}{"code": status, "msg": err.Error()})
}

type ServiceInitializer struct{}

func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	s.RegisterComponent("JSONErrorHandler", turbo.ErrorHandlerFunc(JSONErrorHandler))
	return nil
}

func (i *ServiceInitializer) StopService(s turbo.Servable) {}
```

也可以不写配置，直接 `s.Components.WithErrorHandler(turbo.ErrorHandlerFunc(JSONErrorHandler))`。

### 普通 handler 返回 error 时走它

`doRequest`（`runtime.go`）里，三类失败交给同一个 errorhandler：preprocessor 返回的 error、`switcherFunc` 返回的 error（也就是 gRPC/Thrift 服务实现的错误）、postprocessor 返回的 error。`Before` 拦截器返回 error 时 `handler` 也会调用它。preprocessor/postprocessor 的错误会包一层，但用的是 `%w`，状态码仍然穿得过去：

```
turbo: encounter error in preprocessor for /hello?your_name=x, error: teapot
```

这一层包装用 `req.URL`，也就是**含 query**。绑定错误不会回显参数值，但这一句会带上整条 URL；query 里有 token 时请让 preprocessor 自己给出不含敏感信息的文案。

还要注意：`turbo.Errorf` 的状态码属于 HTTP 层。在 gRPC/Thrift 服务实现里返回它，错误经过 RPC 序列化后只剩文本，HTTP 层读回的是 `StatusOf(...) == 0`，最终仍是 500。要区分 401/403 这类语义，请放在拦截器、preprocessor 或 hijacker 里。

## 错误消息里不会出现请求内容

> **版本**：v0.6.2 起，绑定失败不再回显参数值，解析错误里被引用的数字也被抹掉。

原因很直接：这条消息既返回给调用方，也写进服务日志。调用方本来就知道自己发了什么，需要留下的是**原因**；参数值可能是 token、验证码或签名，落进日志就是泄漏。

`binding.go` 里的 `redactedValue` 是 `<redacted>`，`withoutRequestValues` 负责替换：

- 参数值：`strconv` 会把值加引号写进错误，所以 `parsing "abc": invalid syntax` 变成 `parsing "<redacted>": invalid syntax`。
- 数字：`jsonNumber` 正则把 `number 1.5` 这类片段换成 `number <redacted>`，覆盖 `cannot unmarshal number 1.5 into Go value of type int64` 这种文案。
- body 无法解析时压根不记内容，只记字节数：`request body: 6 bytes`。

原因完整保留。实际响应：

```bash
$ curl -s 'http://127.0.0.1:8081/hello?int64_value=abc'
turbo: cannot bind Int64Value from query/form parameter: strconv.ParseInt: parsing "<redacted>": invalid syntax

$ curl -s -X POST 'http://127.0.0.1:8081/hello' -H 'Content-Type: application/json' -d '{aaaaa'
turbo: failed to BuildRequest for json api, request body: 6 bytes, error: invalid character 'a' looking for beginning of object key string
```

## 客户端如何区分 400 / 401 / 403 / 404 / 500

turbo 主动产生的只有 400、404、500；401 和 403 由你的组件决定。

| 码 | turbo 主动产生的场景 |
| --- | --- |
| 400 | 参数绑定失败且值来自 query/form 或 path；JSON body 解析失败；Thrift JSON body 不是对象或命名了不存在的参数 |
| 401 | 无。用 `turbo.Errorf(http.StatusUnauthorized, ...)` 或 `turbo.WithStatus` 在拦截器里给出 |
| 403 | 无。同上 |
| 404 | 路由不匹配（`notFoundHandler`）；`http.Server.Handler` 拿到的 router 为 nil 时 `http.NotFound` |
| 500 | 默认 errorhandler 遇到 `StatusOf(err) == 0`；注入值绑定失败；preprocessor/postprocessor 返回的普通 error |

补充两点：路由存在但 HTTP 方法不匹配时，405 由 mux 返回，turbo 没有改动；`writeResponse` 里响应转 JSON 失败时只写一段 `turbo: encounter error while converting response to json ...` 文本，没有显式设置状态码，所以是 200。

## 一个能跑的完整例子

拦截器负责认证并注入服务端验证过的值，errorhandler 负责把结果渲染成 JSON：

```go
package component

import (
	"encoding/json"
	"net/http"

	"github.com/vaporz/turbo"
)

type AuthInterceptor struct {
	turbo.BaseInterceptor
}

func (a *AuthInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	switch req.Header.Get("X-Token") {
	case "":
		return turbo.Errorf(http.StatusUnauthorized, "missing X-Token")
	case "let-me-in":
		turbo.InjectParam(req, "user_id", "42")
		return nil
	default:
		return turbo.Errorf(http.StatusForbidden, "token rejected")
	}
}

func JSONErrorHandler(resp http.ResponseWriter, req *http.Request, err error) {
	status := turbo.StatusOf(err)
	if status == 0 {
		status = http.StatusInternalServerError
	}
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp.WriteHeader(status)
	_ = json.NewEncoder(resp).Encode(map[string]interface{}{"code": status, "msg": err.Error()})
}

type ServiceInitializer struct{}

func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	s.RegisterComponent("JSONErrorHandler", turbo.ErrorHandlerFunc(JSONErrorHandler))
	s.RegisterComponent("AuthInterceptor", &AuthInterceptor{})
	s.Components.SetCommonInterceptor(&AuthInterceptor{})
	return nil
}

func (i *ServiceInitializer) StopService(s turbo.Servable) {}
```

`service.yaml` 里补上顶层 `errorhandler`：

```yaml
config:
  environment: development
  file_root_path: /tmp/turbodemo/src
  package_path: github.com/example/greeterservice
  http_port: 8081
  grpc_service_name: GreeterService
  grpc_service_host: 127.0.0.1
  grpc_service_port: 50061
  thrift_service_name: GreeterService
  thrift_service_host: 127.0.0.1
  thrift_service_port: 50062

urlmapping:
  - GET /hello GreeterService SayHello

errorhandler: JSONErrorHandler
```

用 `curl -i` 观察状态码：

```bash
# 401：缺少 X-Token
curl -i http://127.0.0.1:8081/hello

# 403：token 不对
curl -i -H 'X-Token: nope' http://127.0.0.1:8081/hello

# 200：通过认证，user_id 由服务端注入
curl -i -H 'X-Token: let-me-in' 'http://127.0.0.1:8081/hello'

# 400：参数存在但绑不上（消息里不会出现 abc）
curl -i -H 'X-Token: let-me-in' 'http://127.0.0.1:8081/hello?int64_value=abc'

# 404：路由不匹配
curl -i http://127.0.0.1:8081/no/such/path
```

## 相关阅读

- [参数绑定](11-binding.md)：哪些失败算 400，哪些算 500
- [拦截器](06-interceptor.md)：在 Before/After 里返回带状态码的错误
- [前置与后置处理器](07-preprocessor-postprocessor.md)：preprocessor/postprocessor 的错误如何包装
- [日志](14-logging.md)：错误消息落到哪里
- [故障排查](19-troubleshooting.md)：状态码与预期不符时怎么定位
