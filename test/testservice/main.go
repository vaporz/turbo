package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	gcomponent "github.com/vaporz/turbo/test/testservice/grpcapi/component"
	gimpl "github.com/vaporz/turbo/test/testservice/grpcservice/impl"
	//tcomponent "github.com/vaporz/turbo/test/testservice/thriftapi/component"
	//timpl "github.com/vaporz/turbo/test/testservice/thriftservice/impl"
)

// TODO go generate: turbo generate

func main() {
	s := turbo.NewGrpcServer(turbo.GOPATH() + "/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	gcomponent.RegisterComponents(s)
	s.Initializer = &gcomponent.ServiceInitializer{}
	s.StartGRPC(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)
	//s := turbo.NewThriftServer(turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	//tcomponent.RegisterComponents(s)
	//s.Initializer = &tcomponent.ServiceInitializer{}
	//s.StartTHRIFT(tcomponent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
