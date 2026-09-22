# 劫持器 Hijacker

这篇讲什么：`Hijacker` 的签名、命中它之后请求与 RPC 的关系、为什么它必须先有一条 `urlmapping` 路由，以及用它在回调接口上返回纯文本或图片的完整写法。

## 签名：没有返回值

```go
type Hijacker func(http.ResponseWriter, *http.Request)
```

它在 `component.go` 里定义，是函数类型。与拦截器不同，它没有返回值：状态码、响应头、响应体全部由你自己写。想表达失败就自己调 `http.Error` 或 `resp.WriteHeader`，框架不会替你决定。

## 命中之后不再走 RPC

查找与调用发生在 `runtime.go` 的 `doRequest` 开头：

```go
func doRequest(s Servable, serviceName, methodName string, resp http.ResponseWriter, req *http.Request) {
	if hijack := components(req).Hijacker(req); hijack != nil {
		hijack(resp, req)
		return
	}
	err := doPreprocessor(resp, req)
	...
}
```

命中后立刻 `return`，所以下面这些都不会发生：

- `doPreprocessor` 不执行；
- `switcherFunc` 不执行，也就是 RPC 完全不被调用；
- `doPostprocessor` 不执行；
- `writeResponse` 不执行，响应不会被序列化成 JSON，也不会自动加 `Content-Type: application/json`。

但请求仍然经过 `handler` 的完整前半段与后半段：

```go
copyComponentsPtr(s, req)
parseRequestForm(req)
interceptors := getInterceptors(req)
req, err := doBefore(&interceptors, resp, req)
if err == nil {
	doRequest(s, serviceName, methodName, resp, req)
} else {
	components(req).errorHandlerFunc()(resp, req, err)
}
doAfter(interceptors, resp, req)
```

也就是说，`Before` 链条先跑完（全部成功才轮到 hijacker），hijacker 返回之后 `doAfter` 仍然执行。这与拦截器的“环绕”是互补关系：hijacker 决定这段请求做什么，拦截器决定在它前后还做什么。

它取的是哪一条声明，由 `component()` 用 mux 的 `Match` 按注册顺序取第一条命中的路由；一个 URL 上只有一个 hijacker 会生效。

## 必须先有一条 `urlmapping` 路由

`handler` 是从 `urlmapping` 生成的路由注册进去的：

```go
func router(s Servable) *mux.Router {
	r := mux.NewRouter()
	routes := s.ServerField().Config.mappings[urlServiceMaps]
	for _, v := range routes {
		httpMethods := strings.Split(v[0], ",")
		path := v[1]
		serviceName := v[2]
		methodName := v[3]
		log.Infof("route: %s %s -> %s.%s", v[0], path, serviceName, methodName)
		r.HandleFunc(path, handler(s, serviceName, methodName)).Methods(httpMethods...)
	}
	log.Infof("turbo: %d route(s) registered", len(routes))
	r.NotFoundHandler = notFoundHandler()
	return r
}
```

hijacker 只在 `handler` 内部被查找，而 `handler` 只会挂在 `urlmapping` 的某条路由上。所以某个路径没有 `urlmapping` 条目时，请求直接落到 `notFoundHandler`，hijacker 根本没机会被查找。README 的 Features 一节也把 hijacker 描述为接管请求的一类组件，它依赖已经存在的路由。

实践上给回调地址写一条路由，目标可以指向任意一个存在的 RPC；因为 hijacker 命中后 RPC 不会被调用，`serviceName` 与 `methodName` 只是为了让配置通过校验：

```yaml
urlmapping:
  - GET /callback TestService SayHello
hijacker:
  - GET /callback PlainTextHijacker
```

`config.go` 的 `validate` 会拒绝一条路由都没有的配置（`urlmapping is empty, so no route would be served at all`），所以“只挂 hijacker 不写 urlmapping”是不可行的。

## 适用场景

需要返回非 JSON 的接口用 hijacker，而不是 interceptor：

- 第三方回调要求响应体是纯文本：例如支付、设备、消息平台的回调只认 `success`，返回 `{"code":0}` 会被判为失败并触发重试。
- 返回图片、文件、HTML 页面，或需要自己控制 `Content-Type`、`Content-Disposition` 的下载接口。
- 需要原始字节流、SSE（`text/event-stream`）或自己接管连接。
- 纯健康检查、灰度开关这类不查后端的接口。

用 interceptor 做不到的原因是：interceptor 的 `After` 在 `writeResponse` 之后执行，而 `writeResponse` 已经写了一个 JSON body；即便在 `Before` 里写了纯文本，RPC 仍会被调用，响应里还会混进 JSON。要在 interceptor 里彻底拦住 RPC，只能返回 error，那会走 error handler，状态码与响应体又不受你控制。

## 完整示例：回调返回纯文本，另一个路径返回图片

组件：

```go
package component

import (
	"net/http"

	"github.com/vaporz/turbo"
)

// CallbackHijacker 给第三方回调返回纯文本 success。
var CallbackHijacker turbo.Hijacker = func(resp http.ResponseWriter, req *http.Request) {
	// 回调里的签名校验放在这里做，parseRequestForm 已经复原过表单 body。
	if req.Header.Get("X-Callback-Signature") == "" {
		http.Error(resp, "missing signature", http.StatusUnauthorized)
		return
	}
	resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte("success"))
}

// IconHijacker 返回一张图片，响应头与字节都由它自己决定。
var IconHijacker turbo.Hijacker = func(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "image/png")
	resp.Header().Set("Cache-Control", "public, max-age=3600")
	resp.WriteHeader(http.StatusOK)
	resp.Write(iconPNG) // 你自己的图片字节
}

var iconPNG = []byte{0x89, 0x50, 0x4e, 0x47} // 这里只放 PNG 魔数，实际用完整图片
```

注册：

```go
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	s.RegisterComponent("CallbackHijacker", CallbackHijacker)
	s.RegisterComponent("IconHijacker", IconHijacker)
	return nil
}
```

`service.yaml`：

```yaml
urlmapping:
  - GET /callback TestService SayHello
  - GET /icon.png TestService SayHello

hijacker:
  - GET /callback CallbackHijacker
  - GET /icon.png IconHijacker
```

验证：

```bash
curl -i -H "X-Callback-Signature: abc" "http://127.0.0.1:8085/callback"
# HTTP/1.1 200 OK
# Content-Type: text/plain; charset=utf-8
# success

curl -i "http://127.0.0.1:8085/callback"
# HTTP/1.1 401 Unauthorized
# missing signature
```

对照一下 `/icon.png`：响应是 `image/png`，不是 `application/json`，因为 `writeResponse` 根本没被调用。

## 常见坑

- 不要既写 hijacker 又期待 postprocessor 或 `writeResponse` 生效：`doRequest` 命中 hijacker 就返回，postprocessor、RPC、JSON 序列化全都不会执行。只有拦截器的 `Before` 与 `After` 仍然生效。
- 不要忘记自己设置 `Content-Type`：框架不会替你设。`resp.Write` 第一次调用会隐式发送 200，所以要先 `resp.WriteHeader` 再写 body。
- 不要忘记注册路由：没有 `urlmapping` 条目时请求是 404，排查时先看 `turbo: 404 no route for ...` 这条日志（`notFoundHandler`）。
- 组件是单例并发调用的，hijacker 里同样不要放请求状态；每请求数据用 `req.Context()`。
- 表单请求的 body 在 `parseRequestForm` 之后仍然可读，所以回调验签可以在 hijacker 里做；但它读到的是复原后的 body，读到一半再交给下游也没问题。

## 相关阅读

- [05-components.md](05-components.md)
- [06-interceptor.md](06-interceptor.md)
- [07-preprocessor-postprocessor.md](07-preprocessor-postprocessor.md)
- [10-errors.md](10-errors.md)
- [19-troubleshooting.md](19-troubleshooting.md)
