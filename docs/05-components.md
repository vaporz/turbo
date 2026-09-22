# 组件总览与请求生命周期

这篇讲什么：turbo 用五类组件加一个 errorhandler 来定制请求的处理过程。这一篇逐个列出它们的真实签名与职责，给出 `handler`、`doRequest` 的调用顺序，最后用一个可运行的例子把组件接起来。

## 五类组件与 errorhandler

所有组件类型都定义在 `component.go`。

```go
type Interceptor interface {
	Before(http.ResponseWriter, *http.Request) error
	After(http.ResponseWriter, *http.Request) error
}

type Preprocessor func(http.ResponseWriter, *http.Request) error

type Postprocessor func(http.ResponseWriter, *http.Request, interface{}, error) error

type Hijacker func(http.ResponseWriter, *http.Request)

type Convertor func(r *http.Request) reflect.Value

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request, error)
```

职责分别是：

- `Interceptor` 围绕一个请求运行：`Before` 在请求构造之前，`After` 在响应写出之后。它是唯一以接口形式定义的组件，要用结构体实现。
- `Preprocessor` 在请求被构造成 RPC 请求之前执行，拿到原始 `*http.Request`，可以改写或拒绝请求。
- `Postprocessor` 在 RPC 调用返回之后执行，第三个参数是 RPC 响应对象，第四个参数是 RPC 的错误。
- `Hijacker` 没有任何返回值：自己写响应、自己决定状态码，命中后 RPC 不再被调用。
- `Convertor` 告诉 turbo 某个结构体类型的值怎么从请求构造出来，按类型名注册。
- `ErrorHandlerFunc` 通过配置里的 `errorhandler: <组件名>` 指定，默认实现是 `defaultErrorHandler`（`errors.go`），它取 `StatusOf(err)` 作为状态码，取不到就用 500。

## BaseInterceptor

只有 Interceptor 有 base 类型，`component.go` 里写着：

```go
// BaseInterceptor implements an empty Before() and After()
type BaseInterceptor struct{}

func (i *BaseInterceptor) Before(resp http.ResponseWriter, req *http.Request) error { return nil }
func (i *BaseInterceptor) After(resp http.ResponseWriter, req *http.Request) error  { return nil }
```

Preprocessor、Postprocessor、Hijacker、Convertor 都是函数类型，没有对应的 base 类型：函数字面量本身就能定义完整行为。写拦截器时内嵌 `turbo.BaseInterceptor`，就只需覆盖关心的那个方法。

## 注册：一个名字一个实例，并发调用

组件通过 `Servable` 接口注册，`server.go` 的实现把组件存进 `Components.registeredComponents`，一个名字一个实例：

```go
type Servable interface {
	Service(serviceName string) interface{}
	ServerField() *Server
	Stop()
	RegisterComponent(name string, component interface{})
}
```

这个实例被所有请求共享，并且会被并发调用。`component.go` 中 `Interceptor` 的注释把后果写得很直接：在 `Before` 里写字段、在 `After` 里读字段，读到的是碰巧重叠的那些请求的值，表现为偶发、无法复现的串值。每个请求的状态要放进 `req.Context()`，或用 `turbo.InjectParam` 注入。

配置文件里写的是名字，框架按名字取出组件并做类型断言，例如 `server.go` 的 `loadComponents`：

```go
for _, m := range s.Config.mappings[preprocessors] {
	c.SetPreprocessor(strings.Split(m[0], ","), m[1], getComponentByName(s, m[2]).(Preprocessor))
	log.Info("preprocessor:", m)
}
```

名字没注册过时 `getComponentByName` 会 panic，`Component` 给出的错误是 `no such component: <name>, forget to register?`。

## SetCommonInterceptor：整体赋值，排在路由级之前

`SetCommonInterceptor(interceptors ...Interceptor)` 是整体赋值而不是追加：`setCommonInterceptor` 直接替换 `c.commonInterceptors`，第二次调用会覆盖第一次。

> **版本**：v0.6.1 起，全局拦截器与路由级拦截器组成一条链，全局的排在前面；同一个版本起，热重载会保留全局拦截器。在此之前，路由自己声明了拦截器就会替换掉全局的那批。

链的拼接在 `runtime.go` 的 `getInterceptors` 里：先取 `CommonInterceptors()`，再取 `Interceptors(req)`，两边都不为空时新建一个切片把全局的放在前面、路由级的接在后面（不复用任何一方持有的切片）。

`SetCommonInterceptor` 的注释要求它在服务启动前调用。`loadComponents` 在启动和每次重载时都会重建 `Components`，它显式把 `registeredComponents` 和 `commonInterceptors` 带过去，这是全局拦截器能活过热重载的原因。

## 一次请求的完整顺序

入口是 `router` 注册的 `handler`（`runtime.go`），它的顺序是：

```go
copyComponentsPtr(s, req)                        // 1
parseRequestForm(req)                            // 2
interceptors := getInterceptors(req)             // 3
req, err := doBefore(&interceptors, resp, req)   // 4
if err == nil {
	doRequest(s, serviceName, methodName, resp, req) // 5
} else {
	components(req).errorHandlerFunc()(resp, req, err)
}
doAfter(interceptors, resp, req)                 // 6
```

1. `copyComponentsPtr` 把当前 `Server.Components` 指针写进请求 context（键是 `componentsKey`），一次请求生命周期内始终用同一份 `Components`，即使中途发生热重载。
2. `parseRequestForm` 准备参数：表单请求读完 body 后再复原 `req.Body`，非表单请求不碰 body 只把 query 装进 `req.Form`；然后把大写键合并到小写键，并把路由变量并进 `req.Form`（`util.go`）。
3. `getInterceptors` 拼出拦截器链：全局的在前，路由级在后。
4. `doBefore` 正序执行 `Before`。某个 `Before` 返回 error 时，它把链截断成 `(*interceptors)[0:index]` 并记一条 `error in Before():`，返回的 error 交给 `errorHandlerFunc`。失败者以及它后面的拦截器都留在链外。
5. `doRequest` 只在 `doBefore` 没有 error 时执行。
6. `doAfter` 逆序执行，且只跑 `Before` 成功的那些：截断之后传进来的切片里已经没有失败者及其后继。

`doRequest` 依次做：查 hijacker（命中就调用它并返回）、`doPreprocessor`、`switcherFunc` 发起 RPC、`doPostprocessor`、`writeResponse`。前四步任意一步返回 error 都交给 `errorHandlerFunc` 并返回，所以 postprocessor 出错时不会走到 `writeResponse`。`writeResponse` 把响应序列化成 JSON 并设置 `Content-Type: application/json`。

## 组件声明怎么匹配路由

`config.go` 的 `loadMappings` 把每一行按空格切成四列 `[方法, 路径, 名字, 名字]`，`loadComponents` 按段装载：`interceptor`、`preprocessor`、`postprocessor`、`hijacker` 四段各建一张 mux 路由表（存在 `routers[rInterceptor]`、`routers[rPreprocessor]`、`routers[rPostprocessor]`、`routers[rHijacker]`），convertor 单独存进 `convertorMap`。所以一个 URL 上最多只有一个 preprocessor、一个 postprocessor、一个 hijacker，interceptor 则可以是一条链。

方法与组件名都支持逗号分隔，`- GET,POST /hello NameA,NameB` 表示两个方法、两个组件按声明顺序组成链。

取哪一条声明由 `component()` 决定，它用 mux 的 `Match`，按注册顺序返回第一条命中的路由。`audit.go` 的注释把这件事写明：第一条匹配的声明就是生效的那条，后面的不会叠加。路径语法（`/hello`、`/hello/{name}`、`/hello/*`、`/*`）与 `04-routing.md` 讲的是同一套，定义在 `pattern.go` 的 `registerPattern`。

## Components.Reset() 做什么

```go
func (c *Components) Reset() {
	c.commonInterceptors = Interceptors{}
	c.routers = make(map[int]*mux.Router)
	c.convertorMap = make(map[string]Convertor)
	c.errorHandler = nil
}
```

它清空全局拦截器、四张组件路由表、convertor 表和 errorHandler，但不动 `registeredComponents`：注册过的组件实例还在，只是所有声明都没了，需要重新 `Intercept`、`SetPreprocessor` 等。`test/integration_test.go` 大量用它恢复干净状态。

## 完整例子：五类组件接起来

组件写在 `grpcapi/component/components.go`，`proto` 指生成代码包。三段代码合起来需要的 import 是 `context`、`net/http`、`reflect`、`strings`、`time`，以及 `github.com/sirupsen/logrus`、`github.com/vaporz/turbo`、生成代码包。

```go
type startKey struct{}

// AccessLogInterceptor 是全局拦截器：记录耗时，不碰任何全局状态。
type AccessLogInterceptor struct {
	turbo.BaseInterceptor
}

func (a *AccessLogInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	*req = *req.WithContext(context.WithValue(req.Context(), startKey{}, time.Now()))
	return nil
}

func (a *AccessLogInterceptor) After(resp http.ResponseWriter, req *http.Request) error {
	if start, ok := req.Context().Value(startKey{}).(time.Time); ok {
		logrus.Infof("access: %s %s took %s", req.Method, req.URL.Path, time.Since(start))
	}
	return nil
}
```

```go
// TokenInterceptor 是路由级拦截器：校验令牌，注入身份。
type TokenInterceptor struct {
	turbo.BaseInterceptor
}

func (t *TokenInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	token := req.Header.Get("X-Device-Token")
	if token == "" {
		return turbo.Errorf(http.StatusUnauthorized, "missing X-Device-Token")
	}
	turbo.InjectParam(req, "device_id", "device-"+token)
	return nil
}
```

```go
// NormalizePreprocessor 规范化请求参数。
var NormalizePreprocessor turbo.Preprocessor = func(resp http.ResponseWriter, req *http.Request) error {
	if v := req.Form.Get("your_name"); v != "" {
		req.Form.Set("your_name", strings.ToLower(strings.TrimSpace(v)))
	}
	return nil
}

// TimingPostprocessor 检查 RPC 结果，给响应加一个头。
var TimingPostprocessor turbo.Postprocessor = func(resp http.ResponseWriter, req *http.Request, serviceResp interface{}, err error) error {
	if err != nil {
		return nil // 交给 errorHandler 处理
	}
	resp.Header().Add("X-Rpc-Result", "ok")
	return nil
}

// PlainTextHijacker 返回纯文本，用于第三方回调。
var PlainTextHijacker turbo.Hijacker = func(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Add("Content-Type", "text/plain")
	resp.Write([]byte("success"))
}

// convertProtoCommonValues 是转换器：CommonValues 这个类型的值由它构造。
var convertProtoCommonValues turbo.Convertor = func(req *http.Request) reflect.Value {
	result := &proto.CommonValues{}
	result.SomeId = 1111111
	return reflect.ValueOf(result)
}
```

日志用的是 turbo 自己的依赖 `github.com/sirupsen/logrus`，turbo 不额外导出 logger。

在 `InitService` 里注册，它在 HTTP 服务器起来之前执行：

```go
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	s.RegisterComponent("AccessLogInterceptor", &AccessLogInterceptor{})
	s.RegisterComponent("TokenInterceptor", &TokenInterceptor{})
	s.RegisterComponent("NormalizePreprocessor", NormalizePreprocessor)
	s.RegisterComponent("TimingPostprocessor", TimingPostprocessor)
	s.RegisterComponent("PlainTextHijacker", PlainTextHijacker)
	s.RegisterComponent("convertProtoCommonValues", convertProtoCommonValues)
	s.Components.SetCommonInterceptor(&AccessLogInterceptor{})
	return nil
}
```

在 `service.yaml` 里声明：

```yaml
urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TestService SayHello
  - GET /callback TestService SayHello

interceptor:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TokenInterceptor
preprocessor:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} NormalizePreprocessor
postprocessor:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TimingPostprocessor
hijacker:
  - GET /callback PlainTextHijacker
convertor:
  - CommonValues convertProtoCommonValues
```

同一个 `/hello/{...}` 请求的顺序是：`AccessLogInterceptor.Before`、`TokenInterceptor.Before`、`NormalizePreprocessor`、RPC、`TimingPostprocessor`、`writeResponse`、`TokenInterceptor.After`、`AccessLogInterceptor.After`。请求 `/callback` 时 `TokenInterceptor` 不匹配，`PlainTextHijacker` 命中后 RPC 不再调用，响应体是 `success`。

## 相关阅读

- [02-getting-started.md](02-getting-started.md)
- [03-service-yaml.md](03-service-yaml.md)
- [04-routing.md](04-routing.md)
- [06-interceptor.md](06-interceptor.md)
- [07-preprocessor-postprocessor.md](07-preprocessor-postprocessor.md)
- [08-hijacker.md](08-hijacker.md)
- [09-convertor.md](09-convertor.md)
- [10-errors.md](10-errors.md)
