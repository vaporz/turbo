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
	s := turbo.NewServer("grpc", turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	gcomponent.InitComponents(s)
	s.StartGRPC(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	//s := turbo.NewServer("thrift", turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	//tcompoent.InitComponents(s)
	//s.StartTHRIFT(tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
