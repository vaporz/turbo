package gen

import (
	"turbo/example/yourservice/gen/thrift/gen-go/gen"
	"reflect"
	"net/http"
	"turbo"
	"errors"
)

/*
this is a generated file, DO NOT EDIT!
 */
var ThriftSwitcher = func(methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName { 
	case "SayHello":
		args := gen.YourServiceSayHelloArgs{}
		params, err := turbo.BuildArgs(reflect.TypeOf(args), reflect.ValueOf(args), req, buildStructArg)
		if err != nil {
			return nil, err
		}
		return turbo.ParseResult(callThriftMethod(methodName, params))
	case "EatApple":
		args := gen.YourServiceEatAppleArgs{}
		params, err := turbo.BuildArgs(reflect.TypeOf(args), reflect.ValueOf(args), req, buildStructArg)
		if err != nil {
			return nil, err
		}
		return turbo.ParseResult(callThriftMethod(methodName, params))
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}

func callThriftMethod(methodName string, params []reflect.Value) []reflect.Value {
	return reflect.ValueOf(turbo.ThriftService().(*gen.YourServiceClient)).MethodByName(methodName).Call(params)
}

func buildStructArg(typeName string, req *http.Request) (v reflect.Value, err error) {
	switch typeName { 
	case "CommonValues":
		request := &gen.CommonValues{}
		err = turbo.BuildStruct(reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return v, err
		}
		return reflect.ValueOf(request), nil
	case "HelloValues":
		request := &gen.HelloValues{}
		err = turbo.BuildStruct(reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return v, err
		}
		return reflect.ValueOf(request), nil
	default:
		return v, errors.New("unknown typeName[" + typeName + "]")
	}
}
