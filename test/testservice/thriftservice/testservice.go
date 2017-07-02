package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/thriftservice/impl"
)

func main() {
	s := turbo.NewServer("thrift", turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	s.StartThriftService(impl.TProcessor)
}
