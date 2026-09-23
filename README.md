# Turbo  [![Coverage Status](https://coveralls.io/repos/github/vaporz/turbo/badge.svg?branch=master)](https://coveralls.io/github/vaporz/turbo?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/vaporz/turbo)](https://goreportcard.com/report/github.com/vaporz/turbo) [![codebeat badge](https://codebeat.co/badges/7a166e48-dae1-454c-b925-4fbcd3f1f461)](https://codebeat.co/projects/github-com-vaporz-turbo-master)

最新版本 | Latest Release: 0.6.2

文档 | Documentation: [docs/](docs/README.md) —— 完整中文教程（安装、配置、组件、绑定、热重载、部署、排查）

旧版英文文档 | Legacy English documentation: https://vaporz.github.io （停留在 v0.5.x，与 `docs/` 冲突时以 `docs/` 为准）

-------------------------

I'm very happy and ready to help you if you're intersted in Turbo, and want to try it.<br>
Please create an issue if you have encountered any problems or have any new ideas. Thank you!<br>
如果你对Turbo感兴趣，并想试一试，我非常乐意帮助你。<br>
如遇到任何问题，或有新主意，请开issue，谢谢！<br>
![](https://github.com/vaporz/turbo/blob/image/Turbo.gif)(From movie "[Turbo](https://en.wikipedia.org/wiki/Turbo_(film))")

## Features
 * Turbo generates a reverse-proxy server which translates a HTTP request into a grpc/Thrift request.  
 **(In other words, now you have a grpc|thrift service? Turbo turns your grpc|thrift APIs into HTTP APIs!)**
 * Modify and reload [configuration](https://vaporz.github.io/master/en/config.html#config) file at runtime! Without restarting service.
 * Support gRPC and [Thrift](https://vaporz.github.io/master/en/thrift.html).
 * Support [RESTFUL JSON API](https://vaporz.github.io/master/en/json.html) ("application/json").
 * [Interceptor](https://vaporz.github.io/master/en/interceptor.html#interceptor).
 * [PreProcessor](https://vaporz.github.io/master/en/preprocessor.html#preprocessor) and [PostProcessor](https://vaporz.github.io/master/en/postprocessor.html#postprocessor): customizable URL-RPC mapping process.
 * [Hijacker](https://vaporz.github.io/master/en/hijacker.html#hijacker): Take over requests, do anything you want!
 * [Convertor](https://vaporz.github.io/master/en/convertor.html#convertor): Tell Turbo how to set a struct.
 * [Service Multiplexing](https://vaporz.github.io/master/en/multiplexing.html)
## Components are singletons, and what a reload covers

Two things bite people who come from other frameworks, so they are written down here.

**A component is one instance, shared by every request.** `RegisterComponent` keeps
one value per name and turbo calls it concurrently, so a field written in `Before`
and read in `After` belongs to whichever requests overlap. Keep per request state in
the request context instead.

**A configuration change does not restart everything.** Reloading a `service.yaml`
takes effect for:

| Reloaded | Needs a restart |
|---|---|
| `urlmapping` (the routes) | `http_port` |
| `interceptor` / `preprocessor` / `postprocessor` / `hijacker` / `convertor` / `errorhandler` | `grpc_service_port`, `thrift_service_port` |
| `filter_proto_json` and its sub options | `environment`, `turbo_log_path`, `log_level` |
| `json_field_names`, `auth` | `file_root_path`, `package_path` (code generation only) |

`config.log_level` picks the level of turbo's own log (`panic`, `fatal`, `error`,
`warn`, `info`, `debug` or `trace`). Without it the environment decides: `production`
logs at `info`, anything else at `debug`. It is separate from a service's own
top level `log_level`, and it does not change where the log is written or in which
format.

A reload that cannot be loaded is refused: at startup it stops the server, and while
running it is logged and the previous configuration keeps serving. A route audit
reports every route and the interceptors behind it, and refuses a configuration that
would leave one of them without a declared auth interceptor (see `auth.interceptors`
and `auth.public_routes`).

## Index

完整教程在 [`docs/`](docs/README.md)，建议按顺序读；只想先跑起来就看
[docs/02-getting-started.md](docs/02-getting-started.md)。

 * [总览：Turbo 是什么、一次请求的生命周期](docs/01-overview.md)
 * [快速上手：安装、`turbo create`、第一个请求、加一个新 API](docs/02-getting-started.md)
 * [配置参考：`service.yaml` 逐键说明](docs/03-service-yaml.md)
 * [路由与 urlmapping](docs/04-routing.md)
 * [组件总览与执行顺序](docs/05-components.md)
 * [拦截器](docs/06-interceptor.md) · [前后处理器](docs/07-preprocessor-postprocessor.md) · [劫持器](docs/08-hijacker.md) · [转换器](docs/09-convertor.md)
 * [错误与状态码](docs/10-errors.md)
 * [参数绑定：来源与优先级、`InjectParam`](docs/11-binding.md)
 * [CLI 与代码生成](docs/12-code-generation.md)
 * [gRPC 与 Thrift](docs/13-grpc-thrift.md)
 * [日志](docs/14-logging.md) · [热重载](docs/15-hot-reload.md) · [鉴权与路由审计](docs/16-auth-and-route-audit.md)
 * [测试自己的服务](docs/17-testing.md) · [部署与运维](docs/18-deployment.md) · [排查手册](docs/19-troubleshooting.md)
 * [版本与行为变更（升级前必读）](docs/20-migration.md)
 * 附录：[配置全例](docs/appendix-config-example.md) · [速查表](docs/appendix-cheatsheet.md) · [API 索引](docs/appendix-api-index.md)

## Known Issues

写 `docs/` 时对照 **v0.6.2** 源码发现、**尚未修**的四处。四条都能在源码里核实，欢迎开 issue 或 PR。
每一条都给了当下的规避方式。

1. **Thrift 多服务客户端只生成一个 key。** `creator.go` 的 thrift 组件模板取的是
   `GrpcServiceNames()[0]`，所以 `config.thrift_service_name` 写成 `A,B` 时，生成的
   `thriftapi/component/components.go` 里 `ThriftClient` 只有 `A` 一个键，其余服务在网关侧
   拿不到客户端。规避：照 `test/testservice/thriftapi/component/components.go` 的写法，把其余
   服务名手写进那个 map。

2. **`Convertor` 对 `application/json` 请求不生效。** `BuildRequest`（`runtime.go`）的 JSON 分支
   不查 convertor，只有表单 / query / path 的绑定路径（`BuildStructErr`、`BuildArgs`）会用它，
   所以为某个结构体注册了 convertor 之后，用 JSON body 调同一个方法不会有任何效果。
   规避：需要 convertor 的接口不要用 JSON body 调用。完整说明见
   [docs/09-convertor.md](docs/09-convertor.md)。

3. **`errorhandler:` 写成 YAML 列表会被静默忽略。** 这一项必须是标量
   （`errorhandler: MyHandler`）；写成列表时 `Config.ErrorHandler()` 取到空串，于是悄悄用默认
   handler —— 既不报错也不警告。确认方式：启动日志里应该出现 `errorhandler: <name>` 这一行，
   没有就是没生效。

4. **`turbo.Errorf` 在 RPC 方法实现里不生效。** HTTP 层与实现之间是一次真实 RPC，而 RPC 会把
   error 序列化成 status，turbo 的 `statusError` 过不去那一跳 ⇒ `StatusOf(err)` 返回 0 ⇒
   一律按 500 回答（实测于 gRPC 链路，2026-09-23）。规避：impl 里用响应消息自己的 `code`/`msg`
   表达业务错误，需要真实 HTTP 状态码的判定放在拦截器里。见
   [docs/10-errors.md](docs/10-errors.md)。

5. **`SetConvertor` 的注释与实际签名不一致。** 注释写的是
   `usage: SetConvertor(new(SomeInterface), convertorFunc)`，实际签名是
   `SetConvertor(field string, convertorFunc Convertor)`，按**类型名**注册。以代码为准。

## Requirements
Golang version: >= 1.27.1 

Thrift version: 0.19.0  

### Building where downloading a toolchain is not possible

The `go` directive in `go.mod` is the minimum toolchain, and `GOTOOLCHAIN` defaults to
`auto`: a build machine whose Go is older than that directive tries to **download** the
required toolchain, which fails on an isolated network. Install Go 1.27.1 or newer on such
a machine, or set `GOTOOLCHAIN=local` and make sure the toolchain that is installed is new
enough.

### Code generation needs the legacy protoc-gen-go first in PATH

`turbo generate -r grpc` asks protoc for `--go_out=plugins=grpc:...`. The legacy
`protoc-gen-go` understands that option (up to v1.26.x); the modern one refuses it and
prints:

    --go_out: protoc-gen-go: plugins are not supported; use 'protoc --go-grpc_out=...'

Both are often installed at once, and protoc runs the **first one in PATH**, so put the
legacy plugin in front:

    go install github.com/golang/protobuf/protoc-gen-go@v1.5.1
    export PATH="$GOPATH/bin:$PATH"

`protoc-gen-go --version` tells the two apart: the legacy one answers "this program
should be run by protoc", the modern one prints "protoc-gen-go v1.x.y". turbo warns when
it finds a modern one rather than refusing to generate: whether the option is accepted
depends on the plugin build, not on the version alone. When protoc does refuse the option,
turbo adds the fix above to the error it reports.

Regenerating the test fixture also needs a GOPATH-shaped `file_root_path`; the comment at
the top of `test/testservice/service.yaml` has the exact recipe.

### Keeping go.mod tidy

    make tidy

runs `go mod tidy` after removing `test/testcreateservice`, which the integration tests
generate and leave behind. That directory is part of this module, so `go mod tidy` counts
what it imports: run tidy by hand and a dependency that only the generated tree uses is
recorded as direct -- a different answer from the one a fresh checkout gives.

### Checking dependencies for known vulnerabilities

    make vuln

runs [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck), which reports
the known vulnerabilities this code can actually reach, and separates them from the ones
that merely sit in the dependency list. It downloads the vulnerability database, so it
needs the network and is kept out of `make test`; it exits non-zero when it finds
something, so it also works as a check. Use govulncheck v1.8.0 or later: older
releases carry their own copy of `x/tools`, which cannot read the standard library of
Go 1.27 and panics instead of reporting.
