# 鉴权与启动路由审计

turbo 自己不实现鉴权，它做的是另一件事：在每次加载配置时审计所有路由，检查每条路由的有效拦截器链里有没有你声明过的“鉴权拦截器”，把“加了一条路由、忘了加拦截器”这种静默漏洞变成一个能在启动时看见的失败。这篇讲 `auth` 段怎么配、审计怎么裁决、违规时启动和热重载分别发生什么。

## turbo 不做鉴权

框架里没有内置的登录、token 校验、权限模型，鉴权完全靠拦截器（`Interceptor`）实现，写法见 `06-interceptor.md`。

“某个拦截器算不算鉴权拦截器”完全由声明决定，turbo 不看它的语义、不读它的代码，只在 `auth.interceptors` 里按**组件名**做成员判断。同一个组件，你把它列进去它就是鉴权拦截器，不列就不是：

```yaml
auth:
  interceptors:
    - SessionInterceptor   # 只认这个名字
```

因此审计能抓住的是“你忘了在这条路由上放你声明过的那个拦截器”，抓不住的是“这个拦截器本身写错了”。

## auth.interceptors

- 类型：字符串列表，值必须是 `RegisterComponent` 时用的组件名，大小写敏感。
- 作用：给出“哪些名字算鉴权拦截器”的集合，供审计判断。
- 为空时 `authConfig.enforcing()` 返回 false：审计只记录每条路由的裁决，不因为缺鉴权而拒绝配置。这是给还没接入鉴权的服务留的过渡状态。
- 列出但从未注册的名字不会报错，它只是永远匹配不上任何链。

## auth.public_routes

- 类型：字符串列表，每条写成 `METHOD /path`。
- 作用：逐条声明“这条路由允许没有鉴权”。命中公开列表的路由直接记为 public，跳过违规判定。
- 规范化规则在 `routeKey`：

```go
func routeKey(route string) string {
	fields := strings.Fields(route)
	if len(fields) < 2 {
		return strings.ToUpper(strings.TrimSpace(route))
	}
	return strings.ToUpper(fields[0]) + " " + fields[1]
}
```

由此得到四条必须记住的规则：

1. 方法大小写不敏感：`get /health` 和 `GET /health` 是同一个键。
2. 方法名和路径之间的空白随便写：`strings.Fields` 按任意空白切分，`GET   /health` 也能归一化。
3. 路径大小写敏感，也不会被归一化：它按原样与 `urlmapping` 解析出来的 `path` 比较。
4. 路径要写 `urlmapping` 里的**模式原文**，不是实际请求值。路由是 `- GET /users/{id:[0-9]+} UserService GetUser`，public route 就要写 `GET /users/{id:[0-9]+}`，写 `GET /users/1` 不生效。

另外两条来自判定处的代码（`public := auth.publicRoutes[method+" "+path]`）：

- `urlmapping` 一行的每个方法都会被单独审计（方法循环里 `total++`），所以 `GET,HEAD /x` 是两条路由，public_routes 也要写 `GET /x` 和 `HEAD /x` 两条。
- 超过两个字段的部分被 `routeKey` 丢弃，写第三个字段不会报错，但没有意义。

## 生效范围：有效链 = 全局拦截器 + 路由声明的拦截器

一次请求真正跑起来的链由 `getInterceptors` 决定：全局的（`SetCommonInterceptor`）在前，路由自己声明的在后：

```go
common := components(req).CommonInterceptors()
routeLevel := components(req).Interceptors(req)
switch {
case len(common) == 0:
	return routeLevel
case len(routeLevel) == 0:
	return common
}
// CommonInterceptors 和 Interceptors 交回的是 Components 持有的切片，
// 所以新建一个而不是往它们里面 append
chain := make([]Interceptor, 0, len(common)+len(routeLevel))
chain = append(chain, common...)
return append(chain, routeLevel...)
```

审计用同一个顺序拼出有效链，所以用 `SetCommonInterceptor` 装的全局鉴权拦截器也能满足审计。它通过 `namesOf` 把拦截器实例反查回注册名：

```go
for name, component := range registered {
	if component == interface{}(interceptor) {
		names = append(names, name)
	}
}
```

这里比较的是值本身，所以要让审计认得出来，必须把**同一个实例**既 `RegisterComponent("SessionInterceptor", inst)` 又 `SetCommonInterceptor(inst)`。另外注册一个同类型的新实例，指针不同，审计查不到名字：如果这条路由自己也没有拦截器声明，它会被算作“无法核对”并记 warning；如果路由自己声明了链而链里没有鉴权拦截器，仍然按未鉴权处理。

> **版本**：v0.6.0 起拦截器改成链式（全局先跑，路由级跟着跑）；v0.6.1 起 `SetCommonInterceptor` 装的全局拦截器在配置重载后仍然存活。`Server.loadComponents` 重建 `Components` 时会把 `registeredComponents` 和 `commonInterceptors` 一起带过去。

## fail-closed：违规就拒绝

审计在每次 `Server.loadComponents` 里执行，也就是启动（`startHTTPServer` 调用它）和每次重载重建时。返回的 error 被 `panicIf` 升级成 panic，两种场景后果不同：

- 启动：panic 从 `startHTTPServer` 一路抛到调用方，进程直接起不来。这是它存在的意义：只有启动失败才能逼着人补上拦截器声明。
- 热重载：`reloadComponents` -> `loadComponentsErr` 把 panic recover 成 error，`reloadComponents` 把 `server.Config` 恢复成 `previous`，随后日志出现

  ```text
  turbo: configuration reload failed, keeping the running configuration: refusing this configuration: ...
  ```

  正在生效的配置继续服务，watcher 继续等下一次变更。`reload_test.go` 固定了这次重建在写锁下进行的行为。

违规时返回的错误文本形如：

```text
refusing this configuration: 1 route(s) would be served without any declared auth interceptor (list a route under auth.public_routes if it is meant to be public):
  POST /refunds -> RefundService.Refund: neither its route interceptors [[TraceInterceptor RateLimitInterceptor AuditInterceptor]] nor the common ones [] declare an auth interceptor
```

冒号后面逐条列出每个违规路由。`reason` 有两种：

- `it declares no interceptor and the server has no common interceptor`：这条路由既没有自己的拦截器声明，服务也没有全局拦截器。
- `neither its route interceptors %v nor the common ones %v declare an auth interceptor`：有链，但链里没有 `auth.interceptors` 里的名字。

## 审计日志的分级

> **版本**：v0.6.2 起审计日志分两级：逐条裁决是 `debug`，另有一行 `info` 汇总。

逐条裁决（`debug`）：

```text
route audit: GET /health -> HealthService.Check [public, auth=common:[] + route:[]]
route audit: GET /users/{id:[0-9]+} -> UserService.GetUser [auth=common:[] + route:[SessionInterceptor], authenticated]
route audit: GET /hello -> TestService.SayHello [auth=common:[] + route:[]]
```

第一行是公开路由，第二行是判定为已鉴权的路由，第三行是“没有声明 auth.interceptors、审计只报告”时的样子。

汇总（`info`，每个服务每次加载一行）：

```text
route audit: 6 route(s), 4 authenticated, 2 public, 0 unprotected
```

可能带两个后缀：

- `, N unverifiable (a common interceptor is not registered under a name)`：服务确实装了全局拦截器，但至少有一个没注册名字，审计无法核对它是不是鉴权拦截器，这类路由记 warning 而不是违规。

  ```text
  route audit: GET /hello -> TestService.SayHello [auth=common:[] + route:[], cannot be verified: a common interceptor is not registered under a name]
  ```

- ` (no auth.interceptors declared, so the audit only reports)`：没有声明 `auth.interceptors`，审计只报告裁决。

违规路由本身记 `error`：

```text
route audit: POST /refunds -> RefundService.Refund [UNPROTECTED: neither its route interceptors [TraceInterceptor] nor the common ones [] declare an auth interceptor]
```

重叠的拦截器声明记 `warning`：

```text
route audit: GET /hello is matched by 2 interceptor declarations [[BaseInterceptor] [TestInterceptor]], only the first one runs
```

这条 warning 是给“同一条路径声明了多条拦截器”的配置用的：只有第一条会跑（见 `04-routing.md`），审计仍然把重复声明报出来，避免“写了却没生效”悄悄发生。

## 完整示例

一个服务声明 6 个鉴权拦截器，两条公开路由，其余路由各自声明鉴权拦截器。下面出现的 `SessionInterceptor` 之类都是服务自己用 `RegisterComponent` 注册的名字，turbo 只做字符串匹配，不要求它们真的叫这个名字。

```yaml
config:
  http_port: 8085
  grpc_service_name: HealthService,UserService,OrderService,SettleService
  grpc_service_host: 127.0.0.1
  grpc_service_port: 50065

urlmapping:
  - GET /health HealthService Check
  - POST /login UserService Login
  - GET /users/{id:[0-9]+} UserService GetUser
  - POST /orders OrderService CreateOrder
  - DELETE /orders/{id:[0-9]+} OrderService DeleteOrder
  - POST /internal/settle SettleService Settle

# 每行三个字段：方法（可逗号列多个）、路径模式、组件名（可逗号列多个，中间不要空格）
interceptor:
  - GET /users/{id:[0-9]+} SessionInterceptor
  - POST /orders JwtInterceptor,SignatureInterceptor
  - DELETE /orders/{id:[0-9]+} SignatureInterceptor
  - POST /internal/settle ApiKeyInterceptor,InternalOnlyInterceptor

auth:
  # 只认名字，不认语义
  interceptors:
    - SessionInterceptor
    - JwtInterceptor
    - SignatureInterceptor
    - ApiKeyInterceptor
    - InternalOnlyInterceptor
    - AdminInterceptor
  # 路径写 urlmapping 里的模式原文
  public_routes:
    - GET /health
    - POST /login
```

`JwtInterceptor,SignatureInterceptor` 这种写法里两个名字都被注册进这条路由的链；一旦在逗号后加了空格，行会被 `appendMap` 按空格切成四列，第三列变成 `JwtInterceptor,`、第四列被丢掉。按逗号切开后末尾多出一个空名字，加载时报 `no such component: , forget to register?`。

对应的组件注册和全局拦截器（`SetCommonInterceptor` 装的那种也算）：

```go
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	s.RegisterComponent("SessionInterceptor", &SessionInterceptor{})
	s.RegisterComponent("JwtInterceptor", &JwtInterceptor{})
	s.RegisterComponent("SignatureInterceptor", &SignatureInterceptor{})
	s.RegisterComponent("ApiKeyInterceptor", &ApiKeyInterceptor{})
	s.RegisterComponent("InternalOnlyInterceptor", &InternalOnlyInterceptor{})

	// 两个名字指同一个实例：审计反查名字用的是值相等
	admin := &AdminInterceptor{}
	s.RegisterComponent("AdminInterceptor", admin)
	s.Components.SetCommonInterceptor(admin)
	return nil
}
```

`AdminInterceptor` 没有被任何一条 `interceptor` 行引用，它靠全局链满足审计：`SetCommonInterceptor(admin)` 让它跑在每条路由前面，`namesOf` 又能把它反查成 `AdminInterceptor`，所以它属于 `auth.interceptors`，除了两条 public 路由，其余每条都算已鉴权。

如果确实要让一条路径模式覆盖所有方法、所有路径，就显式写通配模式和所有方法：

```yaml
interceptor:
  - GET,POST,PUT,PATCH,DELETE /* TraceInterceptor
```

同一路由被多条拦截器声明命中时，审计会打一条 warning，提醒你只有第一条会跑：

```text
route audit: GET /users/{id:[0-9]+} is matched by 2 interceptor declarations [[SessionInterceptor] [TraceInterceptor]], only the first one runs
```

### 加了一条路由、忘了写 interceptor 行

在上面那份配置后面加一行：

```yaml
urlmapping:
  # ... 已有路由 ...
  - POST /refunds RefundService Refund
```

但没有为它加任何 `interceptor` 行。它的路由链是空的；如果服务这边也没装全局拦截器，那么：

- 启动时：`panicIf(auditRoutes(...))` 让进程起不来，错误里列着

  ```text
  POST /refunds -> RefundService.Refund: it declares no interceptor and the server has no common interceptor
  ```

- 运行中改配置：该次重载被拒绝，日志里是 `turbo: configuration reload failed, keeping the running configuration: ...`，旧路由表继续服务。

如果服务已经用 `SetCommonInterceptor` 装了全局拦截器，`reason` 会换成另一种写法，把两边的链都列出来：

```text
POST /refunds -> RefundService.Refund: neither its route interceptors [] nor the common ones [AdminInterceptor] declare an auth interceptor
```

正确的补法有两种：为它加一条 `interceptor` 行并在 `auth.interceptors` 里包含那个名字；或者确认它本来就该匿名访问，把它加到 `auth.public_routes`：

```yaml
auth:
  public_routes:
    - POST /refunds
```

## 相关阅读

- [03-service-yaml.md](03-service-yaml.md)
- [04-routing.md](04-routing.md)
- [05-components.md](05-components.md)
- [06-interceptor.md](06-interceptor.md)
- [10-errors.md](10-errors.md)
- [14-logging.md](14-logging.md)
- [15-hot-reload.md](15-hot-reload.md)
- [19-troubleshooting.md](19-troubleshooting.md)
- [appendix-cheatsheet.md](appendix-cheatsheet.md)
