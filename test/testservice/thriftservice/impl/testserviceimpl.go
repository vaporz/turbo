package impl

import (
	"errors"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
)

// TProcessor returns TProcessor
func TProcessor() thrift.TProcessor {
	return gen.NewTestServiceProcessor(TestService{})
}

// TestService is the struct which implements generated interface
type TestService struct {
}

// SayHello is an example entry point
func (s TestService) SayHello(values *gen.CommonValues, yourName string, int64Value int64, boolValue bool, float64Value float64,
	uint64Value int64, int32Value int32, int16Value int16, stringList []string, i32List []int32, boolList []bool, doubleList []float64) (r *gen.SayHelloResponse, err error) {
	if boolValue {
		result := fmt.Sprintf("values.TransactionId=%d, yourName=%s,int64Value=%d, boolValue=%t, float64Value=%f, "+
			"uint64Value=%d, int32Value=%d, int16Value=%d, stringList=%v, i32List=%v, boolList=%v, doubleList=%v",
			values.TransactionId, yourName, int64Value, boolValue, float64Value, uint64Value, int32Value, int16Value,
			stringList, i32List, boolList, doubleList)
		return &gen.SayHelloResponse{Message: "[thrift server]" + result}, nil
	}
	if yourName == "error" {
		return &gen.SayHelloResponse{}, errors.New("thrift error")
	}
	return &gen.SayHelloResponse{Message: "[thrift server]Hello, " + yourName}, nil
}

func (s TestService) TestJson(request *gen.TestJsonRequest) (r *gen.TestJsonResponse, err error) {
	return &gen.TestJsonResponse{Message: "[thrift server]json= " + request.String()}, nil
}
