package gen

import (
	"reflect"
	"net/http"
	"turbo"
	"fmt"
)

/*
this is a generated file, DO NOT EDIT!
 */
var GrpcSwitcher = func(methodName string, resp http.ResponseWriter, req *http.Request) {
	switch methodName { 
	case "SayHello":
		request := SayHelloRequest{}
		theType := reflect.TypeOf(request)
		theValue := reflect.ValueOf(&request).Elem()
		fieldNum := theType.NumField()
		for i := 0; i < fieldNum; i++ {
			fieldName := theType.Field(i).Name
			v, ok := req.Form[turbo.ToSnakeCase(fieldName)]
			if !ok || len(v) <= 0 {
				continue
			}
			err := turbo.SetValue(theValue.FieldByName(fieldName), v[0])
			if err != nil {
				resp.Write([]byte(err.Error() + "\n"))
				return
			}
		}
		params := make([]reflect.Value, 2)
		params[0] = reflect.ValueOf(req.Context())
		params[1] = reflect.ValueOf(&request)
		result := reflect.ValueOf(turbo.GrpcService().(YourServiceClient)).MethodByName(methodName).Call(params)
		rsp := result[0].Interface().(*SayHelloResponse)
		if result[1].Interface() == nil {
			resp.Write([]byte(rsp.String() + "\n"))
		} else {
			resp.Write([]byte(result[1].Interface().(error).Error() + "\n"))
		}
	case "EatApple":
		request := EatAppleRequest{}
		theType := reflect.TypeOf(request)
		theValue := reflect.ValueOf(&request).Elem()
		fieldNum := theType.NumField()
		for i := 0; i < fieldNum; i++ {
			fieldName := theType.Field(i).Name
			v, ok := req.Form[turbo.ToSnakeCase(fieldName)]
			if !ok || len(v) <= 0 {
				continue
			}
			err := turbo.SetValue(theValue.FieldByName(fieldName), v[0])
			if err != nil {
				resp.Write([]byte(err.Error() + "\n"))
				return
			}
		}
		params := make([]reflect.Value, 2)
		params[0] = reflect.ValueOf(req.Context())
		params[1] = reflect.ValueOf(&request)
		result := reflect.ValueOf(turbo.GrpcService().(YourServiceClient)).MethodByName(methodName).Call(params)
		rsp := result[0].Interface().(*EatAppleResponse)
		if result[1].Interface() == nil {
			resp.Write([]byte(rsp.String() + "\n"))
		} else {
			resp.Write([]byte(result[1].Interface().(error).Error() + "\n"))
		}
	default:
		resp.Write([]byte(fmt.Sprintf("No such grpc method[%s]", methodName)))
	}
}
