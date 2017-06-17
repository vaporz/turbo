package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/grpcapi/component"
)

func main() {
	component.InitComponents()
	turbo.StartGrpcHTTPServer("github.com/vaporz/turbo/test/testservice", "service", component.GrpcClient, gen.GrpcSwitcher)
}
