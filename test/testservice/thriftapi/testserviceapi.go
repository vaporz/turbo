package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/thriftapi/component"
)

func main() {
	s := turbo.NewThriftServer(&component.ServiceInitializer{}, turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml")
	s.StartThriftHTTPServer(component.ThriftClient, gen.ThriftSwitcher)
}
