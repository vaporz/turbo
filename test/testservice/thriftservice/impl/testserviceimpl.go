package impl

import (
	"github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func TProcessor() thrift.TProcessor {
	return gen.NewTestServiceProcessor(TestService{})
}

type TestService struct {
}

func (s TestService) SayHello(yourName string) (r *gen.SayHelloResponse, err error) {
	return &gen.SayHelloResponse{Message: "[thrift server]Hello, " + yourName}, nil
}
