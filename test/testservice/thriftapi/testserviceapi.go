package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/thriftapi/component"
)

func main() {
	component.InitComponents()
	turbo.StartThriftHTTPServer(turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml",
		component.ThriftClient, gen.ThriftSwitcher)
}
