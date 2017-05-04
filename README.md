# Turbo
## Features
 * Turbo generates a reverse-proxy server which translates a HTTP request into a grpc request.
 * Interceptors.
 * PreProcessor and Hijacker: customizable url-grpc mapping process.
 * (TODO) Support different IDL tools, such as protocol buffers, thrift, etc.

## Create a service on the fly
### 0, Before the start
Obviously, you have to install [Golang](https://golang.org) and [Protocol buffers](https://developers.google.com/protocol-buffers/) firstly.
### 1, Install Turbo
```sh
git clone https://github.com/vaporz/turbo.git
cd turbo/turbo
go install
```

### 2, Create your service
```sh
$ turbo create package/path/to/yourservice YourService
```
Directory "$GOPATH/src/package/path/to/yourservice" should appear.<br>
There're also some codes in this folder, similar with this example:<br>
https://github.com/vaporz/turbo/tree/master/example/yourservice

### 3, Run
That's it! Now let's Play!

Start gRPC server and HTTP server:
```sh
cd $GOPATH/src/yourservice
go run service/yourservice.go
go run yourserviceapi.go
```
Send a request:
```sh
$ curl "http://localhost:8081/hello?your_name=Alice"
message:"Hello, Alice"
```
## Rules and Conventions
There are some rules when you use turbo.
 * When defining a gRPC service, if the name of a gRPC method is "methodName", then the name
 of request message and response message MUST be "MethodNameRequest" and "MethodNameResponse".
 * If multiple paths are assigned to $GOPATH(e.g. devided by ':'), then the last path is used by turbo as GOPATH.
 * When parsing request parameters, values from URL path has a higher priority than those from query string or body.
<br> e.g. In a request like "GET /book/1234?id=5678", both "1234" and "5678" are values to "id", but "1234" is picked
 as value to key "id".
 * The value of a key with all lower case characters has a higher priority to the value of a key with upper case
 characters.<br>
 e.g. In a request like "GET /book?id=1234&Id=5678", "1234" is used for key "id".

## Command line tools
### turbo create [package_name] [service_name]
'turbo create' creates a project with runnable HTTP server and gRPC server.<br>
Project structure:
```sh
yourservice
├── gen
│   ├── switcher.go
│   └── yourservice.pb.go
├── service
│   ├── impl
│   │    └── yourserviceimpl.go
│   └── yourservice.go
├── service.yaml
├── yourservice.proto
└── yourserviceapi.go
```
### turbo generate [package_name] [service_name]
'turbo generate' generates switcher.go and [service_name].pb.go to 'gen' directory.<br>
This command is useful when either service.yaml or [service_name].proto is changed.<br>
For example, add a new API, change an existing API, change url-grpc mapping, etc.

## How to add a new API
### 1, Define new gRPC API
Modify "yourservice.proto", add new method "eatApple":
```proto
message EatAppleRequest {
    string num = 1;
}

message EatAppleResponse {
    string message = 1;
}

service YourService {
    rpc sayHello (SayHelloRequest) returns (SayHelloResponse) {}
    rpc eatApple (EatAppleRequest) returns (EatAppleResponse) {}
}
```
### 2, Add new url-grpc mapping
Modify "service.yaml":
```config
urlmapping:
  - GET /hello SayHello
  - GET /eat_apple EatApple
```
### 3, Generate new codes
```sh
turbo generate package/path/to/yourservice YourService
```

### 4, Implement new gRPC method
Modify "yourserviceimpl.go":
```go
func (s *YourService) SayHello(ctx context.Context, req *gen.SayHelloRequest) (*gen.SayHelloResponse, error) {
	return &gen.SayHelloResponse{Message: "Hello, " + req.YourName}, nil
}

func (s *YourService) EatApple(ctx context.Context, req *gen.EatAppleRequest) (*gen.EatAppleResponse, error) {
	return &gen.EatAppleResponse{Message: "Good taste!"}, nil
}
```

Now, stop and restart both HTTP and gRPC server:
```sh
$ go run service/yourservice.go
$ go run yourserviceapi.go
$ curl "http://localhost:8081/eat_apple"
message:"Good taste!"
```

## Interceptor
 TODO
## PreProcesser and Hijacker
 TODO
