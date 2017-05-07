package main

import (
	"turbo"
	"turbo/example/yourservice/gen"
	t "turbo/example/yourservice/gen/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func main() {
	turbo.StartThriftHTTPServer("turbo/example/yourservice", thriftClient, gen.ThriftSwitcher)
}

func thriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.NewYourServiceClientFactory(trans, f)
}
