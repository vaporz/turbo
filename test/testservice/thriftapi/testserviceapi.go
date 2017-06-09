package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/thriftapi/component"
)

func main() {
	component.InitComponents()
	turbo.StartThriftHTTPServer("github.com/vaporz/turbo/test/testservice", "service",
		component.ThriftClient, gen.ThriftSwitcher)
}
