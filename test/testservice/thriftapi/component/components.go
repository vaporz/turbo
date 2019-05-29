package component

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/vaporz/turbo"
	t "github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/services"
)

func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) map[string]interface{} {
	iprot := f.GetProtocol(trans)
	return map[string]interface{}{
		"TestService":    t.NewTestServiceClientProtocol(trans, iprot, thrift.NewTMultiplexedProtocol(iprot, "TestService")),
		"MinionsService": t.NewMinionsServiceClientProtocol(trans, iprot, thrift.NewTMultiplexedProtocol(iprot, "MinionsService")),
	}
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
