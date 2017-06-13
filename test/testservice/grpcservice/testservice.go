package main

import (
	"github.com/vaporz/turbo/test/testservice/grpcservice/impl"
	"github.com/vaporz/turbo"
)

func main() {
	turbo.StartGrpcService(50052, "github.com/vaporz/turbo/test/testservice", "service", impl.RegisterServer)
}
