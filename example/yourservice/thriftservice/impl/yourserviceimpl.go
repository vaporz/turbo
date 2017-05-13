package impl

import (
	"turbo/example/yourservice/gen/thrift/gen-go/gen"
	"fmt"
)

type YourService struct {
}

func (s YourService) SayHello(yourName string, values *gen.CommonValues, helloValues *gen.HelloValues) (r *gen.SayHelloResponse, err error) {
	fmt.Println(values.TransactionId)
	fmt.Println(helloValues.Message)
	return &gen.SayHelloResponse{Message: "[thrift server]Hello, " + yourName}, nil
}

func (s YourService) EatApple(num int32, stringValue string, boolValue bool) (r *gen.EatAppleResponse, err error) {
	msg := fmt.Sprintf("[thrift server]Good taste! Apple num=%d, string=%s, bool=%t", num, stringValue, boolValue)
	return &gen.EatAppleResponse{Message: msg}, nil
}
