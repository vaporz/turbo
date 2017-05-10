package main

import (
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/example/yourservice/gen"
	t "github.com/vaporz/turbo/example/yourservice/gen/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func main() {
	turbo.StartThriftHTTPServer("github.com/vaporz/turbo/example/yourservice", thriftClient, gen.ThriftSwitcher)
}

func thriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.NewYourServiceClientFactory(trans, f)
}
