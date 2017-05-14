package main

import (
	"turbo"
	"google.golang.org/grpc"
	"turbo/example/yourservice/gen/proto"
	"turbo/example/yourservice/gen"
	i "turbo/example/yourservice/interceptor"
	"net/http"
	"strconv"
	"errors"
	"reflect"
)

func main() {
	turbo.Intercept([]string{"GET"}, "/hello", i.LogInterceptor{})
	turbo.Intercept([]string{"GET"}, "/eat_apple/{num:[0-9]+}", i.LogInterceptor{})
	turbo.Intercept([]string{"GET"}, "/a/a", i.LogInterceptor{Msg: "url interceptor"})
	turbo.Intercept([]string{}, "/a/", i.LogInterceptor{Msg: "path interceptor"})
	turbo.SetPreprocessor("/eat_apple/{num:[0-9]+}", preEatApple)
	//turbo.SetHijacker("/eat_apple/{num:[0-9]+}", hijackEatApple)
	turbo.SetPostprocessor("/eat_apple/{num:[0-9]+}", postEatApple)

	//turbo.RegisterMessageFieldConvertor(new(proto.CommonValues), convertCommonValues)

	turbo.StartGrpcHTTPServer("turbo/example/yourservice", grpcClient, gen.GrpcSwitcher)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return proto.NewYourServiceClient(conn)
}

func convertCommonValues(req *http.Request) reflect.Value {
	result := &proto.CommonValues{}
	result.SomeId = 1111111
	return reflect.ValueOf(result)
}

func hijackEatApple(resp http.ResponseWriter, req *http.Request) {
	client := turbo.GrpcService().(proto.YourServiceClient)
	r := new(proto.EatAppleRequest)
	r.Num = 999
	res, err := client.EatApple(req.Context(), r)
	if err == nil {
		resp.Write([]byte(res.String() + "\n"))
	} else {
		resp.Write([]byte(err.Error() + "\n"))
	}
}

func preEatApple(resp http.ResponseWriter, req *http.Request) error {
	num, err := strconv.Atoi(req.Form["num"][0])
	if err != nil {
		resp.Write([]byte("'num' is not numberic"))
		return errors.New("invalid num")
	}
	if num > 5 {
		resp.Write([]byte("Too many apples!\n"))
		return errors.New("Too many apples")
	}
	return nil
}

func postEatApple(resp http.ResponseWriter, req *http.Request, serviceResp interface{}) {
	sr := serviceResp.(*proto.EatAppleResponse)
	resp.Write([]byte("this is from postprocesser, message=" + sr.Message))
}
