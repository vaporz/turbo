package main

import (
	"turbo"
	"google.golang.org/grpc"
	"turbo/example/yourservice/gen"
	i "turbo/example/yourservice/interceptor"
	"net/http"
	"errors"
	"strconv"
)

func main() {
	// TODO support HTTP method
	turbo.Intercept([]string{"GET"}, "/eat_apple/{num:[0-9]+}", i.LogInterceptor{})
	turbo.Intercept([]string{"GET"},"/a/a", i.LogInterceptor{Msg: "url interceptor"})
	turbo.Intercept([]string{},"/a/", i.LogInterceptor{Msg: "path interceptor"})
	turbo.SetPreprocessor("/eat_apple/{num:[0-9]+}", checkNum)
	turbo.SetHijacker("/eat_apple/{num:[0-9]+}", hijackEatApple)
	turbo.StartGrpcHTTPServer("turbo/example/yourservice", grpcClient, gen.Switcher)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return gen.NewYourServiceClient(conn)
}

func hijackEatApple(resp http.ResponseWriter, req *http.Request) {
	client := turbo.GrpcService().(gen.YourServiceClient)
	r := new(gen.EatAppleRequest)
	r.Num = "999"
	res, err := client.EatApple(req.Context(), r)
	if err == nil {
		resp.Write([]byte(res.String() + "\n"))
	} else {
		resp.Write([]byte(err.Error() + "\n"))
	}
}

func checkNum(resp http.ResponseWriter, req *http.Request) error {
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
