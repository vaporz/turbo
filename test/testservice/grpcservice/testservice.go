package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/grpcservice/impl"
)

func main() {
	turbo.StartGrpcService(turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml", impl.RegisterServer)
}
