# 前置处理器与后置处理器

这篇讲什么：`Preprocessor` 与 `Postprocessor` 的签名、它们各自在请求生命周期的哪一步执行、与拦截器的区别，以及用它们规范化请求参数、给响应加信息的完整写法。

## 签名

```go
type Preprocessor func(http.ResponseWriter, *http.Request) error

type Postprocessor func(http.ResponseWriter, *http.Request, interface{}, error) error
```

两者都定义在 `component.go`，都是函数类型：直接写函数字面量，或把具名函数赋给这个类型的变量。

Preprocessor 的调用点 `doPreprocessor` 与 Postprocessor 的调用点 `doPostprocessor` 都在 `runtime.go`，两者都由 `doRequest` 调用：

```go
err := doPreprocessor(resp, req)
if err != nil {
	components(req).errorHandlerFunc()(resp, req, err)
	return
}
serviceResp, err := switcherFunc(s, serviceName, methodName, resp, req)
if err != nil {
	components(req).errorHandlerFunc()(resp, req, err)
	return
}
err = doPostprocessor(s, resp, req, serviceResp, err)
if err != nil {
	components(req).errorHandlerFunc()(resp, req, err)
	return
}
writeResponse(s, resp, req, serviceResp)
```

## 与 interceptor 的区别

三者都围绕同一个请求，但位置和覆盖面不同：

| | 执行位置 | 拿到什么 | 覆盖面 |
|---|---|---|---|
| `Interceptor.Before` | 参数解析之后、RPC 之前 | `*http.Request` | 一条链，可有全局拦截器 |
| `Preprocessor` | `doRequest` 开头、RPC 之前 | `*http.Request` | 一个 URL 声明一个 |
| `Postprocessor` | `switcherFunc` 返回之后、`writeResponse` 之前 | RPC 响应对象与 RPC 错误 | 一个 URL 声明一个 |
| `Interceptor.After` | `writeResponse` 之后 | `*http.Request` | 一条链，逆序执行 |

几个可以直接从代码读出来的差别：

- Preprocessor 在 RPC 请求构造之前跑：它在 `switcherFunc` 之前，而 `switcherFunc` 里才调 `turbo.BuildRequest` 构造请求。它拿不到 RPC 响应。
- Postprocessor 拿到 RPC 响应：第三个参数是 `serviceResp`，也就是 `switcherFunc` 的第一个返回值。
- Postprocessor 拿到的第四个参数在实际调用路径上总是 `nil`：`doRequest` 里只有 `switcherFunc` 返回 `err == nil` 时才会走到 `doPostprocessor`，而 RPC 失败时 `err != nil` 已经交给 error handler 并 `return` 了。签名保留了 `error` 参数，但“把 RPC 错误交给 postprocessor”不会发生。想处理 RPC 错误请用 error handler，见 `10-errors.md`。
- Postprocessor 在 `writeResponse` 之前跑，所以它可以通过 `resp.Header().Add(...)` 影响响应头；Interceptor 的 `After` 在 `writeResponse` 之后跑，只能追加 body。
- 一个 URL 上最多只有一个 preprocessor 和一个 postprocessor（`component()` 取第一条匹配的声明），而 interceptor 可以是一条链。

## 返回值语义

Preprocessor：

```go
func doPreprocessor(resp http.ResponseWriter, req *http.Request) error {
	if pre := components(req).Preprocessor(req); pre != nil {
		if err := pre(resp, req); err != nil {
			log.Println(err.Error())
			return fmt.Errorf("turbo: encounter error in preprocessor for %s, error: %w", req.URL, err)
		}
	}
	return nil
}
```

返回 error 时它被包一层 `turbo: encounter error in preprocessor for <URL>, error: <原错误>`，然后交给 error handler。包裹用的是 `%w`，所以 `StatusOf` 能穿透到里层的错误：在 preprocessor 里 `return turbo.Errorf(http.StatusTeapot, "teapot")`，响应状态码就是 418，body 是包裹后的那条消息（`TestErrorStatusCodes` 里的断言）。

Postprocessor：

```go
func doPostprocessor(s Servable, resp http.ResponseWriter, req *http.Request, serviceResponse interface{}, err error) error {
	if post := components(req).Postprocessor(req); post != nil {
		if err := post(resp, req, serviceResponse, err); err != nil {
			log.Println(err.Error())
			return fmt.Errorf("turbo: encounter error in postprocessor for %s, error: %w", req.URL, err)
		}
	}
	return nil
}
```

返回 error 时同样包一层并交给 error handler，而且不再执行 `writeResponse`，也就是说 RPC 已经成功但响应不会写出，状态码由 error 决定（没有状态就是 500）。这个位置适合做严格校验：宁可报错也不要写出一个不合规的响应。

## 执行顺序与声明方式

在 `service.yaml` 里，preprocessor 与 postprocessor 各占一段，格式都是 `METHOD /pattern ComponentName`：

```yaml
urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TestService SayHello

preprocessor:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} NormalizePreprocessor
postprocessor:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} AddServerHeader
```

`loadMappings`（`config.go`）把每行按空格切成四列，组件名取第三列，第四列只对 `urlmapping` 有意义。方法列表支持逗号分隔（`GET,POST`）。名字没注册过时 `getComponentByName` 会 panic，错误信息是 `no such component: <name>, forget to register?`。

同一次匹配只生效第一条声明：`component()` 用 mux 的 `Match` 按注册顺序取第一条命中的路由。想要两个 preprocessor，只能用不同的 URL 模式分别声明，它们不会叠加。

组件注册在 `InitService` 里：

```go
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	s.RegisterComponent("NormalizePreprocessor", NormalizePreprocessor)
	s.RegisterComponent("AddServerHeader", AddServerHeader)
	return nil
}
```

## 完整例子

### 规范化的 preprocessor

```go
package component

import (
	"net/http"
	"strings"

	"github.com/vaporz/turbo"
)

// NormalizePreprocessor 把查询参数与表单值去掉首尾空白、统一小写，
// 让服务端实现只需要处理一种拼写。
var NormalizePreprocessor turbo.Preprocessor = func(resp http.ResponseWriter, req *http.Request) error {
	if req.Form == nil {
		return nil
	}
	for key, values := range req.Form {
		normalized := make([]string, len(values))
		for i, v := range values {
			normalized[i] = strings.ToLower(strings.TrimSpace(v))
		}
		req.Form[key] = normalized
	}
	return nil
}
```

注意它改的是 `req.Form`：`parseRequestForm` 已经在 `handler` 里先把 query、表单 body、路由变量合并进 `req.Form`，所以 preprocessor 看到的是完整的一份参数。绑定流程读的也是 `req.Form`，因此这里改的值会进入 RPC 请求。

### 给响应加信息的 postprocessor

```go
// AddServerHeader 检查 RPC 结果并给响应加两个头。
var AddServerHeader turbo.Postprocessor = func(resp http.ResponseWriter, req *http.Request, serviceResp interface{}, err error) error {
	if err != nil {
		return nil // 实际调用路径上 err 总是 nil；保留这行是为了不掩盖将来的变化
	}
	resp.Header().Add("X-Server", "turbo")
	resp.Header().Add("X-Server-Result", "ok")
	return nil
}
```

postprocessor 在 `writeResponse` 之前跑，所以这两个头会出现在真实响应里。想在 postprocessor 里报耗时，正确的做法是让全局拦截器的 `Before` 把起点写进请求 context，再在这里读出来（见 `06-interceptor.md` 的 `startKey` 例子）；postprocessor 与拦截器用的是同一个 `*http.Request`，所以 context 是通的。

### 用 curl 看效果

```bash
curl -i "http://127.0.0.1:8085/hello/name?your_name=%20NAME%20"
# HTTP/1.1 200 OK
# Content-Type: application/json
# X-Server: turbo
# X-Server-Result: ok
```

`your_name` 的值被 preprocessor 规范成 `name`，再交给 `SayHello`；响应头由 postprocessor 补上。

## 常见坑

- preprocessor 返回 error 是“拒绝这个请求”，不是“跳过这个组件”。响应由 error handler 写，RPC 不执行。
- postprocessor 返回 error 会丢掉已经拿到的 RPC 响应，并且状态码多半是 500；只有确实无法产出合规响应时才这么返回。
- postprocessor 里不要试图改 `serviceResp` 来影响输出：它是 `interface{}`，直接改字段是否有用取决于类型与是否传指针；要过滤 JSON 字段请用 `filter_proto_json` 系列配置。
- 组件是单例并发调用的，两个处理器里都不要存请求状态；上面例子里用 `req.Context()` 和 `req.Form`，两者都是每请求的。

## 相关阅读

- [05-components.md](05-components.md)
- [06-interceptor.md](06-interceptor.md)
- [10-errors.md](10-errors.md)
- [11-binding.md](11-binding.md)
- [14-logging.md](14-logging.md)
