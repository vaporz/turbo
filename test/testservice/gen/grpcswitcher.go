package gen

import (
	"errors"
	"github.com/vaporz/turbo"
	g "github.com/vaporz/turbo/test/testservice/gen/proto"
	"net/http"
	"reflect"
)

/*
this is a generated file, DO NOT EDIT!
*/
// GrpcSwitcher is a runtime func with which a server starts.
var GrpcSwitcher = func(methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName {
	case "SayHello":
		request := &g.SayHelloRequest{Values: &g.CommonValues{}}
		err = turbo.BuildStruct(reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return nil, err
		}
		return turbo.GrpcService().(g.TestServiceClient).SayHello(req.Context(), request)
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}
