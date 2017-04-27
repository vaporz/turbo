package gen

import (
	"reflect"
	"net/http"
	cm "zx/demo/framework"
	pb "zx/demo/framework/example/inventoryservice/proto"
	client "zx/demo/framework/clients"
	"fmt"
)

var Handler = func(methodName string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		switch methodName { 
		case "GetVideoList":
			cm.ParseRequestForm(req)
			request := pb.GetVideoListRequest{}
			theType := reflect.TypeOf(request)
			theValue := reflect.ValueOf(&request).Elem()
			fieldNum := theType.NumField()
			for i := 0; i < fieldNum; i++ {
				fieldName := theType.Field(i).Name
				v, ok := req.Form[cm.ToSnakeCase(fieldName)]
				if ok && len(v) > 0 {
					theValue.FieldByName(fieldName).SetString(v[0])
				}
			}
			params := make([]reflect.Value, 2)
			params[0] = reflect.ValueOf(req.Context())
			params[1] = reflect.ValueOf(&request)
			result := reflect.ValueOf(client.GrpcService().(pb.InventoryServiceClient)).MethodByName(methodName).Call(params)

			rsp := result[0].Interface().(*pb.GetVideoListResponse)
			if result[1].Interface() == nil {
				resp.Write([]byte(rsp.String() + "\n"))
			} else {
				resp.Write([]byte(result[1].Interface().(error).Error() + "\n"))
			}
			
		case "GetVideo":
			cm.ParseRequestForm(req)
			request := pb.GetVideoRequest{}
			theType := reflect.TypeOf(request)
			theValue := reflect.ValueOf(&request).Elem()
			fieldNum := theType.NumField()
			for i := 0; i < fieldNum; i++ {
				fieldName := theType.Field(i).Name
				v, ok := req.Form[cm.ToSnakeCase(fieldName)]
				if ok && len(v) > 0 {
					theValue.FieldByName(fieldName).SetString(v[0])
				}
			}
			params := make([]reflect.Value, 2)
			params[0] = reflect.ValueOf(req.Context())
			params[1] = reflect.ValueOf(&request)
			result := reflect.ValueOf(client.GrpcService().(pb.InventoryServiceClient)).MethodByName(methodName).Call(params)

			rsp := result[0].Interface().(*pb.GetVideoResponse)
			if result[1].Interface() == nil {
				resp.Write([]byte(rsp.String() + "\n"))
			} else {
				resp.Write([]byte(result[1].Interface().(error).Error() + "\n"))
			}
			
		default:
			resp.Write([]byte(fmt.Sprintf("No such grpc method[%s]", methodName)))
		}
	}
}