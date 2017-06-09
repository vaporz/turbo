package gen

import (
	"github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
	"github.com/vaporz/turbo"
	"reflect"
	"net/http"
	"errors"
)

/*
this is a generated file, DO NOT EDIT!
 */
var ThriftSwitcher = func(methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName { 
	case "SayHello":
		args := gen.TestServiceSayHelloArgs{}
		params, err := turbo.BuildArgs(reflect.TypeOf(args), reflect.ValueOf(args), req, buildStructArg)
		if err != nil {
			return nil, err
		}
		return turbo.ThriftService().(*gen.TestServiceClient).SayHello(
			params[0].Interface().(string), )
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}

func buildStructArg(typeName string, req *http.Request) (v reflect.Value, err error) {
	switch typeName { 
	default:
		return v, errors.New("unknown typeName[" + typeName + "]")
	}
}
