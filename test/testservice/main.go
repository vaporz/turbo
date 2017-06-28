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
	gcomponent.InitComponents()
	turbo.StartGRPC("github.com/vaporz/turbo/test/testservice", "service",
		gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	//tcompoent.InitComponents()
	//turbo.StartTHRIFT("github.com/vaporz/turbo/test/testservice", "service",
	//	tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
