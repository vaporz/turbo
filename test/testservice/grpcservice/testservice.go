package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/grpcservice/impl"
)

func main() {
	turbo.StartGrpcService(50052, "github.com/vaporz/turbo/test/testservice", "service", impl.RegisterServer)
}
