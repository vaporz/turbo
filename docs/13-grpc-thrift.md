# gRPC 与 Thrift 两条链路

这篇讲 turbo 支持的两种后端：gRPC 与 Thrift 各自的启动入口、客户端 map 与配置里 service name 的对应关系、只属于其中一条链路的能力（gRPC 的 metadata/peer，Thrift 的 processor 与参数列表），以及几条最容易踩的坑。

## 两种 server 的启动 API

两个构造函数形状一样，第二个参数都是 `service.yaml` 的路径，第一个参数为 nil 时使用 `defaultInitializer`（什么都不做）：

```go
func NewGrpcServer(initializer Initializable, configFilePath string) *GrpcServer
func NewThriftServer(initializer Initializable, configFilePath string) *ThriftServer
```

`Initializable` 有两个方法：`InitService(s Servable) error` 在服务启动前跑（注册组件、装公共拦截器），`StopService(s Servable)` 在两个 server 都停下之后跑。

### 四种启动组合与顺序

| 方法 | 做什么 |
| --- | --- |
| `Start(clientCreator, sw, registerServer)` | 起 RPC 服务，再起 HTTP 网关，最后开启配置热重载 |
| `StartHTTPServer(clientCreator, sw)` | 只起 HTTP 网关（并向 RPC 服务端建连接），开启热重载 |
| `StartGrpcService(registerServer)` / `StartThriftService(registerTProcessor)` | 只起 RPC 服务，不建客户端连接、不开热重载 |

`Start` 内部顺序（`grpcserver.go`）：`Initializer.InitService`、`startGrpcServiceInternal`、`startGrpcHTTPServerInternal`、`watchConfigReload`。Thrift 的 `Start` 在两步之间多一次等待：

```go
s.thriftServer = s.startThriftServiceInternal(registerTProcessor, false)
time.Sleep(time.Second * 1)
s.httpServer = s.startThriftHTTPServerInternal(clientCreator, sw)
```

因为 Thrift 客户端在 `connect` 时会真的 `transport.Open()`，服务端还没监听就会 panic，这一秒是留给服务端起监听的时间。

两点要记住：`StartGrpcService` / `StartThriftService` 都不调用 `watchConfigReload`，所以只起 RPC 服务时 `service.yaml` 的改动不会热重载，热重载由 `Start` 或 `StartHTTPServer` 开启；想先起 RPC 再起网关时，顺序必须是先 `StartGrpcService` / `StartThriftService`，再 `StartHTTPServer`。

RPC 服务端本身的行为：

- gRPC：`net.Listen("tcp", ":"+grpc_service_port)`、`grpc.NewServer()`、`registerServer(grpcServer)`、`reflection.Register(grpcServer)`，然后在 goroutine 里 `Serve`，监听失败直接 panic。
- Thrift：`thrift.NewTServerSocket(":"+thrift_service_port)`，处理器是 `thrift.NewTMultiplexedProcessor()`，把 `registerTProcessor()` 返回的 map 逐个 `RegisterProcessor(name, p)`，传输用 `thrift.NewTSimpleServer4(processor, transport, thrift.NewTTransportFactory(), thrift.NewTBinaryProtocolFactoryDefault())`。
- HTTP 网关监听 `http_port`，两条链路共用 `startHTTPServer`。

`Start` 的第三个参数在两条链路上形状不同：gRPC 是 `func(s *grpc.Server)`，Thrift 是 `func() map[string]thrift.TProcessor`。

### main.go 的真实写法

`turbo create` 生成的根 `main.go`（grpc 版）核心就是这几行，thrift 版把中间一行换掉：

```go
func main() {
	s := turbo.NewGrpcServer(&gcomponent.ServiceInitializer{}, "/abs/path/to/service.yaml")
	s.Start(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	// thrift 版：
	// s := turbo.NewThriftServer(&tcomponent.ServiceInitializer{}, "/abs/path/to/service.yaml")
	// s.Start(tcomponent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}
	s.Stop()
	fmt.Println("Service stopped")
}
```

## 客户端 map 与 service_name 的关系

`GrpcClient` / `ThriftClient` 返回的 map 的 key，必须和配置里的 service name 完全一致。

```go
// grpcapi/component/components.go
func GrpcClient(conn *grpc.ClientConn) map[string]interface{} {
	return map[string]interface{}{
		"GreeterService": proto.NewGreeterServiceClient(conn),
	}
}

// thriftapi/component/components.go
func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) map[string]interface{} {
	iprot := f.GetProtocol(trans)
	return map[string]interface{}{
		"GreeterService": t.NewGreeterServiceClientProtocol(trans, iprot,
			thrift.NewTMultiplexedProtocol(iprot, "GreeterService")),
	}
}
```

```yaml
config:
  grpc_service_name: GreeterService
  grpc_service_host: 127.0.0.1
  grpc_service_port: 50061
  thrift_service_name: GreeterService
  thrift_service_host: 127.0.0.1
  thrift_service_port: 50062
```

连接建立发生在 `startGrpcHTTPServerInternal`：`grpc.Dial(host+":"+port, grpc.WithInsecure())`，然后 `clientCreator(conn)` 得到 map。`grpcClient.init` 在 map 已存在时直接返回，所以重复调用不会重建连接。Thrift 一侧是 `thrift.NewTSocket(hostPort)`、拿到 transport、`Open()`、`thrift.NewTBinaryProtocolFactoryDefault()`，再交给 `clientCreator(transport, factory)`。

`Service` 用名字取客户端：

```go
func (s *GrpcServer) Service(serviceName string) interface{}
func (s *ThriftServer) Service(serviceName string) interface{}
```

生成的 switcher 里就是这么用的：

```go
rpcResponse, err = s.Service("GreeterService").(g.GreeterServiceClient).SayHello(req.Context(), request, callOptions...)
```

因此 `Service` 返回的是 `interface{}`，调用方必须做类型断言，断言失败会 panic。连接还没建立时 `Service` 也会 panic，文案是 `grpc connection not initiated!` 或 `thrift connection not initiated!`。也就是说：只要用到 `Service`，就必须先让 `StartHTTPServer`（或 `Start`）跑过。

## gRPC 特有能力

### CallOptions、header、trailer、peer

turbo 在调用 RPC 之前调用一个可替换的包级变量：

```go
var CallOptions = func(serviceName, methodName string, req *http.Request) ([]grpc.CallOption, *metadata.MD, *metadata.MD, *peer.Peer) {
	header := new(metadata.MD)
	trailer := new(metadata.MD)
	peer := &peer.Peer{}
	return []grpc.CallOption{grpc.Header(header), grpc.Trailer(trailer), grpc.Peer(peer)}, header, trailer, peer
}
```

生成的 switcher 会展开它，并在 RPC 返回后把结果放进请求 context：

```go
callOptions, header, trailer, peer := turbo.CallOptions(serviceName, methodName, req)
// ... 真正发起 RPC ...
turbo.WithCallOptions(req, header, trailer, peer)
```

读取用三个函数（`runtime.go`）：

```go
func GrpcMetadataHeader(ctx context.Context) *metadata.MD
func GrpcMetadataTrailer(ctx context.Context) *metadata.MD
func GrpcMetadataPeer(ctx context.Context) *peer.Peer
```

因为它们是在 `WithCallOptions` 之后才可读，而 `WithCallOptions` 又在 RPC 返回之后执行，所以读取要放在拦截器的 `After` 里。服务端负责设置 header / trailer：

```go
package impl

import (
	"context"

	"github.com/example/greeterservice/gen/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func (s *GreeterService) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-request-id", "abc-123")); err != nil {
		return nil, err
	}
	if err := grpc.SetTrailer(ctx, metadata.Pairs("x-served-by", "greeter-1")); err != nil {
		return nil, err
	}
	return &proto.SayHelloResponse{Message: "Hello, " + req.YourName}, nil
}
```

网关侧在 `After` 里把三者读出来：

```go
package component

import (
	"log"
	"net/http"

	"github.com/vaporz/turbo"
)

type MetadataInterceptor struct {
	turbo.BaseInterceptor
}

func (i *MetadataInterceptor) After(resp http.ResponseWriter, req *http.Request) error {
	ctx := req.Context()
	header := *turbo.GrpcMetadataHeader(ctx)
	trailer := *turbo.GrpcMetadataTrailer(ctx)
	peer := turbo.GrpcMetadataPeer(ctx)

	log.Printf("x-request-id=%s", header["x-request-id"][0])
	log.Printf("x-served-by=%s", trailer["x-served-by"][0])
	log.Printf("peer=%s", peer.Addr.String())
	return nil
}
```

### 替换 CallOptions

`CallOptions` 是公开的包级变量，可以在 `init` 或启动前整个替换（签名必须一致）：

```go
func init() {
	turbo.CallOptions = func(serviceName, methodName string, req *http.Request) ([]grpc.CallOption, *metadata.MD, *metadata.MD, *peer.Peer) {
		header := new(metadata.MD)
		trailer := new(metadata.MD)
		p := &peer.Peer{}
		return []grpc.CallOption{
			grpc.Header(header),
			grpc.Trailer(trailer),
			grpc.Peer(p),
			grpc.MaxCallRecvMsgSize(8 * 1024 * 1024),
		}, header, trailer, p
	}
}
```

header/trailer/peer 这三个返回值即使不用也得原样返回，因为 `WithCallOptions` 会无条件把它们放进 context，`GrpcMetadataHeader` 等函数读取时做的是不带检查的类型断言。

## Thrift 特有细节

### TProcessor 与多路复用

`registerTProcessor` 返回 `map[string]thrift.TProcessor`，turbo 把它注册进 `thrift.NewTMultiplexedProcessor()`；客户端侧用 `thrift.NewTMultiplexedProtocol(iprot, "<service name>")` 包装，名字必须与服务端注册的名字一致。生成的实现骨架：

```go
func TProcessor() map[string]thrift.TProcessor {
	return map[string]thrift.TProcessor{
		"GreeterService": gen.NewGreeterServiceProcessor(GreeterService{}),
	}
}
```

### 参数绑定差异

Thrift 方法接收的是参数列表，不是一个请求消息，所以 JSON body 按参数名对应，规则在 `bindThriftArgsFromJSON`：多参数方法的 body 是一个 JSON 对象，key 是参数名（`{"yourName":"x","int64Value":7}`）；单参数方法的 body 就是那个参数本身，把参数名再包一层（`{"request":{...}}`）会被拒绝，并提示 `not fields of the argument Request`；参数名忽略大小写与下划线；命名了不存在的参数返回 400 并列出方法真实参数名；body 不是 JSON 对象（例如 `[1,2]`）被拒绝，空 body 表示没有命名任何参数。完整的来源优先级与示例见 11-binding.md。

### 版本要求

`README.md` 的 Requirements：Golang >= 1.27.1，Thrift 0.19.0。`go.mod` 里 `github.com/apache/thrift v0.19.0`、`google.golang.org/grpc v1.58.3`，`go` 指令为 1.27.1。构建机 Go 版本低于该指令且 `GOTOOLCHAIN=auto` 时会尝试下载工具链，隔离网络上会失败；要么装足够新的 Go，要么设 `GOTOOLCHAIN=local`。gRPC 代码生成用 `--go_out=plugins=grpc`，需要遗留的 `protoc-gen-go` 排在 `PATH` 前面，见 12-code-generation.md。

## 多服务复用

两条链路都用配置里的逗号分隔列表表达多服务。

```yaml
config:
  grpc_service_name: GreeterService,MinionsService
  thrift_service_name: GreeterService,MinionsService

urlmapping:
  - GET /hello GreeterService SayHello
  - POST /eat MinionsService Eat
```

`GrpcServiceNames()` / `ThriftServiceNames()` 按逗号切分。gRPC 侧客户端 map 写上两个客户端，共用同一个 `*grpc.ClientConn`；服务端 `RegisterServer` 里注册两个 service：

```go
func GrpcClient(conn *grpc.ClientConn) map[string]interface{} {
	return map[string]interface{}{
		"GreeterService": proto.NewGreeterServiceClient(conn),
		"MinionsService": proto.NewMinionsServiceClient(conn),
	}
}

func RegisterServer(s *grpc.Server) {
	proto.RegisterGreeterServiceServer(s, &GreeterService{})
	proto.RegisterMinionsServiceServer(s, &MinionsService{})
}
```

Thrift 侧 `TProcessor()` 返回两个 processor，服务端 `TMultiplexedProcessor` 按名字注册，客户端 map 里每个服务各自包一层名字一致的 `TMultiplexedProtocol`，共用同一个 transport。

一个与生成代码有关的注意点：Thrift 项目里 `gen/thrift/build.go` 的反射列表和 `thriftapi/component/components.go` 的客户端名字都取自 `grpc_service_name`（`generator.go` 与 `creator.go` 里传给模板的是 `GrpcServiceNames()`）。所以 Thrift 项目也要让 `grpc_service_name` 有值并与 `thrift_service_name` 一致，否则生成的 Thrift 代码会引用不存在的类型。

## 常见坑

- **`Service()` 在未连上时 panic。** `StartGrpcService` / `StartThriftService` 只起 RPC 服务，不建客户端连接；`Service` 只在 `StartHTTPServer` / `Start` 之后可用。
- **gRPC 的 Dial 不阻塞。** `grpc.Dial` 没有 `WithBlock`，返回成功不代表连上了，连接错误在第一次调用时以 RPC 错误的形式出现。
- **Thrift 客户端连接会立即 `Open()`。** 服务端没在监听就 panic，这也是 `Start` 里那一秒 sleep 的原因；自己分两步启动时要在 `StartHTTPServer` 之前确认 RPC 服务已经起来。
- **Thrift 的 JSON body 有格式约束。** 必须是 JSON 对象（多参数方法）或单个参数的值，不能是数组之类的其它 JSON 值；多余或拼错的 key 会被 400 拒绝，而不是忽略。
- **HTTP 请求上下文不会传进服务实现。** gRPC 只传 metadata，Thrift 生成的 switcher 直接传 `context.Background()`。服务端验证过的值要用 `turbo.InjectParam`，让绑定写进请求消息字段，见 11-binding.md。
- **`Service` 返回 `interface{}`。** 必须断言成生成的客户端接口，断言错了会 panic。
- **热重载只在网关侧开启后生效。** `StartGrpcService` / `StartThriftService` 单独使用时不会监听 `service.yaml`。
- **gRPC / Thrift 的版本是冻结的。** 升级 `google.golang.org/grpc` 与 `github.com/apache/thrift` 会同时影响生成代码与运行时，改之前先确认 protoc-gen-go 与 thrift 编译器的版本。

## 相关阅读

- [参数绑定](11-binding.md)：Thrift 参数列表与 JSON body 的完整规则
- [代码生成](12-code-generation.md)：`GrpcClient` / `ThriftClient` 模板与 `gen/` 产物
- [快速开始](02-getting-started.md)：两条链路各自的起步步骤
- [service.yaml 配置](03-service-yaml.md)：`grpc_service_*` 与 `thrift_service_*` 键
- [部署](18-deployment.md)：同时跑网关与 RPC 服务的部署形态
