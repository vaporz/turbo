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
				return nil, err
			}
		}
		params := make([]reflect.Value, 2)
		params[0] = reflect.ValueOf(req.Context())
		params[1] = reflect.ValueOf(&request)
		result := reflect.ValueOf(turbo.GrpcService().(YourServiceClient)).MethodByName(methodName).Call(params)
		if result[1].Interface() == nil {
			return result[0].Interface(), nil
		} else {
			return nil, result[1].Interface().(error)
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
				return nil, err
			}
		}
		params := make([]reflect.Value, 2)
		params[0] = reflect.ValueOf(req.Context())
		params[1] = reflect.ValueOf(&request)
		result := reflect.ValueOf(turbo.GrpcService().(YourServiceClient)).MethodByName(methodName).Call(params)
		if result[1].Interface() == nil {
			return result[0].Interface(), nil
		} else {
			return nil, result[1].Interface().(error)
		}
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}
