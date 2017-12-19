package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	//gcomponent "github.com/vaporz/turbo/test/testservice/grpcapi/component"
	//gimpl "github.com/vaporz/turbo/test/testservice/grpcservice/impl"
	tcomponent "github.com/vaporz/turbo/test/testservice/thriftapi/component"
	timpl "github.com/vaporz/turbo/test/testservice/thriftservice/impl"
	"os/signal"
	"os"
	"syscall"
	"fmt"
)

// TODO go generate: turbo generate

func main() {
	//s := turbo.NewGrpcServer(&gcomponent.ServiceInitializer{}, turbo.GOPATH() + "/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	//s.Start(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)
	s := turbo.NewThriftServer(&tcomponent.ServiceInitializer{}, turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	s.Start(tcomponent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor) // TODO change name, Start -> Start

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
