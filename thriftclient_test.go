package turbo

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestThriftInit(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.Fail(t, "should not panic")
		}
	}()
	s := &ThriftServer{tClient: new(thriftClient)}
	s.tClient.thriftService = ""
	s.tClient.init("", func(thrift.TTransport, thrift.TProtocolFactory) interface{} { return nil })
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
	s.Service()
}
