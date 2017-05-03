# Turbo
## Features
 * Turbo generates a reverse-proxy server which translates a HTTP request into a grpc request.
 * Interceptors.
 * PreProcessor and Hijacker: customizable url-grpc mapping process.
## Create a service on the fly
### 0, Before the start
Obviously, you have to install [Golang](https://golang.org) and [Protocol buffers](https://developers.google.com/protocol-buffers/) firstly.
### 1, Install Turbo
```sh
git clone https://github.com/vaporz/turbo.git
cd turbo/turbo
go install
```
Test in terminal:
```sh
$turbo
Usage:
  turbo [command]

Available Commands:
  create      Create a project with runnable HTTP server and gRPC server
  generate    Generate Golang codes according to service.yaml and [service_name].proto
  help        Help about any command
  init        Create an empty project
```
### 2, create your service
```sh
turbo create [package_path] [service_name]
$turbo create yourservice YourService
```
A folder named "yourservice" should have appeared at "$GOPATH/src".

That's it! It's Done! Now let's Play!

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

## Interceptor
 TODO
## PreProcesser and Hijacker
 TODO
