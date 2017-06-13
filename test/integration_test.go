package test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/turbo/cmd"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test/testservice/gen/proto"
	tgen "github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
	gcomponent "github.com/vaporz/turbo/test/testservice/grpcapi/component"
	gimpl "github.com/vaporz/turbo/test/testservice/grpcservice/impl"
	tcompoent "github.com/vaporz/turbo/test/testservice/thriftapi/component"
	timpl "github.com/vaporz/turbo/test/testservice/thriftservice/impl"
	"os"
	"net/http"
	"time"
	"fmt"
	"errors"
	"context"
	"reflect"
	"bytes"
)

func TestMain(m *testing.M) {
	turbo.InitGOPATH()
	os.RemoveAll(turbo.GOPATH + "/src/github.com/vaporz/turbo/test/testcreateservice")
	os.Exit(m.Run())
}

func TestCreateGrpcService(t *testing.T) {
	create(t, "grpc")
	generate(t, "grpc")
	writeProto()
	os.RemoveAll(turbo.GOPATH + "/src/github.com/vaporz/turbo/test/testcreateservice/gen")
	generate(t, "grpc")
}

func TestCreateThriftService(t *testing.T) {
	create(t, "thrift")
	generate(t, "thrift")
	writeThrift()
	os.RemoveAll(turbo.GOPATH + "/src/github.com/vaporz/turbo/test/testcreateservice/gen")
	generate(t, "thrift")
}

func TestGrpcService(t *testing.T) {
	httpPort := "8081"
	OverrideServiceYaml("8081", "50051", "development")
	turbo.ResetChans()

	go turbo.StartGRPC("github.com/vaporz/turbo/test/testservice", "service",
		50051, gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)
	time.Sleep(time.Second * 1)

	runCommonTests(t, httpPort, "grpc")

	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", ContextValueInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"test1_intercepted:{\"message\":\"{\\\"values\\\":{},\\\"yourName\\\":\\\"testtest\\\",\\\"int64Value\\\":1234567,\\\"boolValue\\\":true,\\\"float64Value\\\":1.23}\"}")
	ResetComponents()

	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?int64_value=64&bool_value=true&float64_value=0.123&uint64_value=123",
		"{\"message\":\"{\\\"values\\\":{},\\\"yourName\\\":\\\"testtest\\\",\\\"int64Value\\\":64,\\\"boolValue\\\":true,\\\"float64Value\\\":0.123,\\\"uint64Value\\\":123}\"}")

	turbo.RegisterMessageFieldConvertor(new(proto.CommonValues), convertProtoCommonValues)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?bool_value=true",
		"{\"message\":\"{\\\"values\\\":{\\\"someId\\\":1111111},\\\"yourName\\\":\\\"testtest\\\",\\\"boolValue\\\":true}\"}")
	turbo.Stop()
}

func convertProtoCommonValues(req *http.Request) reflect.Value {
	result := &proto.CommonValues{}
	result.SomeId = 1111111
	return reflect.ValueOf(result)
}

func TestThriftService(t *testing.T) {
	httpPort := "8082"
	OverrideServiceYaml(httpPort, "50052", "production")
	turbo.ResetChans()

	go turbo.StartTHRIFT("github.com/vaporz/turbo/test/testservice", "service",
		50052, tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
	time.Sleep(time.Second * 2)
	turbo.SetOutput(os.Stdout)

	runCommonTests(t, httpPort, "thrift")

	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", ContextValueInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"test1_intercepted:{\"message\":\"[thrift server]values.TransactionId=0, yourName=testtest,int64Value=1234567, boolValue=true, float64Value=1.230000, uint64Value=0, int32Value=0, int16Value=0\"}")
	ResetComponents()

	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?transaction_id=111&int64_value=64&bool_value=true&float64_value=0.123&uint64_value=123&int32_value=32&int16_value=16",
		"{\"message\":\"[thrift server]values.TransactionId=111, yourName=testtest,int64Value=64, boolValue=true, float64Value=0.123000, uint64Value=123, int32Value=32, int16Value=16\"}")

	turbo.RegisterMessageFieldConvertor(new(tgen.CommonValues), convertThriftCommonValues)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?bool_value=true",
		"{\"message\":\"[thrift server]values.TransactionId=222222, yourName=testtest,int64Value=0, boolValue=true, float64Value=0.000000, uint64Value=0, int32Value=0, int16Value=0\"}")
	turbo.Stop()
}

func convertThriftCommonValues(req *http.Request) reflect.Value {
	result := &tgen.CommonValues{}
	result.TransactionId = 222222
	return reflect.ValueOf(result)
}

func writeProto() {
	writeFileWithTemplate(
		turbo.GOPATH+"/src/github.com/vaporz/turbo/test/testcreateservice/testcreateservice.proto",
		`syntax = "proto3";
import "shared.proto";
package proto;

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

service TestService {
    rpc sayHello (SayHelloRequest) returns (SayHelloResponse) {}
}
`,
		nil,
	)
	writeFileWithTemplate(
		turbo.GOPATH+"/src/github.com/vaporz/turbo/test/testcreateservice/shared.proto",
		`syntax = "proto3";
package proto;

message CommonValues {
    int64 someId = 1;
}
`,
		nil,
	)
}

func writeThrift() {
	writeFileWithTemplate(
		turbo.GOPATH+"/src/github.com/vaporz/turbo/test/testcreateservice/shared.thrift",
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
		turbo.GOPATH+"/src/github.com/vaporz/turbo/test/testcreateservice/testcreateservice.thrift",
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
}

func create(t *testing.T, rpc string) {
	cmd.RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testcreateservice"})
	err := cmd.Execute()
	assert.Equal(t, "invalid args", err.Error())

	cmd.RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testcreateservice", "test_create_service"})
	err = cmd.Execute()
	assert.Contains(t, err.Error(), "not a CamelCase string")

	cmd.RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testcreateservice", "TestCreateService", "-r", rpc, "-f", "true"})
	err = cmd.Execute()
	assert.Nil(t, err)
	cmd.RpcType = ""
	cmd.FilePaths = []string{}
}

func generate(t *testing.T, rpc string) {
	cmd.RootCmd.SetArgs([]string{"generate"})
	err := cmd.Execute()
	assert.Equal(t, "Usage: generate [package_path]", err.Error())

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
								 "-I", turbo.TurboRootPath + "/test/testcreateservice"})
	err = cmd.Execute()
	assert.Nil(t, err)

	cmd.RpcType = ""
	cmd.FilePaths = []string{}
}

func runCommonTests(t *testing.T, httpPort, rpcType string) {
	testGet(t, "http://localhost:"+httpPort+"/hello/vaporz?yourName=turbo&yourname=xxx",
		"{\"message\":\"["+rpcType+" server]Hello, vaporz\"}")
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"{\"message\":\"["+rpcType+" server]Hello, testtest\"}")
	testPost(t, "http://localhost:"+httpPort+"/hello/testtest",
		"404 page not found\n")

	turbo.SetCommonInterceptor(Test1Interceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"test1_intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	// TODO test errorHandler
	turbo.WithErrorHandler(errorHandler)

	ResetComponents()
	turbo.Intercept([]string{"GET"}, "/", TestInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest?yourName=testname",
		"intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	ResetComponents()
	turbo.Intercept([]string{"GET"}, "/", TestInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	ResetComponents()
	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", BeforeErrorInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"interceptor_error:")

	ResetComponents()
	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", turbo.BaseInterceptor{}, BeforeErrorInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"interceptor_error:")

	ResetComponents()
	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", TestInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	ResetComponents()
	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", TestInterceptor{}, Test1Interceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:test1_intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", TestInterceptor{}, AfterErrorInterceptor{}, Test1Interceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:test1_intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	ResetComponents()
	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", TestInterceptor{}, BeforeErrorInterceptor{}, Test1Interceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:interceptor_error:")

	ResetComponents()
	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", TestInterceptor{})
	turbo.SetPreprocessor("/hello/{your_name:[a-zA-Z0-9]+}", errorPreProcessor)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:error_preprocessor:")

	ResetComponents()
	turbo.Intercept([]string{"GET"}, "/hello/{your_name:[a-zA-Z0-9]+}", TestInterceptor{})
	turbo.SetPreprocessor("/hello/{your_name:[a-zA-Z0-9]+}", preProcessor)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:preprocessor:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	if rpcType == "thrift" {
		turbo.SetPostprocessor("/hello/{your_name:[a-zA-Z0-9]+}", thriftPostProcessor)
	} else {
		turbo.SetPostprocessor("/hello/{your_name:[a-zA-Z0-9]+}", postProcessor)
	}
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:preprocessor:postprocessor:["+rpcType+" server]Hello, testtest")

	turbo.SetHijacker("/hello/{your_name:[a-zA-Z0-9]+}", hijacker)
	testGet(t, "http://localhost:"+httpPort+"/hello/testtest",
		"intercepted:hijacker")
	ResetComponents()
}

func testPost(t *testing.T, url, expected string) {
	resp, err := http.Post(url, "", nil)
	defer resp.Body.Close()
	assert.Nil(t, err)
	assert.Equal(t, expected, readResp(resp))
}

func readResp(resp *http.Response) string {
	var bytes bytes.Buffer
	bytes.ReadFrom(resp.Body)
	return bytes.String()
}

func testGet(t *testing.T, url, expected string) {
	resp, err := http.Get(url)
	defer resp.Body.Close()
	assert.Nil(t, err)
	assert.Equal(t, expected, readResp(resp))
}

type BeforeErrorInterceptor struct {
	turbo.BaseInterceptor
}

func (l BeforeErrorInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	resp.Write([]byte("interceptor_error:"))
	return req, errors.New("error!")
}

type AfterErrorInterceptor struct {
	turbo.BaseInterceptor
}

func (l AfterErrorInterceptor) After(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	fmt.Println("[After] Request URL:" + req.URL.Path)
	return req, errors.New("error: after interceptor")
}

type TestInterceptor struct {
	turbo.BaseInterceptor
}

func (l TestInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	fmt.Println("TestInterceptor before")
	resp.Write([]byte("intercepted:"))
	return req, nil
}

func (l TestInterceptor) After(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	fmt.Println("[After] Request URL:" + req.URL.Path)
	return req, nil
}

type Test1Interceptor struct {
	turbo.BaseInterceptor
}

func (l Test1Interceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	resp.Write([]byte("test1_intercepted:"))
	return req, nil
}

func (l Test1Interceptor) After(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	fmt.Println("[After] Request URL:" + req.URL.Path)
	return req, nil
}

type ContextValueInterceptor struct {
	turbo.BaseInterceptor
}

func (l ContextValueInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	ctx := req.Context()
	fmt.Println("set context!!")
	ctx = context.WithValue(ctx, "bool_value", "true")
	ctx = context.WithValue(ctx, "Int64Value", "1234567")
	ctx = context.WithValue(ctx, "float64_value", "1.23")
	resp.Write([]byte("test1_intercepted:"))
	return req.WithContext(ctx), nil
}

func preProcessor(resp http.ResponseWriter, req *http.Request) error {
	resp.Write([]byte("preprocessor:"))
	return nil
}

func errorPreProcessor(resp http.ResponseWriter, req *http.Request) error {
	resp.Write([]byte("error_preprocessor:"))
	return errors.New("error in preprocessor")
}

func postProcessor(resp http.ResponseWriter, req *http.Request, serviceResp interface{}, err error) {
	r := serviceResp.(*proto.SayHelloResponse)
	resp.Write([]byte("postprocessor:" + r.Message))
}

func thriftPostProcessor(resp http.ResponseWriter, req *http.Request, serviceResp interface{}, err error) {
	r := serviceResp.(*tgen.SayHelloResponse)
	resp.Write([]byte("postprocessor:" + r.Message))
}

func hijacker(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte("hijacker"))
}

func errorHandler(resp http.ResponseWriter, req *http.Request, err error) {
	resp.Write([]byte("from errorHandler:" + err.Error()))
}
