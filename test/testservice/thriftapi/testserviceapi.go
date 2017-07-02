package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/thriftapi/component"
)

func main() {
	s := turbo.NewThriftServer("thrift", turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	component.InitComponents(s)
	s.StartThriftHTTPServer(component.ThriftClient, gen.ThriftSwitcher)
}
