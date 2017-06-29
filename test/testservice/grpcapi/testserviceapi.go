package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/grpcapi/component"
)

func main() {
	component.InitComponents()
	turbo.StartGrpcHTTPServer(turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml", component.GrpcClient, gen.GrpcSwitcher)
}
