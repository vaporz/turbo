package turbo

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"log"
)

var (
	tClient       = new(thriftClient)
	thriftService interface{}
)

type thriftClient struct {
	transport thrift.TTransport
	factory   thrift.TProtocolFactory
}

func (t *thriftClient) connect(hostPort string) (err error) {
	transport, err := thrift.NewTSocket(hostPort)
	if err != nil {
		return err
	}
	t.transport = thrift.NewTTransportFactory().GetTransport(transport)
	if err := t.transport.Open(); err != nil {
		return err
	}
	t.factory = thrift.NewTBinaryProtocolFactoryDefault()
	return nil
}

func (t *thriftClient) close() error {
	if t.transport == nil {
		return nil
	}
	return t.transport.Close()
}

func initThriftService(clientCreator func(trans thrift.TTransport, f thrift.TProtocolFactory) interface{}) error {
	if thriftService != nil {
		return nil
	}
	err := tClient.connect(configs[thriftServiceAddress])
	if err == nil {
		thriftService = clientCreator(tClient.transport, tClient.factory)
	}
	return err
}

func closeThriftService() error {
	return tClient.close()
}

// ThriftService returns a Thrift client instance,
// example: client := turbo.ThriftService().(proto.YourServiceClient)
func ThriftService() interface{} {
	if thriftService == nil {
		log.Fatalln("thrift connection not initiated!")
	}
	return thriftService
}
