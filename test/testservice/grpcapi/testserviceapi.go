package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/grpcapi/component"
)

func main() {
	s := turbo.NewGrpcServer(turbo.GOPATH() + "/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	component.RegisterComponents(s)
	s.Initializer = &component.ServiceInitializer{}
	s.StartGrpcHTTPServer(component.GrpcClient, gen.GrpcSwitcher)
}
