package main

import (
	"github.com/vaporz/turbo/test/testservice/thriftservice/impl"
	"github.com/vaporz/turbo"
)

func main() {
	turbo.StartThriftService(50052, "github.com/vaporz/turbo/test/testservice", "service", impl.TProcessor)
}
