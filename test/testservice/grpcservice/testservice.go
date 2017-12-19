package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/grpcservice/impl"
	"os"
	"os/signal"
	"syscall"
	"fmt"
)

func main() {
	s := turbo.NewGrpcServer(nil, turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	s.StartGrpcService(impl.RegisterServer)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
