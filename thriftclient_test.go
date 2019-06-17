package turbo

import (
	"testing"

	"github.com/apache/thrift/lib/go/thrift"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestThriftInit(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.Fail(t, "should not panic")
		}
	}()
	s := &ThriftServer{tClient: new(thriftClient)}
	s.tClient.thriftService = make(map[string]interface{})
	s.tClient.init("", func(thrift.TTransport, thrift.TProtocolFactory) map[string]interface{} { return nil })
}

func TestThriftClose(t *testing.T) {
	s := &ThriftServer{tClient: new(thriftClient)}
	err := s.tClient.close()
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
	s := &ThriftServer{tClient: new(thriftClient)}
	s.Service("")
}
