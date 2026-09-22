# 路由与 urlmapping

`urlmapping` 是 turbo 的路由表，每一行把一条 HTTP 路由绑到一个 RPC 方法。这篇讲这一行怎么被解析、路径模式的准确语义、请求路径里的参数怎么进 RPC、重叠声明谁生效，以及匹配不上时到底回什么。

## 一行 urlmapping 的格式

```text
METHOD /path ServiceName MethodName
```

解析代码是 `Config.loadUrlMap` -> `loadMappings("urlmapping")` -> `appendMap`：

```go
func appendMap(mapping [][4]string, line string) [][4]string {
	values := strings.Split(line, " ")
	HTTPMethod := strings.TrimSpace(values[0])
	url := strings.TrimSpace(values[1])
	v1 := strings.TrimSpace(values[2])
	var v2 string
	if len(values) > 3 {
		v2 = strings.TrimSpace(values[3])
	}
	return append(mapping, [4]string{HTTPMethod, url, v1, v2})
}
```

从这段代码能读出的行为：

- 只按单个空格 `" "` 切分，不合并连续空格，也不认制表符。
- 少于 3 个字段时直接下标越界 panic。启动阶段这个 panic 会终止进程；热重载阶段被 `loadServiceConfigErr` 的 recover 包成 `invalid configuration <file>: ...`，该次变更被忽略。
- 恰好 3 个字段不报错，但第四列方法名是空串。路由会被注册，生成 switcher 时会落进 `No such method[...]` 分支。
- 第 5 个及以后的字段被静默丢弃。
- `Config.validate` 另外要求整张表非空，否则报 `urlmapping is empty, so no route would be served at all`。

## HTTP 方法

`router` 的实现是：

```go
httpMethods := strings.Split(v[0], ",")
r.HandleFunc(path, handler(s, serviceName, methodName)).Methods(httpMethods...)
```

- 多个方法用逗号分隔，例如 `GET,HEAD`。
- 方法不区分大小写：gorilla/mux 的 `Methods` 会把每个声明的方法 `strings.ToUpper`，所以 `get /hello ...` 和 `GET /hello ...` 一样。
- 请求行里的方法按 mux 的 `matchInArray` 逐字比较，HTTP 规范要求大写，实际请求也应当用大写。
- 同一个 RPC 方法可以被多条路由映射，例如 `/hello` 和 `/hello/{your_Name:[a-zA-Z0-9]+}` 都指向 `TestService SayHello`，这是 `test/testservice/service.yaml` 里的真实写法。
- 路径匹配上了但方法不对，走的是 mux 的 405 分支，不是 404，详见下面“匹配不上时”。

## 路径模式

路径模式由 `pattern.go` 的 `matchPattern` / `registerPattern` 和 `component.go` 的 `setComponent` 共同定义，语义如下。

| 模式 | 匹配的路径 |
|---|---|
| `/hello` | 只有 `/hello` |
| `/hello/{name}` | `/hello/` 加恰好一个非空段 |
| `/hello/{name:[a-zA-Z]+}` | 同上，正则部分由 mux 在匹配时执行 |
| `/hello/*` | `/hello` 本身，以及它下面的所有路径 |
| `/*` | 全部路径 |
| `/` | 只有根路径 `/` |

几个必须记住的点：

1. `/hello` 是字面量，不是前缀。`/hello/world` 不会命中 `- GET /hello ...` 这行。
2. `{...}` 段代表恰好一个非空路径段。`/hello/{name}` 不匹配 `/hello`，也不匹配 `/hello/a/b`。
3. `/hello/*` 是显式通配，`registerPattern` 会为它注册两条 mux 路由：一条 `m.Handle("/hello")` 覆盖模式本身，一条 `m.PathPrefix("/hello/")` 覆盖其下所有路径。`/*` 走 `m.PathPrefix("/")`。
4. `/` 只匹配根路径。旧版本里“以 `/` 结尾表示其下全部”的语义已经废弃，想要全局就写 `/*`。

> **版本**：v0.6.0 起使用这套模式语言（`pattern.go`）。此前 `- GET / X` 会被当成覆盖整个服务的全局声明，因为 `setComponent` 内部把尾随 `/` 当成前缀；这层隐式规则已移除。

### 路径段数必须一致

非通配模式下，模式和路径都经 `pathSegments` 切分后逐段比较，段数不同直接不匹配：

```go
patternParts := pathSegments(pattern)
pathParts := pathSegments(path)
if len(patternParts) != len(pathParts) {
	return false
}
```

`pathSegments` 不做任何归一化，`/hello/` 和 `/hello` 是两个不同的路径（前者切成 `["hello", ""]`，后者切成 `["hello"]`），mux 也这么认为，审计和路由器因此保持一致。`pattern_test.go` 用同一组模式与路径跑两边，要求结论相同。

审计侧还有一点要知道：`matchPattern` 判断占位符时只看“这一段含不含 `{`、路径那一段非空”，并不执行 `{name:[a-zA-Z]+}` 里的正则，所以对带正则的段它会当成普通通配，倾向于把路由判为“已被覆盖”。

## 请求路径里的参数怎么进 RPC

一次请求的入口是 `handler`，它先 `copyComponentsPtr` 把当前 `Components` 指针放进 context，再 `parseRequestForm`，然后调用生成代码里的 switcher：

```go
func handler(s Servable, serviceName, methodName string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
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
	}
}
```

switcher 最终调用 `turbo.BuildRequest`（gRPC）或 `turbo.BuildThriftRequest`（Thrift）构造请求体。路径参数有两个来源：

- JSON 请求（`Content-Type` 含 `application/json`）：正文用 `jsonpb` 反序列化后，`setPathParams` 从 `mux.Vars(req)` 取值填进消息字段。
- 表单 / query 请求：`BuildStructErr` 走 `findValue`，优先级是“注入值 > 路径 > query/form”。

字段名匹配是忽略拼写的。`findPathParamValue` 会先试 `lookupKeys` 给出的三种写法（Go 字段名、全小写、snake_case），再用 `normalizeKey`（去下划线后小写）兜底，所以路由里写 `{your_Name}`、proto 字段叫 `YourName`，也能对上：

```yaml
urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TestService SayHello
```

```bash
curl 'http://127.0.0.1:8085/hello/vaporz'
```

绑定规则的完整说明在 `11-binding.md`。

## 注册顺序：第一条生效

`router` 按 `Config.mappings[urlServiceMaps]` 的顺序逐行注册：

```go
routes := s.ServerField().Config.mappings[urlServiceMaps]
for _, v := range routes {
	r.HandleFunc(path, handler(s, serviceName, methodName)).Methods(httpMethods...)
}
log.Infof("turbo: %d route(s) registered", len(routes))
```

gorilla/mux 返回第一个匹配上的路由，所以同一条请求路径有多条声明时，**排在前面的那条生效**，后面的永远轮不到。把更具体的模式放在更宽的模式前面，例如 `/hello/{name:[a-z]+}` 要写在 `/hello/*` 前面。

启动日志会逐条打印路由，格式是 `route: GET /hello -> TestService.SayHello`，最后跟一行 `turbo: <N> route(s) registered`。

路由审计另外会为重叠的拦截器声明打 warning（注意它检查的是 `interceptor` 段，不是 `urlmapping` 本身）：

```text
route audit: GET /hello is matched by 2 interceptor declarations [[BaseInterceptor] [TestInterceptor]], only the first one runs
```

也就是说，拦截器链的裁决和路由一样是“第一条生效”，重复声明不会叠加。细节见 `16-auth-and-route-audit.md`。

## 匹配不上时：404 与 405

`router` 末尾设置了 `r.NotFoundHandler = notFoundHandler()`。路径没命中任何路由时：

```go
log.Errorf("turbo: 404 no route for %s %s, host=%s, remote=%s, user-agent=%q",
	req.Method, req.URL.Path, req.Host, req.RemoteAddr, req.UserAgent())
http.NotFound(resp, req)
```

- 日志级别是 error，字段有 method、path、host、remote、user-agent，**刻意不含 query**（query 里可能带 token 或签名）。
- 响应体是 Go 标准库的 `404 page not found`，状态码 404，`Content-Type` 为 `text/plain; charset=utf-8`。

方法不匹配是另一条路：路径能命中某条路由但方法不在 `Methods` 列表里时，mux 返回 405，turbo 没有设置 `MethodNotAllowedHandler`，所以响应是空的 405，**不经过 `notFoundHandler`，也就没有上面那行 error 日志**。

还有一个 mux 自身的行为：非规范路径（例如 `//hello`、`/a/../hello`）会先被 `cleanPath` 301 重定向，然后才进入匹配。

## 服务复用（multiplexing）

turbo 允许一个网关进程后端挂多个 gRPC / Thrift 服务。

- `config.grpc_service_name` / `config.thrift_service_name` 用逗号列出服务名。`GrpcServiceNames()` 的实现是 `strings.Split(names, ",")`，不做 TrimSpace，多个名字之间不要留空格。
- `urlmapping` 的第三列是服务名，它会被传给 switcher，switcher 用 `s.Service(serviceName)` 取客户端：gRPC 是 `gClient.grpcServiceMap[serviceName]`，Thrift 是 `tClient.thriftService[serviceName]`。
- 这个 map 的 key 由代码里的 `GrpcClient` / `ThriftClient` 决定；生成模板按 `grpc_service_name` 里的名字各生成一个 key。所以第三列必须和 map key 完全一致。

`test/testservice` 是一个真实的两服务例子。`test/testservice/grpcapi/component/components.go`：

```go
func GrpcClient(conn *grpc.ClientConn) map[string]interface{} {
	return map[string]interface{}{
		"TestService":    proto.NewTestServiceClient(conn),
		"MinionsService": proto.NewMinionsServiceClient(conn),
	}
}
```

`test/testservice/service.yaml` 里两个服务各有路由：

```yaml
config:
  # 夹具的 GrpcClient 手写了两个 key，所以第三列可以用 MinionsService；
  # 生成模板只会按这里列出的名字生成 key，多服务时要在 component 代码里补齐
  grpc_service_name: TestService
  grpc_service_host: 127.0.0.1
  grpc_service_port: 50065

urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TestService SayHello
  - GET /hello TestService SayHello
  - POST /eat MinionsService Eat
```

Thrift 侧同理：客户端用 `thrift.NewTMultiplexedProtocol(iprot, "<ServiceName>")` 构造，服务端用 `thrift.NewTMultiplexedProcessor()` 按同样的名字注册 processor。`test/testservice/thriftapi/component/components.go` 里同样是两个 key：

```go
func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) map[string]interface{} {
	iprot := f.GetProtocol(trans)
	return map[string]interface{}{
		"TestService":    t.NewTestServiceClientProtocol(trans, iprot, thrift.NewTMultiplexedProtocol(iprot, "TestService")),
		"MinionsService": t.NewMinionsServiceClientProtocol(trans, iprot, thrift.NewTMultiplexedProtocol(iprot, "MinionsService")),
	}
}
```

注意生成模板只按 `GrpcServiceNames()[0]` 生成一个 Thrift key（`creator.go` 的 `generateThriftHTTPComponent` 传的是 `c.c.GrpcServiceNames()[0]`），要多服务得像上面这个夹具一样在代码里补齐。

## 完整示例：新增一条路由

假设服务里已经有一个 `GetUser` 方法，现在要把 `GET /users/{id:[0-9]+}` 暴露出去。

第一步，在 `service.yaml` 里加一行路由：

```yaml
urlmapping:
  - GET /hello TestService SayHello
  - GET /users/{id:[0-9]+} UserService GetUser
```

第二步，确认 `UserService` 在 `GrpcClient` 的 map 里注册过，并且名字与第三列一致：

```go
func GrpcClient(conn *grpc.ClientConn) map[string]interface{} {
	return map[string]interface{}{
		"TestService": proto.NewTestServiceClient(conn),
		"UserService": proto.NewUserServiceClient(conn),
	}
}
```

第三步，如果 `GetUser` 是新增的 RPC 方法，重新生成 switcher（生成输入就是 `urlmapping`，见 `12-code-generation.md`）：

```bash
turbo generate github.com/you/yourservice -r grpc -I $PWD
```

`switcher` 是编译进二进制的包级变量，所以新增服务或新增方法必须重新生成并重新构建。相反，只是把已有的方法换一条路径暴露，改完 `service.yaml` 保存即可，热重载会重新注册路由表：

```bash
# 修改 service.yaml 后，日志里会出现
# Reloading configuration...
# Configuration reloaded
```

第四步，验证：

```bash
curl 'http://127.0.0.1:8085/users/42'
```

如果响应是 `No such method[GetUser]` 或 `No such service[UserService]`，说明 switcher 里没有这个分支，回到第三步；如果是一行 `turbo: 404 no route for GET /users/42, ...`，说明路径模式没匹配上，检查段数与模式写法。

## 相关阅读

- [03-service-yaml.md](03-service-yaml.md)
- [05-components.md](05-components.md)
- [11-binding.md](11-binding.md)
- [12-code-generation.md](12-code-generation.md)
- [13-grpc-thrift.md](13-grpc-thrift.md)
- [15-hot-reload.md](15-hot-reload.md)
- [16-auth-and-route-audit.md](16-auth-and-route-audit.md)
- [19-troubleshooting.md](19-troubleshooting.md)
- [appendix-cheatsheet.md](appendix-cheatsheet.md)
