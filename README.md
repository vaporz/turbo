# Turbo
## Features
 * Turbo generates a reverse-proxy server which translates a HTTP request into a grpc request.
 * Interceptors.
 * PreProcessor and Hijacker: customizable url-grpc mapping process.

## How to run example
```sh
git clone https://github.com/vaporz/turbo.git
cd turbo/example/inventoryservice

Start grpc service:
go run service/inventoryservice.go

Start HTTP server:
go run ui_api.go -p turbo/example/inventoryservice

Test:
curl "http://localhost:8081/videos?network_id=122222"
```
## Basic Usage
### 1, Install Turbo
```sh
git clone https://github.com/vaporz/turbo.git
cd turbo/turbo
go install
```
Test in console:
```sh
$ turbo
Usage:
  turbo [command]

Available Commands:
  generate    generate handler.go
```
### 2, Define your gRPC service
```sh
cd $GOPATH
mkdir yourservice
cd yourservice
vim yourservice.proto
```
Edit "yourservice.proto":
```protobuf
syntax = "proto3";
package gen;
message SayHelloRequest {
    string yourName = 1;
}
message SayHelloResponse {
    string message = 1;
}
service YourService {
    rpc sayHello (SayHelloRequest) returns (SayHelloResponse) {}
}
```
### 3, Define url-grpc mapping and generate switcher.go
TODO: 'turbo init', create "gen" folder, generate ".pb.go" file, generate service.yaml, generate yourserviceapi.go<br>
 * Edit "yourservice/service.yaml":
```yaml
config:
  port: 8081
  service_name: YourService
  service_address: 127.0.0.1:50051
urlmapping:
  - GET /hello SayHello
```
This defines that, requests for "GET /hello" will be redireted to "YourService.SayHello()".

* Generate "switcher.go" in "yourservice/gen":
```sh
$turbo generate yourservice
```
### 4, Implement your gRPC server
 * Generate protobuf stub file "yourservice.pb.go" in "yourservice/gen".
```sh
$ protoc -I . yourservice.proto --go_out=plugins=grpc:gen
```
 * Implement gRPC server:
```sh
$mkdir -p $GOPATH/src/yourservice/service/impl
```
  Edit "yourservice/service/impl/yourserviceimpl.go"
```go
package impl
import (
	"golang.org/x/net/context"
	"yourservice/gen"
)
type YourService struct {
}
func (s *YourService) SayHello(ctx context.Context, req *gen.SayHelloRequest) (*gen.SayHelloResponse, error) {
	return &gen.SayHelloResponse{Message: "Hello, " + req.YourName}, nil
}
```
  Edit "yourservice/service/yourservice.go"
```go
package main
import (
	"net"
	"log"
	"google.golang.org/grpc"
	"yourservice/service/impl"
	"yourservice/gen"
	"google.golang.org/grpc/reflection"
)
func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	gen.RegisterYourServiceServer(grpcServer, &impl.YourService{})
	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
```
### 5, Add main() for HTTP server
Edit "yourservice/api.go"
```go
package main
import (
	"flag"
	"fmt"
	"os"
	"turbo"
	"google.golang.org/grpc"
	"yourservice/gen"
)
var (
	pkgPath = flag.String("p", "", "package path")
)
func main() {
	flag.Parse()
	if len(*pkgPath) == 0 {
		fmt.Println("package path is empty")
		os.Exit(1)
	}
	turbo.StartGrpcHTTPServer(*pkgPath, grpcClient, gen.Switcher)
}
func grpcClient(conn *grpc.ClientConn) interface{} {
	return gen.NewYourServiceClient(conn)
}
```

Done! Let's Play!

Start gRPC server:
```sh
go run service/yourservice.go
```
Start HTTP server:
```sh
go run api.go -p yourservice
```
Send a request:
```sh
$ curl "http://localhost:8081/hello?your_name=Alice"
message:"Hello, Alice"
```

## Interceptor
 TODO
## PreProcesser and Hijacker
 TODO
