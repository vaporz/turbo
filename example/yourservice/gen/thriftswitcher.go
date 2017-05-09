package gen

import (
	"turbo/example/yourservice/gen/gen-go/gen"
	"reflect"
	"net/http"
	"turbo"
	"fmt"
	"errors"
)

/*
this is a generated file, DO NOT EDIT!
 */
var ThriftSwitcher = func(methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName { 
	case "SayHello":
		args := gen.YourServiceSayHelloArgs{}
		argsType := reflect.TypeOf(args)
		argsValue := reflect.ValueOf(args)
		fieldNum := argsType.NumField()
		params := make([]reflect.Value, fieldNum)
		for i := 0; i < fieldNum; i++ {
			fieldName := argsType.Field(i).Name
			v, ok := req.Form[turbo.ToSnakeCase(fieldName)]
			if !ok || len(v) <= 0 {
				v = []string{""}
			}
			value, err := turbo.ReflectValue(argsValue.FieldByName(fieldName), v[0])
			if err != nil {
				return nil, err
			}
			params[i] = value
		}
		result := reflect.ValueOf(turbo.ThriftService().(*gen.YourServiceClient)).MethodByName(methodName).Call(params)
		if result[1].Interface() == nil {
			return result[0].Interface(), nil
		} else {
			return nil, result[1].Interface().(error)
		}
	case "EatApple":
		args := gen.YourServiceEatAppleArgs{}
		argsType := reflect.TypeOf(args)
		argsValue := reflect.ValueOf(args)
		fieldNum := argsType.NumField()
		params := make([]reflect.Value, fieldNum)
		for i := 0; i < fieldNum; i++ {
			fieldName := argsType.Field(i).Name
			v, ok := req.Form[turbo.ToSnakeCase(fieldName)]
			if !ok || len(v) <= 0 {
				v = []string{""}
			}
			value, err := turbo.ReflectValue(argsValue.FieldByName(fieldName), v[0])
			if err != nil {
				return nil, err
			}
			params[i] = value
		}
		result := reflect.ValueOf(turbo.ThriftService().(*gen.YourServiceClient)).MethodByName(methodName).Call(params)
		if result[1].Interface() == nil {
			return result[0].Interface(), nil
		} else {
			return nil, result[1].Interface().(error)
		}
	default:
		resp.Write([]byte(fmt.Sprintf("No such grpc method[%s]", methodName)))
	}
	return nil, errors.New("Unknown methodName[" + methodName + "]")
}
