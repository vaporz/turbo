package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/grpcservice/impl"
)

func main() {
	turbo.StartGrpcService("github.com/vaporz/turbo/test/testservice", "service", impl.RegisterServer)
}
