package turbo

import (
	"errors"
	"git.apache.org/thrift.git/lib/go/thrift"
)

type thriftClient struct {
	thriftService interface{}
	transport     thrift.TTransport
	factory       thrift.TProtocolFactory
}

func (t *thriftClient) init(clientCreator func(trans thrift.TTransport, f thrift.TProtocolFactory) interface{}) error {
	if t.thriftService != nil {
		return nil
	}
	addr := Config.ThriftServiceAddress()
	if len(addr) == 0 {
		return errors.New("Error: missing [thrift_service_address] in config")
	}
	log.Debugf("connecting thrift addr: %s", addr)
	err := t.connect(addr)
	if err == nil {
		t.thriftService = clientCreator(t.transport, t.factory)
	}
	return err
}

func (t *thriftClient) connect(hostPort string) (err error) {
	tSocket, err := thrift.NewTSocket(hostPort)
	if err != nil {
		return err
	}
	t.transport, err = thrift.NewTTransportFactory().GetTransport(tSocket)
	if err != nil {
		return err
	}
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

// ThriftService returns a Thrift client instance,
// example: client := turbo.ThriftService().(proto.YourServiceClient)
func ThriftService() interface{} {
	if client == nil || client.tClient == nil || client.tClient.thriftService == nil {
		log.Panic("thrift connection not initiated!")
	}
	return client.tClient.thriftService
}
