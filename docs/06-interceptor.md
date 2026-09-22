# 拦截器 Interceptor

这篇讲什么：`Interceptor` 的 `Before` / `After` 两个方法的签名与返回值语义，一个失败的拦截器会让哪些代码不再执行，以及用拦截器做鉴权、注入身份、打访问日志的完整写法。

## 签名与语义

```go
type Interceptor interface {
	Before(http.ResponseWriter, *http.Request) error
	After(http.ResponseWriter, *http.Request) error
}
```

两者都在 `component.go` 中定义，`BaseInterceptor` 提供空的默认实现：

```go
type BaseInterceptor struct{}

func (i *BaseInterceptor) Before(resp http.ResponseWriter, req *http.Request) error { return nil }
func (i *BaseInterceptor) After(resp http.ResponseWriter, req *http.Request) error  { return nil }
```

`Before` 的两点语义：

- 返回 `nil`：这个请求继续往后走。
- 返回非 `nil`：`doBefore` 记一条 `error in Before():`，把拦截器链截断到当前下标之前，然后 `handler` 调 `components(req).errorHandlerFunc()(resp, req, err)`。RPC 不会被调用，这个拦截器以及排在它后面的拦截器的 `After` 也不会执行。

`After` 的两点语义：

- 它在响应已经写出之后执行，所以它能追加内容，但改不了已经写出的部分。
- 它返回非 `nil` 只会被记一条 `turbo: error in After():`，`doAfter` 仍然继续跑链上剩下的 `After`，并且 `doAfter` 本身总是返回 `nil`。`After` 的返回值不会触发 error handler。

`doBefore` 与 `doAfter` 都在 `runtime.go`：

```go
func doBefore(interceptors *[]Interceptor, resp http.ResponseWriter, req *http.Request) (request *http.Request, err error) {
	for index, i := range *interceptors {
		err = i.Before(resp, req)
		if err != nil {
			log.Errorln("error in Before(): ", err.Error())
			*interceptors = (*interceptors)[0:index]
			return req, err
		}
	}
	return req, nil
}

func doAfter(interceptors []Interceptor, resp http.ResponseWriter, req *http.Request) (err error) {
	l := len(interceptors)
	for i := l - 1; i >= 0; i-- {
		err = interceptors[i].After(resp, req)
		if err != nil {
			log.Errorln("turbo: error in After(): ", err.Error())
		}
	}
	return nil
}
```

`doAfter` 从最后一个往第一个跑，所以链是“正序进入、逆序退出”的。`doBefore` 返回的 `req` 会替换原变量，所以 `Before` 里用 `*req = *req.WithContext(ctx)` 换掉的 request 对后续拦截器和 RPC 都生效。

被拒请求的日志：`doBefore` 记 `error in Before(): <err>`，error handler 的默认实现 `defaultErrorHandler` 再记一次 `log.Error(err.Error())`。状态码取 `StatusOf(err)`，没有就用 500。

## 拦截器里能做什么

- 读 header：`req.Header.Get("X-Device-Token")`；写 header：`resp.Header().Add(...)`（要在写 body 之前）。
- 鉴权：校验签名、令牌、时间戳，失败时 `return turbo.Errorf(http.StatusUnauthorized, ...)`。
- 注入身份：`turbo.InjectParam(req, key, value)`，注入值在绑定时的优先级高于 path、body 和 query，详见 `11-binding.md`。
- 打访问日志：`Before` 记开始时间，`After` 记耗时与状态。
- 计时、限流、链路标识：共享的单例上不要放计数器以外的请求状态，计时这类每请求数据放 `req.Context()`。

`InjectParam` 的签名在 `binding.go`：

```go
func InjectParam(req *http.Request, key, value string)
```

它把值存进请求 context 里的一张表，`key` 的拼写不敏感（`deviceCode`、`device_code`、`DEVICE_CODE` 等价）。它原地更新 `req`，调用之后必须继续用同一个 `*http.Request`。

## 顺序与常见布局

`getInterceptors`（`runtime.go`）把全局拦截器放在路由级拦截器之前：

> **版本**：v0.6.1 起，全局拦截器与路由级拦截器组成一条链，全局的先跑；在此之前路由自己声明了拦截器就会替换掉全局的那批。同一个版本起，热重载会保留全局拦截器。

常见布局：

- 访问日志、链路追踪、耗时统计放全局（`SetCommonInterceptor`），所有路由都覆盖到。
- 鉴权放路由级，按 URL 精确控制；公开接口不声明即可。
- 与业务无关的全局拦截器要保证不失败：它排在前面，一旦返回 error，后面的鉴权与业务拦截器都不会跑。反过来，鉴权拦截器之后的拦截器可以假定身份已经就绪。

“排在失败拦截器之后的拦截器不会执行”是 `doBefore` 里 `*interceptors = (*interceptors)[0:index]` 这一行的直接结果，测试 `test/integration_test.go` 也钉住了它：`TestInterceptor` 后面接 `BeforeErrorInterceptor` 再跟 `Test1Interceptor` 时，响应只有 `intercepted:interceptor_error:error!`，`Test1Interceptor` 的标记不出现。

## 完整例子：设备令牌鉴权

组件：

```go
package component

import (
	"net/http"
	"strings"

	"github.com/vaporz/turbo"
)

// DeviceTokenInterceptor 校验设备令牌，并把校验出来的身份注入请求。
type DeviceTokenInterceptor struct {
	turbo.BaseInterceptor
}

func (d *DeviceTokenInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	token := strings.TrimSpace(req.Header.Get("X-Device-Token"))
	if token == "" {
		return turbo.Errorf(http.StatusUnauthorized, "missing X-Device-Token")
	}
	if !validDeviceToken(token) {
		return turbo.Errorf(http.StatusForbidden, "invalid X-Device-Token")
	}
	turbo.InjectParam(req, "device_id", token)
	return nil
}

func validDeviceToken(token string) bool {
	return strings.HasPrefix(token, "device-")
}
```

注册，放在 `InitService` 里：

```go
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	s.RegisterComponent("DeviceTokenInterceptor", &DeviceTokenInterceptor{})
	return nil
}
```

`service.yaml` 里声明在需要保护的路由上：

```yaml
urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TestService SayHello

interceptor:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} DeviceTokenInterceptor
```

参数 `device_id` 会被注入进请求。绑定时注入值优先于 path、body 和 query：服务端的 proto 消息里如果有 `deviceId` 这样的字段，它会用注入值填，请求方在 body 或 query 里再传一次也覆盖不了。测试夹具里的 `SayHelloRequest` 没有这个字段，所以注入的 `device_id` 在那里只是留在请求上；夹具中演示注入的 `InjectInterceptor` 注入的是 `your_Name`，对应 `SayHelloRequest.YourName`。用 curl 验证：

```bash
# 缺少令牌：401，body 是 error handler 写出的消息
curl -i "http://127.0.0.1:8085/hello/name"

# 令牌格式不对：403
curl -i -H "X-Device-Token: bad" "http://127.0.0.1:8085/hello/name"

# 通过
curl -i -H "X-Device-Token: device-abc" "http://127.0.0.1:8085/hello/name"
```

`turbo.Errorf` 把状态码附在 error 上（`errors.go`）：

```go
func Errorf(status int, format string, args ...interface{}) error
```

默认 error handler 会用它作为响应的 HTTP 状态码，所以 401 不会被写成 500。想换成自己的响应格式，用 `WithErrorHandler`，见 `10-errors.md`。

## 常见坑

- 单例并发：拦截器实例被所有请求共享。`Before` 里写字段、`After` 里读字段必然串值；用 `req.Context()`。
- 不要在拦截器里改全局状态：改 `Components`、改配置、改包级变量都会影响其他并发请求，而且在热重载期间更危险。
- `req.Form` 在 JSON 请求下的注意点：`parseRequestForm` 对非表单请求只做 `req.Form = req.URL.Query()`，不读 body；所以 JSON 请求里 `req.Form` 只有 URL query，没有 body 字段。要拿 body 里的值请用 `turbo.InjectParam` 之外的绑定路径，或在 preprocessor / 服务端读取。
- 表单请求的 body 会被 `parseRequestForm` 读完再复原，所以 `Before` 里仍然可以读 `req.Body` 做验签。
- 拦截器的 `Before` 里写响应后返回 error，响应里会同时留下你写的片段和 error handler 写的内容；测试夹具 `BeforeErrorInterceptor` 就是这样，输出是 `interceptor_error:error!\n`。

## 相关阅读

- [05-components.md](05-components.md)
- [07-preprocessor-postprocessor.md](07-preprocessor-postprocessor.md)
- [10-errors.md](10-errors.md)
- [11-binding.md](11-binding.md)
- [16-auth-and-route-audit.md](16-auth-and-route-audit.md)
