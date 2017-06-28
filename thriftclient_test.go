package turbo

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestThriftInit(t *testing.T) {
	client = &Client{tClient: new(thriftClient)}
	client.tClient.thriftService = ""
	err := client.tClient.init(func(thrift.TTransport, thrift.TProtocolFactory) interface{} { return nil })
	assert.Nil(t, err)
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
