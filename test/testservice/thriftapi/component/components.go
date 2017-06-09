package component

import (
	t "github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.NewTestServiceClientFactory(trans, f)
}

func InitComponents() {
}
