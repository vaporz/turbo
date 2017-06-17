package turbo

import (
	logger "github.com/sirupsen/logrus"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestThriftInit(t *testing.T) {
	thriftServiceAddress := Config.ThriftServiceAddress()
	Config.SetThriftServiceAddress("")

	client = &Client{tClient: new(thriftClient)}
	err := client.tClient.init(func(thrift.TTransport, thrift.TProtocolFactory) interface{} { return nil })
	assert.NotNil(t, err)
	assert.Equal(t, "Error: missing [thrift_service_address] in config", err.Error())

	client.tClient.thriftService = ""
	err = client.tClient.init(func(thrift.TTransport, thrift.TProtocolFactory) interface{} { return nil })
	assert.Nil(t, err)

	Config.SetThriftServiceAddress(thriftServiceAddress)
}

func TestThriftClose(t *testing.T) {
	client = &Client{tClient: new(thriftClient)}
	err := client.tClient.close()
	assert.Nil(t, err)
}

func TestThriftService(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.Equal(t, "thrift connection not initiated!", err.(*logger.Entry).Message)
		} else {
			t.Errorf("The code did not panic")
		}
	}()
	client = &Client{tClient: new(thriftClient)}
	ThriftService()
}
