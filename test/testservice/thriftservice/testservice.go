package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/thriftservice/impl"
	"os"
	"os/signal"
	"syscall"
	"fmt"
)

func main() {
	s := turbo.NewThriftServer(nil, turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	s.StartThriftService(impl.TProcessor)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
