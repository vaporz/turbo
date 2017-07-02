package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/grpcservice/impl"
)

func main() {
	s := turbo.NewGrpcServer("grpc", turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	s.StartGrpcService(impl.RegisterServer)
}
