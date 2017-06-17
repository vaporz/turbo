package component

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	t "github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
)

func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.NewTestServiceClientFactory(trans, f)
}

func InitComponents() {
}
