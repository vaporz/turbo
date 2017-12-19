package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/thriftapi/component"
	"os"
	"os/signal"
	"syscall"
	"fmt"
)

func main() {
	s := turbo.NewThriftServer(&component.ServiceInitializer{}, turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	s.StartHTTPServer(component.ThriftClient, gen.ThriftSwitcher)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
