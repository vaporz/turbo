package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	//gcomponent "github.com/vaporz/turbo/test/testservice/grpcapi/component"
	//gimpl "github.com/vaporz/turbo/test/testservice/grpcservice/impl"
	tcomponent "github.com/vaporz/turbo/test/testservice/thriftapi/component"
	timpl "github.com/vaporz/turbo/test/testservice/thriftservice/impl"
)

// TODO go generate: turbo generate

func main() {
	//s := turbo.NewGrpcServer(&gcomponent.ServiceInitializer{}, turbo.GOPATH() + "/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	//s.StartGRPC(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)
	s := turbo.NewThriftServer(&tcomponent.ServiceInitializer{}, turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	s.StartTHRIFT(tcomponent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
