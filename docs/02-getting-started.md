# 快速上手

这篇从零开始：装好工具 → 用 `turbo create` 生成一个可运行的项目 → 跑通第一个 HTTP 请求 → 加一个新 API。
照着敲一遍大约 20 分钟，之后再看 [03-service-yaml.md](03-service-yaml.md) 与 [04-routing.md](04-routing.md) 就顺了。

## 1. 准备环境

| 需要 | 说明 |
|---|---|
| Go | **≥ 1.27.1**（`turbo` 自身 `go.mod` 的 `go` 指令即最低要求） |
| `protoc` | 用 gRPC 时必须；`protoc --version` 能打印版本即可 |
| `protoc-gen-go` | 必须是**能接受 `--go_out=plugins=grpc` 的旧版插件**（v1.5.1 是最省事的选择），见第 2 步 |
| `protoc-gen-buildfields` | Turbo 自己的 protoc 插件，生成 `gen/grpcfields.yaml` |
| `thrift` 编译器 | 只有用 Thrift 时才需要（当前教程以 gRPC 为例） |

## 2. 安装 CLI

```bash
go install github.com/vaporz/turbo/turbo@latest
go install github.com/vaporz/turbo/protoc-gen-buildfields@latest

# gRPC 生成需要旧版 protoc-gen-go：新版插件已经不支持 plugins=grpc
go install github.com/golang/protobuf/protoc-gen-go@v1.5.1

export PATH="$(go env GOPATH)/bin:$PATH"
turbo --version    # 例如 turbo version v0.6.2
```

> **为什么必须钉住旧版 `protoc-gen-go`**：Turbo 生成 gRPC 代码时用的是
> `--go_out=plugins=grpc:...`（把 `*.pb.go` 与 gRPC 桩一次生成）。这个选项在 legacy 插件（到 v1.26.x 为止）
> 里可用，新版 `google.golang.org/protobuf/cmd/protoc-gen-go` 会直接拒绝。
> 两个插件常常同时装在机器上，而 protoc 只会用 **PATH 里第一个**，所以要么把 legacy 放在前面，
> 要么删掉另一个。`protoc-gen-go --version` 可以区分它们：legacy 会回答
> "this program should be run by protoc"，新版会打印 `protoc-gen-go v1.x.y`。

## 3. 创建项目

```bash
mkdir -p ~/work && cd ~/work
turbo create example.com/hello Hello -r grpc -p .
```

| 参数 | 含义 |
|---|---|
| `package_path`（第一个位置参数） | 包的导入路径，同时是**目录名**：`-p` 之下会创建它。它必须与你之后 `go mod init` 的模块路径一致 |
| `ServiceName`（第二个位置参数） | 服务名，必须是 CamelCase（`Hello`、`ImageToy`；`hello` 会被拒绝） |
| `-r, --rpctype` | `grpc`（默认）或 `thrift` |
| `-p, --rootpath` | 新包创建在哪个目录下，默认 `.`（当前目录） |
| `-f, --force` | 目录已存在时不做交互确认，直接覆盖 |

如果目标目录已经存在，`turbo create` 会**交互式**问你是否删除（输入 `y` 两次）；脚本里请用 `-f`。

### 生成出来的东西

```
example.com/hello/
├── service.yaml                 # Turbo 的配置：路由、组件、端口、日志
├── hello.proto                  # 你的接口定义（文件名是服务名的小写）
├── main.go                      # 单进程入口：RPC 服务 + HTTP 网关
├── gen/
│   ├── grpcswitcher.go          # Turbo 生成：HTTP 请求 → RPC 调用的分派代码
│   ├── grpcfields.yaml          # protoc-gen-buildfields 生成：请求消息的字段结构
│   └── proto/
│       └── hello.pb.go          # protoc + legacy protoc-gen-go 生成
├── grpcapi/
│   ├── helloapi.go              # 只启动 HTTP 网关的入口
│   └── component/components.go  # 组件注册处 + GrpcClient
└── grpcservice/
    ├── hello.go                 # 只启动 RPC 服务的入口
    └── impl/helloimpl.go        # 服务实现 —— 你要写业务的地方
```

生成的 `service.yaml` 长这样（已去掉注释）：

```yaml
config:
  environment: development
  file_root_path: /home/you/work
  package_path: example.com/hello
  turbo_log_path:
  http_port: 8081
  grpc_service_name: Hello
  grpc_service_host: 127.0.0.1
  grpc_service_port: 50061
  thrift_service_name: Hello
  thrift_service_host: 127.0.0.1
  thrift_service_port: 50062

urlmapping:
  - GET /hello Hello SayHello
```

`file_root_path` 被写成**绝对路径**（`filepath.Abs` 的结果），它与 `package_path` 拼起来就是项目根目录。
换机器或换目录后记得同步改这一项。

## 4. 让它编译起来

`turbo create` **不生成 `go.mod`**（它不知道你想用什么模块路径），所以第一次要自己初始化：

```bash
cd example.com/hello
go mod init example.com/hello    # 必须与 service.yaml 的 package_path 一致
go mod tidy                      # 拉取 turbo / grpc / protobuf 等依赖
go build ./...
```

## 5. 跑起来

```bash
go run .
```

开发环境（`environment: development`）下，Turbo 的日志是文本格式、写到 stderr、级别 `debug`，
启动大致会看到：

```
Starting Turbo...
Starting GRPC Service...
GRPC Service started
Starting HTTP Server...
[grpc]connecting addr:127.0.0.1:50061
route audit: 1 route(s), 0 authenticated, 0 public, 0 unprotected (no auth.interceptors declared, so the audit only reports)
route: GET /hello -> Hello.SayHello
turbo: 1 route(s) registered
HTTP Server started
```

最后那行是**启动路由审计**的汇总（详见 [16-auth-and-route-audit.md](16-auth-and-route-audit.md)）；
逐条路由的审计明细在 `debug` 级别下才会打印。日志相关的所有开关见 [14-logging.md](14-logging.md)。

## 6. 第一个请求

```bash
curl 'http://127.0.0.1:8081/hello?yourName=world'
```

```
{"message":"[grpc server]Hello, world"}
```

这个请求完整地走了一遍：mux 路由 `/hello` → 组装拦截器（这里是空的）→ 绑定参数（proto 里的
`yourName` 字段从 query 取得）→ 调用 `Hello.SayHello` → 把 `SayHelloResponse` 序列化成 JSON 写回。
参数绑定的完整规则在 [11-binding.md](11-binding.md)。

试几个等价写法（参数名拼写不敏感）：

```bash
curl 'http://127.0.0.1:8081/hello?your_name=world'
curl 'http://127.0.0.1:8081/hello?YOURNAME=world'
```

## 7. 加一个新 API

假设要加一个 `Echo`：把请求里的文本原样返回。

**① 改 proto**（`hello.proto`）：

```proto
message EchoRequest {
    string text = 1;
}

message EchoResponse {
    string text = 1;
}

service Hello {
    rpc sayHello (SayHelloRequest) returns (SayHelloResponse) {}
    rpc echo (EchoRequest) returns (EchoResponse) {}
}
```

**② 重新生成**（在项目根目录，`-I` 传**包含 `service.yaml` 与 `.proto` 的目录**，建议绝对路径）：

```bash
turbo generate example.com/hello -r grpc -I "$(pwd)"
```

它会重新跑 protoc 生成 `gen/proto/hello.pb.go`、`gen/grpcfields.yaml`，并重写 `gen/grpcswitcher.go`。
生成是**原子**的：中途失败不会把已有产物写坏。

**③ 实现方法**（`grpcservice/impl/helloimpl.go`）：

```go
func (s *Hello) Echo(ctx context.Context, req *proto.EchoRequest) (*proto.EchoResponse, error) {
	return &proto.EchoResponse{Text: req.Text}, nil
}
```

**④ 加一条路由**（`service.yaml` 的 `urlmapping`）：

```yaml
urlmapping:
  - GET /hello Hello SayHello
  - GET,POST /echo Hello Echo
```

**⑤ 重新编译并重启**：

```bash
go run .
curl -X POST -H 'Content-Type: application/json' -d '{"text":"hi"}' http://127.0.0.1:8081/echo
# {"text":"hi"}
```

> **为什么这次不能只靠热重载**：Turbo 的热重载能换掉**路由表**（`urlmapping`）与组件，但
> `gen/grpcswitcher.go` 是**编译进二进制**的代码。新增一个 RPC 方法必须重新生成、重新编译、重启；
> 给**已存在**的方法加一条新路由则只需改 `urlmapping`（保存即生效）。
> 详见 [15-hot-reload.md](15-hot-reload.md)。

## 8. 三个入口怎么选

| 入口 | 起什么 | 命令 |
|---|---|---|
| `main.go` | RPC 服务 + HTTP 网关（同一进程，最常用） | `go run .` |
| `grpcservice/hello.go` | 只起 RPC 服务（HTTP 网关在别处部署） | `go run ./grpcservice` |
| `grpcapi/helloapi.go` | 只起 HTTP 网关（连到别处的 RPC 服务） | `go run ./grpcapi` |

三种都读同一份 `service.yaml`；分开部署时把 `grpc_service_host` / `grpc_service_port`
指到真正的服务地址即可。

## 9. 第一次容易踩的坑

| 现象 | 原因 | 处理 |
|---|---|---|
| `turbo create` 生成的 `*.pb.go` 缺失或报 `plugins are not supported` | PATH 里第一个 `protoc-gen-go` 是新版 | 按第 2 步装 legacy 插件并放到 PATH 前面 |
| 启动 panic：`file_root_path` MUST be an absolute path | 手改配置时写成了相对路径 | 改成绝对路径 |
| 启动 panic：`urlmapping is empty, so no route would be served at all` | 配置里没有（或没读到）`urlmapping` | 补上路由；若刚才是"保存到一半"，见 [15-hot-reload.md](15-hot-reload.md) |
| 启动 panic：`no such component: XXX, forget to register?` | `service.yaml` 写了一个没在 `InitService` 里注册的组件名 | 在 `grpcapi/component/components.go` 的 `InitService` 里 `s.RegisterComponent("XXX", …)` |
| `go run ./grpcapi` 连不上 RPC | 只起了网关，RPC 服务没在 `grpc_service_host:port` 上 | 先起 `./grpcservice` 或 `main.go` |
| 改完配置没生效 | 改的键不参与热重载（端口、`environment`、`log_level` 等） | 见 [15-hot-reload.md](15-hot-reload.md) 的对照表 |

更多症状见 [19-troubleshooting.md](19-troubleshooting.md)。

## 相关阅读

- [03-service-yaml.md](03-service-yaml.md) —— 配置里还有哪些键
- [04-routing.md](04-routing.md) —— 路径参数、`/*`、多方法
- [11-binding.md](11-binding.md) —— 参数从哪来、谁优先
- [12-code-generation.md](12-code-generation.md) —— `turbo create` / `turbo generate` 的全部细节
