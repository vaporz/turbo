package component

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	t "github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
	"github.com/vaporz/turbo"
)

func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.NewTestServiceClientFactory(trans, f)
}

func RegisterComponents(s *turbo.ThriftServer) {
}

type ServiceInitializer struct {
}

// InitService from defaultInitializer does nothing
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	return nil
}

// StopService from defaultInitializer does nothing
func (i *ServiceInitializer) StopService(s turbo.Servable) {
}
