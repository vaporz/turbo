package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/thriftservice/impl"
)

func main() {
	turbo.StartThriftService(turbo.GOPATH()+"/src/github.com/vaporz/turbo/test/testservice/service.yaml", impl.TProcessor)
}
