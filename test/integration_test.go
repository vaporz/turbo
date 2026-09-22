package test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	tgen "github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
	gcomponent "github.com/vaporz/turbo/test/testservice/grpcapi/component"
	gimpl "github.com/vaporz/turbo/test/testservice/grpcservice/impl"
	tcompoent "github.com/vaporz/turbo/test/testservice/thriftapi/component"
	timpl "github.com/vaporz/turbo/test/testservice/thriftservice/impl"
	"github.com/vaporz/turbo/turbo/cmd"
)

func TestMain(m *testing.M) {
	os.RemoveAll(turbo.GetWD() + "/testcreateservice")
	os.Exit(m.Run())
}

func TestCreateGrpcService(t *testing.T) {
	create(t, "grpc")
	generate(t, "grpc")
	overwriteProto()
	_ = os.RemoveAll(turbo.GetWD() + "/testcreateservice/gen")
	generate(t, "grpc")
}

func TestCreateThriftService(t *testing.T) {
	create(t, "thrift")
	generate(t, "thrift")
	overwriteThrift()
	_ = os.RemoveAll(turbo.GetWD() + "/testcreateservice/gen")
	generate(t, "thrift")
	// recover grpc gen code
	overwriteProto()
	generate(t, "grpc")
}

func component(s *turbo.Server, name string) interface{} {
	com, err := s.Component(name)
	if err != nil {
		panic(err)
	}
	return com
}

func TestGrpcService(t *testing.T) {
	httpPort := "8081"
	cfg := testConfigPath(t)
	overwriteServiceYaml(cfg, "8081", "50061", "development")

	s := turbo.NewGrpcServer(&testInitializer{}, cfg)
	s.Start(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 1000)

	runCommonTests(t, s.Server, httpPort, "grpc")

	testGet(t, "http://localhost:"+httpPort+"/hello/error",
		"rpc error: code = Unknown desc = grpc error\n")

	testGet(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true&string_list=a,b&int64_list=1,2&bool_list=true,false"+
		"&doubleList=1.1,2.2&uint64_list=3,4",
		`{"message":"{\"values\":{},\"yourName\":\"name\",\"boolValue\":true,\"stringList\":[\"a\",\"b\"],\"int64List\":[1,2],\"boolList\":[true,false],\"doubleList\":[1.1,2.2],\"uint64List\":[3,4]}"}`)

	// a list carrying an element the server cannot read is refused as a client
	// error; it used to be dropped whole, with a 200 response
	testRejectedParameter(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true&string_list=a,b&int64_list=1,a,2"+
		"&bool_list=true,a,false&doubleList=1.1,a,2.2&uint64_list=3,a,4")

	// an empty list is not the same thing as an unreadable one
	testGet(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true&string_list=",
		`{"message":"{\"values\":{},\"yourName\":\"name\",\"boolValue\":true}"}`)

	testRejectedParameter(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true&int64_list=1-2")
	testRejectedParameter(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true&bool_list=aaa")
	// doublelist normalises to the same parameter as double_list
	testRejectedParameter(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true&doublelist=aaa")

	testRejectedParameter(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true&uint64_list=aaa")

	s.Components.WithErrorHandler(component(s.Server, "errorHandler").(turbo.ErrorHandlerFunc))
	testGet(t, "http://localhost:"+httpPort+"/hello/error",
		"from errorHandler:rpc error: code = Unknown desc = grpc error")
	s.Components.Reset()

	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s.Server, "ContextValueInterceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`test1_intercepted:{"message":"{\"values\":{},\"yourName\":\"testtest\",\"int64Value\":1234567,\"boolValue\":true,\"float64Value\":1.23,\"uint64Value\":456}"}`)
	s.Components.Reset()

	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?int64_value=64&bool_value=true&float64_value=0.123&uint64_value=123",
		`{"message":"{\"values\":{},\"yourName\":\"testtest\",\"int64Value\":64,\"boolValue\":true,\"float64Value\":0.123,\"uint64Value\":123}"}`)

	s.Components.SetConvertor("CommonValues", component(s.Server, "convertProtoCommonValues").(turbo.Convertor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?bool_value=true",
		`{"message":"{\"values\":{\"someId\":1111111},\"yourName\":\"testtest\",\"boolValue\":true}"}`)
	s.Components.Reset()

	s.Components.SetConvertor("SayHelloRequest", component(s.Server, "convertProtoSayHelloRequest").(turbo.Convertor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?bool_value=true",
		`{"message":"[grpc server]Hello, from convertor"}`)
	s.Components.Reset()

	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s.Server, "MetadataInterceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`{"message":"[grpc server]Hello, testtest"}metadata:header:headerval:trailer:trailerval:peer:127.0.0.1:50061`)
	s.Components.Reset()

	body := strings.NewReader(`{"values":{"someId":123}, "yourName":"a name", "boolValue":true}`)
	testPostWithContentType(t, "http://localhost:"+httpPort+"/hello", "application/json", body,
		`{"message":"{\"values\":{\"someId\":123},\"yourName\":\"a name\",\"boolValue\":true}"}`)

	body = strings.NewReader(`{aaaaa`)
	testPostWithContentType(t, "http://localhost:"+httpPort+"/hello", "application/json", body,
		"turbo: failed to BuildRequest for json api, request body: {aaaaa, error: invalid character 'a' looking for beginning of object key string\n")

	s.Stop()
}

func TestThriftService(t *testing.T) {
	httpPort := "8082"
	cfg := testConfigPath(t)
	overwriteServiceYaml(cfg, httpPort, "50062", "production")

	s := turbo.NewThriftServer(&testInitializer{}, cfg)
	turbo.SetOutput(os.Stdout)
	s.Start(tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
	time.Sleep(time.Second * 2)

	runCommonTests(t, s.Server, httpPort, "thrift")

	testGet(t, "http://localhost:"+httpPort+"/hello/error",
		"Internal error processing sayHello: thrift error\n")

	testGet(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true",
		`{"message":"[thrift server]values.TransactionId=0, yourName=name,int64Value=0, boolValue=true, float64Value=0.000000, uint64Value=0, int32Value=0, int16Value=0, stringList=[], i32List=[], boolList=[], doubleList=[]"}`)

	testGet(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true&stringlist=a,b&i32_list=1,2,3&boolList=true,false,true&doubleList=1.1,2.2",
		`{"message":"[thrift server]values.TransactionId=0, yourName=name,int64Value=0, boolValue=true, float64Value=0.000000, uint64Value=0, int32Value=0, int16Value=0, stringList=[a b], i32List=[1 2 3], boolList=[true false true], doubleList=[1.1 2.2]"}`)

	// as on the grpc path, a list with an element the server cannot read is
	// refused instead of being dropped silently
	testRejectedParameter(t, "http://localhost:"+httpPort+"/hello/name?bool_value=true&stringlist=a,b&i32_list=1,a,3"+
		"&boolList=true,a,true&doubleList=1.1,a,2.2")

	s.Components.WithErrorHandler(component(s.Server, "errorHandler").(turbo.ErrorHandlerFunc))
	testGet(t, "http://localhost:"+httpPort+"/hello/error",
		"from errorHandler:Internal error processing sayHello: thrift error")
	s.Components.Reset()

	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s.Server, "ContextValueInterceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`test1_intercepted:{"message":"[thrift server]values.TransactionId=0, yourName=testtest,int64Value=1234567, boolValue=true, float64Value=1.230000, uint64Value=456, int32Value=0, int16Value=0, stringList=[], i32List=[], boolList=[], doubleList=[]"}`)
	s.Components.Reset()

	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?transaction_id=111&int64_value=64&bool_value=true&float64_value=0.123&uint64_value=123&int32_value=32&int16_value=16",
		`{"message":"[thrift server]values.TransactionId=111, yourName=testtest,int64Value=64, boolValue=true, float64Value=0.123000, uint64Value=123, int32Value=32, int16Value=16, stringList=[], i32List=[], boolList=[], doubleList=[]"}`)

	s.Components.SetConvertor("CommonValues", component(s.Server, "convertThriftCommonValues").(turbo.Convertor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?bool_value=true",
		`{"message":"[thrift server]values.TransactionId=222222, yourName=testtest,int64Value=0, boolValue=true, float64Value=0.000000, uint64Value=0, int32Value=0, int16Value=0, stringList=[], i32List=[], boolList=[], doubleList=[]"}`)
	s.Components.Reset()

	body := strings.NewReader(`{"StringValue":"123", "int32Value":456, "boolvalue":true}`)
	testPostWithContentType(t, "http://localhost:"+httpPort+"/testjson", "application/json", body,
		`{"message":"[thrift server]json= TestJsonRequest({StringValue:123 Int32Value:456 BoolValue:true})"}`)

	body = strings.NewReader(`{"BoolValue":true}`)
	testPostWithContentType(t, "http://localhost:"+httpPort+"/testjson/123/456", "application/json", body,
		`{"message":"[thrift server]json= TestJsonRequest({StringValue:123 Int32Value:456 BoolValue:true})"}`)

	body = strings.NewReader(`{ttttt`)
	testPostWithContentType(t, "http://localhost:"+httpPort+"/testjson/123/456", "application/json", body,
		"turbo: failed to BuildThriftRequest for json api, request body: {ttttt, error: invalid character 't' looking for beginning of object key string\n")

	s.Stop()
}
func TestHTTPGrpcService(t *testing.T) {
	httpPort := "8083"
	cfg := testConfigPath(t)
	overwriteServiceYaml(cfg, httpPort, "50063", "development")

	s := turbo.NewGrpcServer(nil, cfg)
	s.StartGrpcService(gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 300)

	s.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Millisecond * 300)

	testGet(t, "http://localhost:"+httpPort+"/hello/testtest", `{"message":"[grpc server]Hello, testtest"}`)

	s.Stop()
}

func TestHTTPThriftService(t *testing.T) {
	httpPort := "8084"
	cfg := testConfigPath(t)
	overwriteServiceYaml(cfg, httpPort, "50064", "development")

	s := turbo.NewThriftServer(nil, cfg)
	s.StartThriftService(timpl.TProcessor)
	time.Sleep(time.Millisecond * 500)

	s.StartHTTPServer(tcompoent.ThriftClient, gen.ThriftSwitcher)
	time.Sleep(time.Millisecond * 500)

	testGet(t, "http://localhost:"+httpPort+"/hello/testtest", `{"message":"[thrift server]Hello, testtest"}`)

	s.Stop()
}

func TestLoadComponentsFromConfig(t *testing.T) {
	httpPort := "8085"
	cfg := testConfigPath(t)
	overwriteServiceYamlWithGrpcComponents(cfg, httpPort, "50065", "production")

	s := turbo.NewGrpcServer(&testInitializer{}, cfg)
	_, err := s.Component("test")
	assert.Equal(t, "no such component: test, forget to register?", err.Error())
	s.StartGrpcService(gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 300)

	s.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Millisecond * 300)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest", `{"message":"[grpc server]Hello, testtest"}`)
	testGet(t, "http://localhost:"+httpPort+"/hello", `intercepted:{"message":"[grpc server]Hello, "}`)
	testGet(t, "http://localhost:"+httpPort+"/hellointerceptor", "interceptor_error:from errorHandler:error!")
	testGet(t, "http://localhost:"+httpPort+"/hello_preprocessor", `preprocessor:{"message":"[grpc server]Hello, "}`)
	testGet(t, "http://localhost:"+httpPort+"/hello_postprocessor", `postprocessor:{"message":"[grpc server]Hello, "}`)
	testGet(t, "http://localhost:"+httpPort+"/hello_hijacker", "hijacker")
	testGet(t, "http://localhost:"+httpPort+"/hello_convertor?bool_value=true", `{"message":"{\"values\":{\"someId\":1111111},\"boolValue\":true}"}`)
	testGet(t, "http://localhost:"+httpPort+"/hello_hijacker", "hijacker")
	testGet(t, "http://localhost:"+httpPort+"/hello/error", "from errorHandler:rpc error: code = Unknown desc = grpc error")

	changeServiceYamlWithGrpcComponents(cfg, httpPort, "50065", "production")
	testGetEventually(t, "http://localhost:"+httpPort+"/hello",
		`test1_intercepted:preprocessor:postprocessor:{"message":"[grpc server]Hello, "}`,
		time.Second*10)
	s.Stop()
}

// TestParameterBinding pins the precedence binding resolves conflicts with:
// injected > path > body > query/form. The last two assertions are the ones that
// describe what changed: a JSON request no longer ignores the URL, and a request
// body can no longer override a value the server verified.
func TestParameterBinding(t *testing.T) {
	httpPort := "8086"
	cfg := testConfigPath(t)
	overwriteServiceYamlForBinding(cfg, httpPort, "50066")

	s := turbo.NewGrpcServer(&testInitializer{}, cfg)
	s.StartGrpcService(gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 300)
	s.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Millisecond * 300)
	defer s.Stop()

	base := "http://localhost:" + httpPort
	greeting := `{"message":"[grpc server]Hello, `

	// the query reaches the binding path
	testGet(t, base+"/hello?your_name=from-query", greeting+`from-query"}`)

	// the body wins over the query for the fields it carries
	testPostWithContentType(t, base+"/hello?your_name=from-query", "application/json",
		strings.NewReader(`{"yourName":"from-body"}`), greeting+`from-body"}`)

	// an injected value was verified by the server, so a body cannot override it
	testPostWithContentType(t, base+"/helloinject?your_name=from-query", "application/json",
		strings.NewReader(`{"yourName":"from-body"}`), greeting+`from-server"}`)

	// nor can a query value
	testGet(t, base+"/helloinject?your_name=from-query", greeting+`from-server"}`)

	// a JSON request used to ignore the URL completely
	testPostWithContentType(t, base+"/hello?your_name=from-query", "application/json",
		strings.NewReader(`{}`), greeting+`from-query"}`)
}

func overwriteProto() {
	writeFileWithTemplate(
		turbo.GetWD()+"/testcreateservice/testcreateservice.proto",
		`syntax = "proto3";
import "shared.proto";
package proto;
option go_package = "/;proto";

message SayHelloRequest {
    CommonValues values = 1;
    string yourName = 2;
    int64 int64Value = 3;
    bool boolValue = 4;
    double float64Value = 5;
    uint64 uint64Value = 6;
}

message SayHelloResponse {
    string message = 1;
}

service TestCreateService {
    rpc sayHello (SayHelloRequest) returns (SayHelloResponse) {}
}
`,
		nil,
	)
	writeFileWithTemplate(
		turbo.GetWD()+"/testcreateservice/shared.proto",
		`syntax = "proto3";
package proto;
option go_package = "/;proto";

message CommonValues {
    int64 someId = 1;
}
`,
		nil,
	)
}

func overwriteThrift() {
	writeFileWithTemplate(
		turbo.GetWD()+"/testcreateservice/shared.thrift",
		`namespace go gen

struct CommonValues {
  1: i64 transactionId,
}

struct HelloValues {
  1: string message,
}
`,
		nil,
	)
	writeFileWithTemplate(
		turbo.GetWD()+"/testcreateservice/testcreateservice.thrift",
		`namespace go gen
include "shared.thrift"

struct SayHelloResponse {
  1: string message,
}

service TestCreateService {
    SayHelloResponse sayHello (1:shared.CommonValues values, 2:string yourName, 3:i64 int64Value, 4:bool boolValue, 5:double float64Value, 6:i64 uint64Value)
}
`,
		nil,
	)

	writeFileWithTemplate(
		turbo.GetWD()+"/testcreateservice/thriftservice/impl/testcreateserviceimpl.go",
		`package impl

import (
	"context"
	"github.com/vaporz/turbo/test/testcreateservice/gen/thrift/gen-go/gen"
	"github.com/apache/thrift/lib/go/thrift"
)

func TProcessor() map[string]thrift.TProcessor {
	return map[string]thrift.TProcessor{
		"TestCreateService": gen.NewTestCreateServiceProcessor(TestCreateService{}),
	}
}


type TestCreateService struct {
}

func (s TestCreateService) SayHello(ctx context.Context, values *gen.CommonValues, yourName string, int64Value int64, boolValue bool, float64Value float64, uint64Value int64) (r *gen.SayHelloResponse, err error) {
	return &gen.SayHelloResponse{Message: "[thrift server]Hello, " + yourName}, nil
}
`,
		nil,
	)

}

func create(t *testing.T, rpc string) {
	cmd.RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testcreateservice"})
	err := cmd.Execute()
	assert.Equal(t, "invalid args", err.Error())

	cmd.RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testcreateservice", "test_create_service"})
	err = cmd.Execute()
	assert.Contains(t, err.Error(), "not a CamelCase string")

	cmd.RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testcreateservice", "TestCreateService", "-r", "aaa"})
	err = cmd.Execute()
	assert.Contains(t, err.Error(), "invalid value for -r, should be grpc or thrift")

	cmd.RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testcreateservice", "TestCreateService", "-r", rpc, "-f", "true",
		"-p", "../../../../"})
	err = cmd.Execute()
	assert.Nil(t, err)
	cmd.RpcType = ""
	cmd.FilePaths = []string{}
}

func generate(t *testing.T, rpc string) {
	cmd.RootCmd.SetArgs([]string{"generate"})
	err := cmd.Execute()
	assert.Equal(t, "Usage: generate [package_path] -r [grpc|thrift] -I (absolute_paths_to_proto|thrift_files)",
		err.Error())

	cmd.RootCmd.SetArgs([]string{"generate", "github.com/vaporz/turbo/test/testcreateservice"})
	err = cmd.Execute()
	assert.Equal(t, "missing rpctype (-r)", err.Error())

	cmd.RootCmd.SetArgs([]string{"generate", "github.com/vaporz/turbo/test/testcreateservice", "-r", "unknown"})
	err = cmd.Execute()
	assert.Equal(t, "invalid rpctype", err.Error())

	if rpc == "grpc" {
		cmd.RootCmd.SetArgs([]string{"generate", "github.com/vaporz/turbo/test/testcreateservice", "-r", rpc})
		err = cmd.Execute()
		assert.Equal(t, "missing .proto file path (-I)", err.Error())
	}

	cmd.RootCmd.SetArgs([]string{"generate", "github.com/vaporz/turbo/test/testcreateservice", "-r", rpc,
		"-I", turbo.GetWD() + "/testcreateservice"})
	err = cmd.Execute()
	assert.Nil(t, err)

	cmd.RpcType = ""
	cmd.FilePaths = []string{}
}

func runCommonTests(t *testing.T, s *turbo.Server, httpPort, rpcType string) {
	testPost(t, "http://localhost:"+httpPort+"/eat?food=banana",
		`{"message":"Yummy!"}`)
	testGet(t, "http://localhost:"+httpPort+"/hello",
		`{"message":"[`+rpcType+` server]Hello, "}`)
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=turbo",
		`{"message":"[`+rpcType+` server]Hello, turbo"}`)
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=turbo&yourname=xxx",
		`{"message":"[`+rpcType+` server]Hello, xxx"}`)
	// a route variable wins over the query whatever spelling the caller used:
	// both of these used to let the query win, purely because yourName and
	// yourname are merged into req.Form under a key the path was not stored in
	testGet(t, "http://localhost:"+httpPort+"/hello/vaporz?yourName=turbo&yourname=xxx",
		`{"message":"[`+rpcType+` server]Hello, vaporz"}`)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`{"message":"[`+rpcType+` server]Hello, testtest"}`)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?your_name=aaa",
		`{"message":"[`+rpcType+` server]Hello, testtest"}`)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?YOURNAME=aaa",
		`{"message":"[`+rpcType+` server]Hello, testtest"}`)
	testPost(t, "http://localhost:"+httpPort+"/hello/testtest",
		"") // 405 Method Not Allowed

	s.Components.SetCommonInterceptor(component(s, "Test1Interceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`test1_intercepted:{"message":"[`+rpcType+` server]Hello, testtest"}`)

	s.Components.Reset()
	s.Components.Intercept([]string{"GET"}, "/", component(s, "TestInterceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?yourName=testname",
		`intercepted:{"message":"[`+rpcType+` server]Hello, testtest"}`)

	s.Components.Reset()
	s.Components.Intercept([]string{"GET"}, "/", component(s, "TestInterceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`intercepted:{"message":"[`+rpcType+` server]Hello, testtest"}`)

	s.Components.Reset()
	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "BeforeErrorInterceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"interceptor_error:error!\n")

	s.Components.Reset()
	list := turbo.Interceptors{component(s, "BaseInterceptor").(turbo.Interceptor), component(s, "BeforeErrorInterceptor").(turbo.Interceptor)}
	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", list...)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"interceptor_error:error!\n")

	s.Components.Reset()
	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "TestInterceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`intercepted:{"message":"[`+rpcType+` server]Hello, testtest"}`)

	s.Components.Reset()
	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "TestInterceptor").(turbo.Interceptor), component(s, "Test1Interceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`intercepted:test1_intercepted:{"message":"[`+rpcType+` server]Hello, testtest"}`)

	s.Components.Reset()
	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "TestInterceptor").(turbo.Interceptor), component(s, "AfterErrorInterceptor").(turbo.Interceptor), component(s, "Test1Interceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`intercepted:test1_intercepted:{"message":"[`+rpcType+` server]Hello, testtest"}:after_error:`)

	s.Components.Reset()
	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "TestInterceptor").(turbo.Interceptor), component(s, "BeforeErrorInterceptor").(turbo.Interceptor), component(s, "Test1Interceptor").(turbo.Interceptor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:interceptor_error:error!\n")

	s.Components.Reset()
	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "TestInterceptor").(turbo.Interceptor))
	s.Components.SetPreprocessor([]string{}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "errorPreProcessor").(turbo.Preprocessor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:error_preprocessor:turbo: encounter error in preprocessor for /hello/testtest, error: error in preprocessor\n")

	s.Components.Reset()
	s.Components.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "TestInterceptor").(turbo.Interceptor))
	s.Components.SetPreprocessor([]string{}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "preProcessor").(turbo.Preprocessor))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`intercepted:preprocessor:{"message":"[`+rpcType+` server]Hello, testtest"}`)

	if rpcType == "thrift" {
		s.Components.SetPostprocessor([]string{}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "thriftPostProcessor").(turbo.Postprocessor))
	} else {
		s.Components.SetPostprocessor([]string{}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "postProcessor").(turbo.Postprocessor))
	}
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		`intercepted:preprocessor:postprocessor:{"message":"[`+rpcType+` server]Hello, testtest"}`)

	s.Components.SetHijacker([]string{}, "/hello/{your_name:[a-zA-Z0-9]+}", component(s, "hijacker").(turbo.Hijacker))
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:hijacker")
	s.Components.Reset()
}

func testPostWithContentType(t *testing.T, url, contentType string, body io.Reader, expected string) {
	resp, err := http.Post(url, contentType, body)
	if err != nil {
		t.Fail()
	}
	defer resp.Body.Close()
	assert.Nil(t, err)
	assert.Equal(t, expected, readResp(resp))
}

func testPost(t *testing.T, url, expected string) {
	testPostWithContentType(t, url, "", nil, expected)
}

func readResp(resp *http.Response) string {
	var bytes bytes.Buffer
	bytes.ReadFrom(resp.Body)
	return bytes.String()
}

// TestParameterBindingModesThrift covers the thrift binding path for the form
// branch, which carried the same precedence bug as the grpc one and needed the
// same fix. JSON requests are deliberately not exercised: turbo's thrift JSON
// branch returns a single reflect.Value while the generated switcher indexes one
// per method argument, so a thrift JSON request panics before reaching binding
// at all. That is a separate defect, reported separately.
func TestParameterBindingThrift(t *testing.T) {
	httpPort := "8087"
	cfg := testConfigPath(t)
	overwriteServiceYamlForBinding(cfg, httpPort, "50067")

	s := turbo.NewThriftServer(&testInitializer{}, cfg)
	s.StartThriftService(timpl.TProcessor)
	time.Sleep(time.Millisecond * 500)
	s.StartHTTPServer(tcompoent.ThriftClient, gen.ThriftSwitcher)
	time.Sleep(time.Millisecond * 500)
	defer s.Stop()

	base := "http://localhost:" + httpPort
	greeting := `{"message":"[thrift server]Hello, `

	// both the query and a form body reach the binding path
	testGet(t, base+"/hello?your_name=from-query", greeting+`from-query"}`)
	testPostWithContentType(t, base+"/hello", "application/x-www-form-urlencoded",
		strings.NewReader("your_name=from-form"), greeting+`from-form"}`)

	// an injected value wins over both
	testGet(t, base+"/helloinject?your_name=from-query", greeting+`from-server"}`)
	testPostWithContentType(t, base+"/helloinject", "application/x-www-form-urlencoded",
		strings.NewReader("your_name=from-form"), greeting+`from-server"}`)
}

// TestPathWinsOverQueryWhateverTheSpelling pins T18. A route variable used to
// lose to a query parameter whenever the caller spelled the query differently
// from the route, because both end up in req.Form under one particular key and
// binding probed that key before the other. Which source wins is now decided by
// the source, not by the spelling.
func TestPathWinsOverQueryWhateverTheSpelling(t *testing.T) {
	httpPort := "8094"
	cfg := testConfigPath(t)
	overwriteServiceYaml(cfg, httpPort, "50074", "development")

	s := turbo.NewGrpcServer(&testInitializer{}, cfg)
	s.StartGrpcService(gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 300)
	s.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Millisecond * 300)
	defer s.Stop()

	base := "http://localhost:" + httpPort
	greeting := `{"message":"[grpc server]Hello, `

	// the route declares {your_Name}; every spelling of the query loses to it
	for _, query := range []string{"your_name", "yourName", "YOURNAME", "YourName"} {
		testGet(t, base+"/hello/vaporz?"+query+"=from-query", greeting+`vaporz"}`)
	}

	// the query is still used when the route carries no such variable
	testGet(t, base+"/hello?yourName=from-query", greeting+`from-query"}`)

	// and a JSON request resolves the same way
	for _, body := range []string{`{}`, `{"yourName":"from-body"}`} {
		req, err := http.NewRequest("GET", base+"/hello/vaporz?yourName=from-query", strings.NewReader(body))
		assert.Nil(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		assert.Nil(t, err)
		assert.Equal(t, greeting+`vaporz"}`, readResp(resp))
		resp.Body.Close()
	}
}

// TestInvalidParameterIsRejected pins T3. A parameter that is present but cannot
// be used is the caller's mistake: leaving the field at its zero value and
// answering 200 makes that request indistinguishable from one that never carried
// the parameter, so the caller has no way to learn.
func TestInvalidParameterIsRejected(t *testing.T) {
	httpPort := "8096"
	cfg := testConfigPath(t)
	overwriteServiceYaml(cfg, httpPort, "50076", "development")

	s := turbo.NewGrpcServer(&testInitializer{}, cfg)
	s.StartGrpcService(gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 300)
	s.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Millisecond * 300)
	defer s.Stop()

	base := "http://localhost:" + httpPort

	// int64_value is an int64; "abc" used to become 0, with a 200 response
	body := testRejectedParameter(t, base+"/hello?int64_value=abc")
	// the message names the field and the value, which is all a caller needs
	assert.Contains(t, body, "Int64Value")
	assert.Contains(t, body, `"abc"`)

	// a usable value still binds, and the request succeeds
	testGet(t, base+"/hello?your_name=ok&int64_value=64", `{"message":"[grpc server]Hello, ok"}`)
}

// TestInvalidParameterIsRejectedThrift covers the thrift binding path, which
// swallowed the same failure inside BuildArgs.
func TestInvalidParameterIsRejectedThrift(t *testing.T) {
	httpPort := "8097"
	cfg := testConfigPath(t)
	overwriteServiceYaml(cfg, httpPort, "50077", "development")

	s := turbo.NewThriftServer(&testInitializer{}, cfg)
	s.StartThriftService(timpl.TProcessor)
	time.Sleep(time.Millisecond * 500)
	s.StartHTTPServer(tcompoent.ThriftClient, gen.ThriftSwitcher)
	time.Sleep(time.Millisecond * 500)
	defer s.Stop()

	base := "http://localhost:" + httpPort
	testRejectedParameter(t, base+"/hello?int64_value=abc")
	testGet(t, base+"/hello?your_name=ok&int64_value=64", `{"message":"[thrift server]Hello, ok"}`)
}

// testRejectedParameter asserts that a parameter the server cannot use is refused
// as a client error, and returns the message so a caller can check that it says
// enough to fix the request.
func testRejectedParameter(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := readResp(resp)
	resp.Body.Close()
	assert.Contains(t, body, "turbo: cannot bind")
	return body
}

// TestRouteTableAndNotFoundAreObservable pins T12. Two things used to be
// invisible: which routes a router actually serves, and a request that matched
// none of them. Both matter exactly when something in front of the service is
// dropping requests -- then "it never arrived" and "it arrived and was dropped"
// look identical from the outside.
func TestRouteTableAndNotFoundAreObservable(t *testing.T) {
	httpPort := "8089"
	cfg := testConfigPath(t)
	writeRawConfig(t, cfg, rawServiceYaml(httpPort, "50069", ""))

	s := turbo.NewGrpcServer(&testInitializer{}, cfg)
	s.StartGrpcService(gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 300)

	// capture the log before the HTTP server is started, because that is when the
	// routing table is reported
	logged := &syncBuffer{}
	turbo.SetOutput(logged)
	defer turbo.SetOutput(os.Stdout)

	s.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Millisecond * 300)
	defer s.Stop()

	base := "http://localhost:" + httpPort
	testGet(t, base+"/hello?your_name=world", `{"message":"[grpc server]Hello, world"}`)

	assert.Contains(t, logged.String(), "route: GET /hello -> TestService.SayHello")
	assert.Contains(t, logged.String(), "route: POST /hello -> TestService.SayHello")
	assert.Contains(t, logged.String(), "turbo: 2 route(s) registered")

	// a request nobody handles is reported, and still answered the same way
	resp, err := http.Get(base + "/no/such/path?token=secret")
	assert.Nil(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "404 page not found\n", readResp(resp))
	resp.Body.Close()

	reported := logged.String()
	assert.Contains(t, reported, "404 no route for GET /no/such/path")
	assert.NotContains(t, reported, "token=secret", "the query must not be logged")
}

// syncBuffer collects log output written by request handlers, which run in their
// own goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestErrorStatusCodes pins T9. The HTTP status is what everything in front of a
// service reads -- a gateway, an alert rule, a dashboard -- and answering every
// failure with 500 makes a caller's mistake look like a broken service. An error
// can now carry the status it deserves, and the default handler honours it.
func TestErrorStatusCodes(t *testing.T) {
	httpPort := "8091"
	cfg := testConfigPath(t)
	overwriteServiceYaml(cfg, httpPort, "50071", "development")

	s := turbo.NewGrpcServer(&testInitializer{}, cfg)
	s.StartGrpcService(gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 300)
	s.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Millisecond * 300)
	defer s.Stop()

	base := "http://localhost:" + httpPort

	// a service failure that says nothing about the status is still a 500, and
	// the body is unchanged
	resp, err := http.Get(base + "/hello/error")
	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "rpc error: code = Unknown desc = grpc error\n", readResp(resp))
	resp.Body.Close()

	// a handler that knows what went wrong can say so
	s.Components.SetPreprocessor([]string{"GET"}, "/hello", statusPreProcessor)
	resp, err = http.Get(base + "/hello?your_name=x")
	assert.Nil(t, err)
	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
	assert.Equal(t, "turbo: encounter error in preprocessor for /hello?your_name=x, error: teapot\n", readResp(resp))
	resp.Body.Close()

	// the first matching declaration is the one that runs, so the next case needs
	// a clean component set to be reached at all
	s.Components.Reset()

	// an error that says nothing keeps falling back to 500
	s.Components.SetPreprocessor([]string{"GET"}, "/hello", plainErrorPreProcessor)
	resp, err = http.Get(base + "/hello?your_name=x")
	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	resp.Body.Close()

	s.Components.Reset()

	// a body the server cannot use is the caller's mistake, not a service failure
	resp, err = http.Post(base+"/hello", "application/json", strings.NewReader("{aaaaa"))
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := readResp(resp)
	resp.Body.Close()
	assert.Contains(t, body, "turbo: failed to BuildRequest for json api")
	assert.Contains(t, body, "invalid character 'a'")
}

// TestRouteAuditRefusesUnprotectedRoutes pins T4②. Declaring which interceptors
// authenticate a request turns the route audit into an enforcement: a
// configuration that would serve a route without one is refused. At startup that
// means the server does not come up; on reload it means the running
// configuration stays exactly as it was.
func TestRouteAuditRefusesUnprotectedRoutes(t *testing.T) {
	httpPort := "8090"
	cfg := testConfigPath(t)
	auth := "auth:\n  interceptors:\n    - TestInterceptor\n"
	writeRawConfig(t, cfg, rawServiceYaml(httpPort, "50070",
		"interceptor:\n  - GET /hello TestInterceptor\n  - POST /hello TestInterceptor\n")+auth)

	s := turbo.NewGrpcServer(&testInitializer{}, cfg)
	s.StartGrpcService(gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 300)

	logged := &syncBuffer{}
	turbo.SetOutput(logged)
	defer turbo.SetOutput(os.Stdout)

	s.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Millisecond * 300)
	defer s.Stop()

	base := "http://localhost:" + httpPort
	guarded := `intercepted:{"message":"[grpc server]Hello, ok"}`
	testGet(t, base+"/hello?your_name=ok", guarded)
	assert.Contains(t, logged.String(), "route audit: GET /hello -> TestService.SayHello")
	assert.Contains(t, logged.String(), "authenticated")

	// an interceptor that is not declared as an auth interceptor leaves the route
	// effectively unprotected, so this configuration is refused
	writeRawConfig(t, cfg, rawServiceYaml(httpPort, "50070",
		"interceptor:\n  - GET /hello Test1Interceptor\n  - POST /hello Test1Interceptor\n")+auth)
	time.Sleep(time.Millisecond * 800)
	testGet(t, base+"/hello?your_name=ok", guarded)
	assert.Contains(t, logged.String(), "refusing this configuration")
	assert.Contains(t, logged.String(), "UNPROTECTED")

	// and so is one that forgot the interceptor line altogether, which is the
	// mistake this audit exists for
	writeRawConfig(t, cfg, rawServiceYaml(httpPort, "50070", "")+auth)
	time.Sleep(time.Millisecond * 800)
	testGet(t, base+"/hello?your_name=ok", guarded)

	// at startup the same violation means the server does not come up at all
	refusedCfg := testConfigPath(t)
	writeRawConfig(t, refusedCfg, rawServiceYaml("8092", "50072", "")+auth)
	refused := turbo.NewGrpcServer(&testInitializer{}, refusedCfg)
	assert.Panics(t, func() {
		refused.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	})
}

// TestFailedConfigReloadKeepsServing pins T14: a configuration change that cannot
// be loaded must not be able to take down a server that is already serving. Both
// ways a reload used to fail are covered, because each was fatal in its own way:
// a component that was never registered left the server holding no components at
// all, and a file that does not parse panicked inside the watcher goroutine,
// where nothing recovered it.
func TestFailedConfigReloadKeepsServing(t *testing.T) {
	httpPort := "8088"
	cfg := testConfigPath(t)
	writeRawConfig(t, cfg, rawServiceYaml(httpPort, "50068", ""))

	s := turbo.NewGrpcServer(&testInitializer{}, cfg)
	s.StartGrpcService(gimpl.RegisterServer)
	time.Sleep(time.Millisecond * 300)
	s.StartHTTPServer(gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Millisecond * 300)
	defer s.Stop()

	base := "http://localhost:" + httpPort
	before := `{"message":"[grpc server]Hello, before"}`
	testGet(t, base+"/hello?your_name=before", before)

	// a reload that names a component nobody registered must be rejected whole
	writeRawConfig(t, cfg, rawServiceYaml(httpPort, "50068",
		"interceptor:\n  - GET /hello NoSuchInterceptor\n"))
	time.Sleep(time.Millisecond * 500)

	// so must a file that does not parse at all
	writeRawConfig(t, cfg, "config: [this is not\n  - valid yaml")
	time.Sleep(time.Millisecond * 500)

	// the running server never lost the configuration it already had
	testGet(t, base+"/hello?your_name=before", before)

	// and the reloader is still alive rather than wedged by the failures
	writeRawConfig(t, cfg, rawServiceYaml(httpPort, "50068",
		"preprocessor:\n  - GET /hello preProcessor\n"))
	testGetEventually(t, base+"/hello?your_name=after",
		`preprocessor:{"message":"[grpc server]Hello, after"}`, time.Second*10)
}

func testGet(t *testing.T, url, expected string) {
	resp, err := http.Get(url)
	if err != nil {
		t.Fail()
	}
	defer resp.Body.Close()
	assert.Nil(t, err)
	assert.Equal(t, expected, readResp(resp))
}

// testGetEventually polls url until its response body equals expected, then
// returns. Config hot reloading is asynchronous: an fsnotify event triggers a
// viper callback, which hands the new config to a reload goroutine that finally
// swaps the router. Asserting after a fixed sleep therefore races with that
// chain -- on slow filesystems (e.g. WSL DrvFs) the swap can take longer than
// the sleep, and the assertion observes the previous routing table instead.
// Polling also tolerates the intermediate states produced while the config file
// is being rewritten. It fails only if the expectation never materializes.
func testGetEventually(t *testing.T, url, expected string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var actual string
	for {
		resp, err := http.Get(url)
		if err == nil {
			actual = readResp(resp)
			resp.Body.Close()
			if actual == expected {
				return
			}
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(time.Millisecond * 50)
	}
	assert.Equal(t, expected, actual, "config hot reload did not take effect within %s", timeout)
}

type testInitializer struct {
}

func (t *testInitializer) InitService(s turbo.Servable) error {
	s.RegisterComponent("BaseInterceptor", &turbo.BaseInterceptor{})
	s.RegisterComponent("BeforeErrorInterceptor", &BeforeErrorInterceptor{})
	s.RegisterComponent("AfterErrorInterceptor", &AfterErrorInterceptor{})
	s.RegisterComponent("TestInterceptor", &TestInterceptor{})
	s.RegisterComponent("Test1Interceptor", &Test1Interceptor{})
	s.RegisterComponent("ContextValueInterceptor", &ContextValueInterceptor{})
	s.RegisterComponent("InjectInterceptor", &InjectInterceptor{})
	s.RegisterComponent("MetadataInterceptor", &MetadataInterceptor{})
	s.RegisterComponent("preProcessor", preProcessor)
	s.RegisterComponent("errorPreProcessor", errorPreProcessor)
	s.RegisterComponent("postProcessor", postProcessor)
	s.RegisterComponent("thriftPostProcessor", thriftPostProcessor)
	s.RegisterComponent("hijacker", hijacker)
	s.RegisterComponent("errorHandler", errorHandler)
	s.RegisterComponent("convertProtoCommonValues", convertProtoCommonValues)
	s.RegisterComponent("convertProtoSayHelloRequest", convertProtoSayHelloRequest)
	s.RegisterComponent("convertThriftCommonValues", convertThriftCommonValues)
	return nil
}

func (t *testInitializer) StopService(s turbo.Servable) {
}

type BeforeErrorInterceptor struct {
	turbo.BaseInterceptor
}

func (l *BeforeErrorInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	resp.Write([]byte("interceptor_error:"))
	return errors.New("error!")
}

type AfterErrorInterceptor struct {
	turbo.BaseInterceptor
}

func (l *AfterErrorInterceptor) After(resp http.ResponseWriter, req *http.Request) error {
	fmt.Println("[After] Request URL:" + req.URL.Path)
	resp.Write([]byte(":after_error:"))
	return errors.New("error: after interceptor")
}

type TestInterceptor struct {
	turbo.BaseInterceptor
}

func (l *TestInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	fmt.Println("TestInterceptor before")
	resp.Write([]byte("intercepted:"))
	return nil
}

func (l *TestInterceptor) After(resp http.ResponseWriter, req *http.Request) error {
	fmt.Println("[After] Request URL:" + req.URL.Path)
	return nil
}

type Test1Interceptor struct {
	turbo.BaseInterceptor
}

func (l *Test1Interceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	resp.Write([]byte("test1_intercepted:"))
	return nil
}

func (l *Test1Interceptor) After(resp http.ResponseWriter, req *http.Request) error {
	fmt.Println("[After] Request URL:" + req.URL.Path)
	return nil
}

type ContextValueInterceptor struct {
	turbo.BaseInterceptor
}

func (l *ContextValueInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	ctx := req.Context()
	fmt.Println("set context!!")
	ctx = context.WithValue(ctx, "bool_value", "true")
	ctx = context.WithValue(ctx, "Int64Value", "1234567")
	ctx = context.WithValue(ctx, "float64_value", "1.23")
	ctx = context.WithValue(ctx, "uint64value", "456")
	resp.Write([]byte("test1_intercepted:"))
	*req = *req.WithContext(ctx)
	return nil
}

// InjectInterceptor stores a value the server itself established, the way a
// real service verifies a signature or resolves a device code before binding.
type InjectInterceptor struct {
	turbo.BaseInterceptor
}

func (i *InjectInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	turbo.InjectParam(req, "your_Name", "from-server")
	return nil
}

type MetadataInterceptor struct {
	turbo.BaseInterceptor
}

func (m *MetadataInterceptor) After(resp http.ResponseWriter, req *http.Request) error {
	ctx := req.Context()
	resp.Write([]byte("metadata:header:" + (*turbo.GrpcMetadataHeader(ctx))["header-key"][0] +
		":trailer:" + (*turbo.GrpcMetadataTrailer(ctx))["trailer-key"][0] +
		":peer:" + (*turbo.GrpcMetadataPeer(ctx)).Addr.String()))
	return nil
}

var preProcessor turbo.Preprocessor = func(resp http.ResponseWriter, req *http.Request) error {
	resp.Write([]byte("preprocessor:"))
	return nil
}

var errorPreProcessor turbo.Preprocessor = func(resp http.ResponseWriter, req *http.Request) error {
	resp.Write([]byte("error_preprocessor:"))
	return errors.New("error in preprocessor")
}

// statusPreProcessor fails with an error that says which status the response
// should carry; plainErrorPreProcessor fails without saying anything.
var statusPreProcessor turbo.Preprocessor = func(resp http.ResponseWriter, req *http.Request) error {
	return turbo.Errorf(http.StatusTeapot, "teapot")
}

var plainErrorPreProcessor turbo.Preprocessor = func(resp http.ResponseWriter, req *http.Request) error {
	return errors.New("plain failure")
}

var postProcessor turbo.Postprocessor = func(resp http.ResponseWriter, req *http.Request, serviceResp interface{}, err error) error {
	resp.Write([]byte("postprocessor:"))
	return nil
}

var thriftPostProcessor turbo.Postprocessor = func(resp http.ResponseWriter, req *http.Request, serviceResp interface{}, err error) error {
	resp.Write([]byte("postprocessor:"))
	return nil
}

var hijacker turbo.Hijacker = func(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte("hijacker"))
}

var errorHandler turbo.ErrorHandlerFunc = func(resp http.ResponseWriter, req *http.Request, err error) {
	resp.Write([]byte("from errorHandler:" + err.Error()))
}

var convertProtoCommonValues turbo.Convertor = func(req *http.Request) reflect.Value {
	result := &proto.CommonValues{}
	result.SomeId = 1111111
	return reflect.ValueOf(result)
}

var convertProtoSayHelloRequest turbo.Convertor = func(req *http.Request) reflect.Value {
	result := &proto.SayHelloRequest{}
	result.YourName = "from convertor"
	return reflect.ValueOf(result)
}

var convertThriftCommonValues turbo.Convertor = func(req *http.Request) reflect.Value {
	result := &tgen.CommonValues{}
	result.TransactionId = 222222
	return reflect.ValueOf(result)
}

// testConfigPath returns a config file path used by this test alone.
//
// Every server started during the suite installs a config file watcher, and a
// server's watcher is never torn down -- neither by Stop() nor by the reload
// that replaces its Config. When all tests share a single config file, every
// rewrite therefore fires a hot reload in every server that was ever started,
// including servers that were already stopped and whose components are gone:
// those reloads panic, loadComponentsNoPanic recovers and leaves that server's
// Components nil, and the running server that owns the file can end up never
// observing its own reload. A per-test file keeps the tests isolated (and keeps
// the checked-in testservice/service.yaml from being rewritten by a test run).
//
// The file lives in t.TempDir() rather than under testservice/, because the hot
// reload under test is driven by inotify: on a DrvFs mount (for example a repo
// checked out under /mnt/d in WSL) events are dropped, and the watcher then
// never reports the change at all.
func testConfigPath(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "service_*.yaml")
	if err != nil {
		t.Fatalf("cannot create config file: %v", err)
	}
	path := f.Name()
	f.Close()
	return path
}

func overwriteServiceYaml(file, httpPort, servicePort, env string) {
	type serviceYamlValues struct {
		HttpPort    string
		ServicePort string
		Env         string
	}
	writeFileWithTemplate(
		file,
		`config:
  file_root_path: /src
  package_path: github.com/vaporz/turbo/test/testservice
  http_port: {{.HttpPort}}
  environment: {{.Env}}
  turbo_log_path: 
  grpc_service_name: TestService,MinionsService
  grpc_service_host: 127.0.0.1
  grpc_service_port: {{.ServicePort}}
  thrift_service_name: TestService,MinionsService
  thrift_service_host: 127.0.0.1
  thrift_service_port: {{.ServicePort}}

urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TestService SayHello
  - GET,POST /hello TestService SayHello
  - POST /testjson TestService TestJson
  - POST /testjson/{StringValue:[a-zA-Z0-9]+}/{int32_value:[a-zA-Z0-9]+} TestService TestJson
  - POST /eat MinionsService Eat
`,
		serviceYamlValues{
			HttpPort:    httpPort,
			ServicePort: servicePort,
			Env:         env,
		},
	)
}

func overwriteServiceYamlWithGrpcComponents(file, httpPort, servicePort, env string) {
	type serviceYamlValues struct {
		HttpPort    string
		ServiceName string
		ServicePort string
		Env         string
	}
	writeFileWithTemplate(
		file,
		`config:
  file_root_path: /src
  package_path: github.com/vaporz/turbo/test/testservice
  http_port: {{.HttpPort}}
  environment: {{.Env}}
  turbo_log_path:
  grpc_service_name: {{.ServiceName}}
  grpc_service_host: 127.0.0.1
  grpc_service_port: {{.ServicePort}}
  thrift_service_name: {{.ServiceName}}
  thrift_service_host: 127.0.0.1
  thrift_service_port: {{.ServicePort}}

urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} {{.ServiceName}} SayHello
  - GET /hello {{.ServiceName}} SayHello
  - GET /hellointerceptor {{.ServiceName}} SayHello
  - GET /hello_preprocessor {{.ServiceName}} SayHello
  - GET /hello_postprocessor {{.ServiceName}} SayHello
  - GET /hello_hijacker {{.ServiceName}} SayHello
  - GET /hello_convertor {{.ServiceName}} SayHello
  - POST /testjson {{.ServiceName}} TestJson
  - POST /testjson/{StringValue:[a-zA-Z0-9]+}/{int32_value:[a-zA-Z0-9]+} {{.ServiceName}} TestJson

interceptor:
  - GET /hello TestInterceptor
  - GET /hellointerceptor BeforeErrorInterceptor,Test1Interceptor
preprocessor:
  - GET /hello_preprocessor preProcessor
postprocessor:
  - GET /hello_postprocessor postProcessor
hijacker:
  - GET /hello_hijacker hijacker
convertor:
  - CommonValues convertProtoCommonValues
errorhandler: errorHandler
`,
		serviceYamlValues{
			HttpPort:    httpPort,
			ServiceName: "TestService",
			ServicePort: servicePort,
			Env:         env,
		},
	)
}

// writeRawConfig writes content to path as is, so a test can describe a
// configuration that is invalid on purpose.
func writeRawConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("cannot write %s: %v", path, err)
	}
}

// rawServiceYaml renders a service.yaml for the test service whose component
// sections are supplied verbatim by the caller. It is not a text/template on
// purpose: an invalid configuration has to survive being written unchanged.
func rawServiceYaml(httpPort, servicePort, components string) string {
	return `config:
  file_root_path: /src
  package_path: github.com/vaporz/turbo/test/testservice
  http_port: ` + httpPort + `
  environment: development
  turbo_log_path:
  grpc_service_name: TestService
  grpc_service_host: 127.0.0.1
  grpc_service_port: ` + servicePort + `
  thrift_service_name: TestService
  thrift_service_host: 127.0.0.1
  thrift_service_port: ` + servicePort + `

urlmapping:
  - GET /hello TestService SayHello
  - POST /hello TestService SayHello

` + components
}

// overwriteServiceYamlForBinding writes a config that routes /helloinject
// through an interceptor which injects a value.
func overwriteServiceYamlForBinding(file, httpPort, servicePort string) {
	type serviceYamlValues struct {
		HttpPort    string
		ServicePort string
	}
	writeFileWithTemplate(
		file,
		`config:
  file_root_path: /src
  package_path: github.com/vaporz/turbo/test/testservice
  http_port: {{.HttpPort}}
  environment: development
  turbo_log_path:
  grpc_service_name: TestService
  grpc_service_host: 127.0.0.1
  grpc_service_port: {{.ServicePort}}
  thrift_service_name: TestService
  thrift_service_host: 127.0.0.1
  thrift_service_port: {{.ServicePort}}

urlmapping:
  - GET /hello TestService SayHello
  - POST /hello TestService SayHello
  - GET /helloinject TestService SayHello
  - POST /helloinject TestService SayHello

interceptor:
  - GET /helloinject InjectInterceptor
  - POST /helloinject InjectInterceptor
`,
		serviceYamlValues{
			HttpPort:    httpPort,
			ServicePort: servicePort,
		},
	)
}

func changeServiceYamlWithGrpcComponents(file, httpPort, servicePort, env string) {
	type serviceYamlValues struct {
		HttpPort    string
		ServiceName string
		ServicePort string
		Env         string
	}
	writeFileWithTemplate(
		file,
		`config:
  file_root_path: /src
  package_path: github.com/vaporz/turbo/test/testservice
  http_port: {{.HttpPort}}
  environment: {{.Env}}
  turbo_log_path: 
  grpc_service_name: {{.ServiceName}}
  grpc_service_host: 127.0.0.1
  grpc_service_port: {{.ServicePort}}
  thrift_service_name: {{.ServiceName}}
  thrift_service_host: 127.0.0.1
  thrift_service_port: {{.ServicePort}}

urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} {{.ServiceName}} SayHello
  - GET /hello {{.ServiceName}} SayHello
  - GET /hellointerceptor {{.ServiceName}} SayHello
  - GET /hello_preprocessor {{.ServiceName}} SayHello
  - GET /hello_postprocessor {{.ServiceName}} SayHello
  - GET /hello_hijacker {{.ServiceName}} SayHello
  - GET /hello_convertor {{.ServiceName}} SayHello
  - POST /testjson {{.ServiceName}} TestJson
  - POST /testjson/{StringValue:[a-zA-Z0-9]+}/{int32_value:[a-zA-Z0-9]+} {{.ServiceName}} TestJson

interceptor:
  - GET /hello Test1Interceptor
preprocessor:
  - GET /hello preProcessor
postprocessor:
  - GET /hello postProcessor
`,
		serviceYamlValues{
			HttpPort:    httpPort,
			ServiceName: "TestService",
			ServicePort: servicePort,
			Env:         env,
		},
	)
}

func writeFileWithTemplate(filePath, text string, data interface{}) {
	f, err := os.Create(filePath)
	if err != nil {
		panic("fail to create file:" + filePath)
	}
	bf := bufio.NewWriter(f)
	tmpl, err := template.New("").Parse(text)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(bf, data)
	if err != nil {
		panic(err)
	}
	bf.Flush()
}
