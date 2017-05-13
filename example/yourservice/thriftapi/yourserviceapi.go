package main

import (
	"turbo"
	"turbo/example/yourservice/gen"
	t "turbo/example/yourservice/gen/thrift/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
	i "turbo/example/yourservice/interceptor"
	"reflect"
	"net/http"
)

func main() {
	turbo.Intercept([]string{"GET"}, "/hello", i.LogInterceptor{})
	turbo.RegisterMessageFieldConvertor(new(t.HelloValues), convertHelloValues)
	turbo.StartThriftHTTPServer("turbo/example/yourservice", thriftClient, gen.ThriftSwitcher)
}

func thriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.NewYourServiceClientFactory(trans, f)
}

func convertHelloValues(req *http.Request) reflect.Value {
	result := &t.HelloValues{}
	result.Message = "a message from convertor"
	return reflect.ValueOf(result)
}
