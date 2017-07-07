package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	gcomponent "github.com/vaporz/turbo/test/testservice/grpcapi/component"
	gimpl "github.com/vaporz/turbo/test/testservice/grpcservice/impl"
	//tcompoent "github.com/vaporz/turbo/test/testservice/thriftapi/component"
	//timpl "github.com/vaporz/turbo/test/testservice/thriftservice/impl"
)

func main() {
	s := turbo.NewGrpcServer(turbo.GOPATH() + "/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	gcomponent.RegisterComponents(s)
	s.StartGRPC(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	//s := turbo.NewThriftServer("thrift", turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	//tcompoent.RegisterComponents(s)
	//s.StartTHRIFT(tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
