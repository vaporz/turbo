# Turbo  [![Coverage Status](https://coveralls.io/repos/github/vaporz/turbo/badge.svg?branch=master)](https://coveralls.io/github/vaporz/turbo?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/vaporz/turbo)](https://goreportcard.com/report/github.com/vaporz/turbo) [![codebeat badge](https://codebeat.co/badges/7a166e48-dae1-454c-b925-4fbcd3f1f461)](https://codebeat.co/projects/github-com-vaporz-turbo-master)

最新版本 | Latest Release: 0.6.1

文档地址 | Documentation: https://vaporz.github.io

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
 * [Create a service on the fly](https://vaporz.github.io/master/en/create.html)
 * [Command line tools](https://vaporz.github.io/master/en/command.html)
 * [Rules and Conventions](https://vaporz.github.io/master/en/rules.html)
 * [How to add a new API](https://vaporz.github.io/master/en/add.html)
 * [Use a shared struct](https://vaporz.github.io/master/en/shared.html)
 * [Support RESTFUL JSON API](https://vaporz.github.io/master/en/json.html)
 * [Interceptor](https://vaporz.github.io/master/en/interceptor.html)
 * [PreProcessor](https://vaporz.github.io/master/en/preprocessor.html#preprocessor) and [PostProcessor](https://vaporz.github.io/master/en/postprocessor.html#postprocessor)
 * [Hijacker](https://vaporz.github.io/master/en/hijacker.html#hijacker)
 * [Convertor](https://vaporz.github.io/master/en/convertor.html#convertor)
 * [Error Handler](https://vaporz.github.io/master/en/errorhandler.html)
 * [Thrift support](https://vaporz.github.io/master/en/thrift.html)
 * [Configs in service.yaml](https://vaporz.github.io/master/en/config.html#config)
 * [Service Multiplexing](https://vaporz.github.io/master/en/multiplexing.html)
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
