# 部署与运维

这篇讲把一个 turbo 服务放到服务器上要准备什么：Go 版本、二进制和 `service.yaml` 怎么放、改配置的正确姿势、探活与日志、进程怎么优雅停止，以及 systemd、容器和依赖版本这几件事。

> **版本**：v0.6.2 起，修改配置文件后 turbo 会先等文件内容稳定再读取，并拒绝 `urlmapping` 为空的配置；`config.log_level` 可以单独指定 turbo 自己的日志级别。升级部署前建议先看 [20-migration.md](20-migration.md)。

## Go 版本与离线构建机

`go.mod` 第一行是：

```text
go 1.27.1
```

README 的 Requirements 也写着 `Golang version: >= 1.27.1`，Thrift 版本要求是 `0.19.0`。

这里的 `go` 指令是最低工具链版本，而 `GOTOOLCHAIN` 默认是 `auto`：构建机上的 Go 比这个指令旧时，`go build` 会尝试**下载**指定版本的工具链。内网或隔离环境下载不了，构建就会失败。README 给了两条路：

- 在构建机上安装 Go 1.27.1 或更新版本；
- 或者设置 `GOTOOLCHAIN=local`，前提是本地装的工具链已经够新。

容器镜像里如果固定了 `GOTOOLCHAIN=local` 又用了一个旧的基础镜像，报错会出现在 `go build` 一开始，而不是编译到一半。

## 部署形态：二进制加一份 service.yaml

turbo 是一个库，不是一个独立进程。你部署的是自己那个 `main` 编译出来的二进制，配置文件的路径由你传给 `turbo.NewGrpcServer` / `turbo.NewThriftServer` 的第二个参数决定。

需要澄清一点：**turbo 自己不解析命令行参数**，`turbo create` 生成的 `main.go` 里配置路径是写死的。所谓 `-c` 是服务自己定义的约定，不是 turbo 的功能。想支持 `-c`，自己在 `main` 里加：

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/vaporz/turbo"
	"github.com/your/module/gen"
	"github.com/your/module/grpcapi/component"
	"github.com/your/module/grpcservice/impl"
)

var configPath = flag.String("c", "service.yaml", "path to service.yaml")

func main() {
	flag.Parse()

	s := turbo.NewGrpcServer(&component.ServiceInitializer{}, *configPath)
	s.Start(component.GrpcClient, gen.GrpcSwitcher, impl.RegisterServer)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-exit

	fmt.Println("Service is stopping...")
	s.Stop()
	fmt.Println("Service stopped")
}
```

`service.yaml` 放哪都行，只要进程读得到。常见做法是二进制同目录一份，或者 `/etc/<service>/service.yaml` 并用 `-c` 指过去。`turbo create` 生成的形态是把配置文件放在服务根目录，`main.go` 里写绝对路径指过去。

### 两个路径键

| 键 | 要求 | 用在哪 | 注意 |
|---|---|---|---|
| `file_root_path` | 必须是绝对路径，否则 `FileRootPath()` panic：`fileRootPath MUST be an absolute path, got: ...` | 只有 `turbo create` / `turbo generate` 用 | 它会被拼上 `package_path` 组成服务根目录（`ServiceRootPath`），所以要指到能容纳 `package_path` 的目录 |
| `turbo_log_path` | 可以是相对路径 | `production` 环境写日志用 | 相对路径按**进程当前工作目录**解析，不是可执行文件所在目录 |

`turbo_log_path` 那条尤其容易踩。`log.go` 里源码自己的 TODO 就写着：`os.Getwd()` 返回的不是 exe 文件所在的目录，而是执行文件时所处的目录。systemd 如果不设 `WorkingDirectory`，工作目录很可能是 `/`；容器里则可能是镜像的 `WORKDIR`。所以生产环境请写绝对路径：

```yaml
config:
  environment: production
  turbo_log_path: /var/log/yourservice
```

运行时日志文件固定叫 `turbo.log`，以追加方式打开。`file_root_path` 不在运行期使用，一份只用于部署的配置可以不带它；但要跑代码生成就必须带上，而且必须是绝对路径。

## 改配置的正确姿势

turbo 支持热重载，但前提是这次改动“读得出来、装得上”。v0.6.2 起有两层保护：

1. 写完触发事件之后，turbo 先调用 `waitForStableFile`：每 20 毫秒看一次文件，连续两次读到相同内容且间隔达到 100 毫秒才算稳定，最多等 3 秒。两次内容相同但为空不算稳定，因为写文件的人可能还没写出第一批字节。
2. 读出来的配置如果 `urlmapping` 为空，会被拒绝，错误文本是 `urlmapping is empty, so no route would be served at all (a configuration file read while it is being written looks like this)`。这条校验就是为“先截断再写”的写法定制的。

所以推荐的操作是：

- 用一次写入完成整份文件的替换，不要在一次改写里分几段追加。
- 改完立刻打一条健康路由确认，例如 `curl -sS http://127.0.0.1:8080/hello?your_name=ping`。
- 看日志确认：成功是 `Configuration reloaded`，失败是 `turbo: configuration reload failed, keeping the running configuration: ...`（读得出但装不上）或 `turbo: ignoring configuration change, it cannot be loaded: ...`（读都读不了）。

如果你习惯写临时文件再 `mv` 覆盖，要注意 viper 监视的是配置文件本身，`mv` 之后事件是否还能收到，取决于底层的 fsnotify 实现，这一点在 turbo 源码里看不出来，属于**未验证**；用一次 `os.WriteFile` / `cp` 原地覆盖是源码明确支持并处理了的路径。

不是所有键都能靠重载生效。启动时 `watchConfigReload` 会打印一行提示，把它说清楚：

```text
turbo: a configuration change reloads urlmapping, components and filter_proto_json; http_port, grpc_service_port, thrift_service_port, environment, turbo_log_path and log_level need a restart
```

改端口、改 `environment`、改日志路径或级别，都要重启进程。

## 探活

turbo **不自带**健康检查接口，HTTP 上有哪些路径完全由 `urlmapping` 决定。`turbo create` 生成的 `service.yaml` 里默认有 `GET /hello`，很多服务就直接拿它探活。

如果想加一个不经过后端的 `/healthz`，可以给它声明一条路由，再挂一个 hijacker。hijacker 在 RPC 之前执行，所以后端不可用时它也能应答：

```yaml
urlmapping:
  - GET /healthz YourService SayHello

hijacker:
  - GET /healthz healthzHijacker
```

```go
s.RegisterComponent("healthzHijacker", turbo.Hijacker(
	func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		resp.Write([]byte("ok"))
	},
))
```

`RegisterComponent` 要在 `StartHTTPServer`（或 `Start`）之前调用。如果配置里声明了 `auth.interceptors`，别忘了把这条路由写进 `auth.public_routes`，否则路由审计会拒绝启动（见 [16-auth-and-route-audit.md](16-auth-and-route-audit.md)）。

## 日志与多实例

- `config.turbo_log_path` 决定目录，文件名固定 `turbo.log`，追加写入。
- `config.log_level` 决定 turbo 自己的级别；`production` 默认 `info`，其它环境默认 `debug`。它在 `config` 段下面，和服务自己顶层的 `log_level` 没有关系。
- 一个目录跑多个实例时，它们会追加到同一个 `turbo.log`。要分开就每个实例给一个 `turbo_log_path`。
- 换机器部署前先在目标机上确认目录存在且可写；只有 `production` 才会尝试建目录（`os.MkdirAll(logPath, 0755)`），建不出来会 panic。

细节见 [14-logging.md](14-logging.md)。

## 进程管理与优雅停止

生成的 `main.go` 已经在监听信号，模板里的写法是 `signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)`（`os.Kill` 实际无法被捕获，能生效的是 `SIGINT`、`SIGTERM`、`SIGQUIT`），收到之后调用 `s.Stop()`。`Stop()` 做的事（`server.go` 的 `shutdown` 与 `stop`）：

1. `close(done)`：停掉热重载 goroutine，watcher 不再往上送变更。`stopOnce` 保证只关一次。
2. 调用注册的 `Initializer.StopService(s)`。
3. 有 HTTP server 时，用 5 秒超时的 context 调 `httpServer.Shutdown(ctx)`，然后打印 `Http Server stopped`。
4. 有 gRPC server 时，先关闭内部 gRPC 客户端，再 `grpcServer.GracefulStop()`，打印 `Grpc Server stopped`。
5. 有 Thrift server 时，关闭内部 Thrift 客户端，`thriftServer.Stop()`，打印 `Thrift Server stopped`。

`GrpcServer.Stop` 之前还会打印 `Stop() invoked, Service is stopping...`。注意 HTTP 的 5 秒超时只作用于 `Shutdown`，超过之后 `Shutdown` 返回错误但代码没有处理，进程会继续往下走。

## systemd 最小示例

```ini
[Unit]
Description=Your Turbo service
After=network-online.target
Wants=network-online.target

[Service]
User=yourservice
Group=yourservice
WorkingDirectory=/opt/yourservice
ExecStart=/opt/yourservice/yourservice -c /etc/yourservice/service.yaml
Environment=GOTOOLCHAIN=local
Restart=on-failure
RestartSec=2
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

要点：

- `ExecStart` 里的 `-c` 是上面那段 `main.go` 定义的，不是 turbo 的参数。
- `WorkingDirectory` 设成固定目录，即使 `turbo_log_path` 用了相对路径也不会飘。
- systemd 停止服务时发 `SIGTERM`，走的是 `Stop()` 那条路，可以在 `systemctl stop` 后从日志确认三个 `... stopped` 行。

容器里部署时，配置通常来自挂载卷或 ConfigMap。挂载卷里的文件被替换通常不会在容器内产生 inotify 事件（这是挂载机制的一般行为，不是 turbo 源码里的结论），所以热重载在容器里经常不生效，需要重启 Pod 让新配置生效。这是运维层面的取舍，不是 turbo 的缺陷。

## 依赖版本

turbo 的 `go.mod` 里 gRPC 和 Thrift 都是直接依赖：

```text
github.com/apache/thrift v0.19.0
google.golang.org/grpc v1.58.3
google.golang.org/protobuf v1.34.1
```

生成的代码会直接 `import` Thrift 和 protobuf（例如 gRPC 的 `impl` 包里用 `google.golang.org/grpc`，Thrift 的 `impl` 包里用 `github.com/apache/thrift/lib/go/thrift`），所以你的模块最终也会解析到这些版本。部署和升级时注意：

- README 把 Thrift 的要求写成 `0.19.0`，如果你的服务自己 `require` 了别的版本，冲突会在 `go build` 时暴露。
- 升级 turbo 可能把你的 gRPC / protobuf 版本一起带上去。先在 `go.mod` 里看清解析结果，再重新生成代码：

```bash
go list -m all | grep -E 'grpc|thrift|protobuf'
turbo generate github.com/your/module -r grpc -I /abs/path/to/protos
make test
```

- gRPC 调用选项走 `turbo.CallOptions`（默认每次调用带 `grpc.Header` / `grpc.Trailer` / `grpc.Peer`，不设置超时）。它是公开变量，可以覆盖来加 `grpc.WaitForReady` 之类的选项；但调用用的 context 来自请求本身，超时要通过替换请求 context 来做，见 [19-troubleshooting.md](19-troubleshooting.md)。

## 相关阅读

- [02-getting-started.md](02-getting-started.md)：从零搭一个服务
- [03-service-yaml.md](03-service-yaml.md)：`config` 段全部配置键
- [15-hot-reload.md](15-hot-reload.md)：热重载覆盖哪些键
- [14-logging.md](14-logging.md)：日志路径与级别
- [13-grpc-thrift.md](13-grpc-thrift.md)：后端服务与依赖
- [20-migration.md](20-migration.md)：升级前的检查清单
