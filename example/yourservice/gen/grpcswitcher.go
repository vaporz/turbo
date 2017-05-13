package gen

import (
	"reflect"
	"net/http"
	"turbo"
	"errors"
)

/*
this is a generated file, DO NOT EDIT!
 */
var GrpcSwitcher = func(methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName {
	case "SayHello":
		request := &SayHelloRequest{Values: &CommonValues{},}
		err = turbo.BuildStruct(reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return nil, err
		}
		params := turbo.MakeParams(req, reflect.ValueOf(request))
		return turbo.ParseResult(callGrpcMethod(methodName, params))
	case "EatApple":
		request := &EatAppleRequest{}
		turbo.BuildStruct(reflect.TypeOf(request), reflect.ValueOf(&request).Elem(), req)
		params := turbo.MakeParams(req, reflect.ValueOf(&request))
		return turbo.ParseResult(callGrpcMethod(methodName, params))
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}

func callGrpcMethod(methodName string, params []reflect.Value) []reflect.Value {
	return reflect.ValueOf(turbo.GrpcService().(YourServiceClient)).MethodByName(methodName).Call(params)
}
