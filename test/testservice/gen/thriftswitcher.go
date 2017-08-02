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
// ThriftSwitcher is a runtime func with which a server starts.
var ThriftSwitcher = func(s turbo.Servable, methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName {

	case "SayHello":
		params, err := turbo.BuildThriftRequest(s, gen.TestServiceSayHelloArgs{}, req, buildStructArg)
		if err != nil {
			return nil, err
		}
		return s.Service().(*gen.TestServiceClient).SayHello(
			params[0].Interface().(*gen.CommonValues),
			params[1].Interface().(string),
			params[2].Interface().(int64),
			params[3].Interface().(bool),
			params[4].Interface().(float64),
			params[5].Interface().(int64),
			params[6].Interface().(int32),
			params[7].Interface().(int16), )

	case "TestJson":
		params, err := turbo.BuildThriftRequest(s, gen.TestServiceTestJsonArgs{}, req, buildStructArg)
		if err != nil {
			return nil, err
		}
		return s.Service().(*gen.TestServiceClient).TestJson(
			params[0].Interface().(*gen.TestJsonRequest), )

	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}

func buildStructArg(s turbo.Servable, typeName string, req *http.Request) (v reflect.Value, err error) {
	switch typeName {

	case "CommonValues":
		request := &gen.CommonValues{}
		turbo.BuildStruct(s, reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		return reflect.ValueOf(request), nil

	case "TestJsonRequest":
		request := &gen.TestJsonRequest{}
		turbo.BuildStruct(s, reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		return reflect.ValueOf(request), nil

	default:
		return v, errors.New("unknown typeName[" + typeName + "]")
	}
}
