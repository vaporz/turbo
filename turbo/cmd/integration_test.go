package cmd

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/vaporz/turbo"
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
	"bytes"
	"fmt"
	"io"
	"text/template"
	"errors"
)

func TestMain(m *testing.M) {
	//os.RemoveAll(turbo.TurboRootPath + "/test/testservice/service.yaml")
	//os.RemoveAll(turbo.TurboRootPath + "/test/testservice")
	//time.Sleep(time.Second*1)
	turbo.InitGOPATH()
	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	create(t, "grpc")
	rpcType = ""
	filePaths = []string{}

	os.RemoveAll(turbo.TurboRootPath + "/test/testservice/service.yaml")
	create(t, "thrift")
	rpcType = ""
	filePaths = []string{}
}

func TestGenerate(t *testing.T) {
	generate(t, "grpc")
	rpcType = ""
	filePaths = []string{}

	generate(t, "thrift")
	rpcType = ""
	filePaths = []string{}
}

func create(t *testing.T, rpcType string) {
	RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testservice"})
	err := Execute()
	assert.Equal(t, "invalid args", err.Error())

	RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testservice", "test_service"})
	err = Execute()
	assert.Contains(t, err.Error(), "not a CamelCase string")

	RootCmd.SetArgs([]string{"create", "github.com/vaporz/turbo/test/testservice", "TestService", "-r", rpcType, "-f", "true"})
	err = Execute()
	assert.Nil(t, err)
}

func generate(t *testing.T, rpcType string) {
	RootCmd.SetArgs([]string{"generate"})
	err := Execute()
	assert.Equal(t, "Usage: generate [package_path]", err.Error())

	RootCmd.SetArgs([]string{"generate", "github.com/vaporz/turbo/test/testservice"})
	err = Execute()
	assert.Equal(t, "missing rpctype (-r)", err.Error())

	RootCmd.SetArgs([]string{"generate", "github.com/vaporz/turbo/test/testservice", "-r", "unknown"})
	err = Execute()
	assert.Equal(t, "invalid rpctype", err.Error())

	if rpcType == "grpc" {
		RootCmd.SetArgs([]string{"generate", "github.com/vaporz/turbo/test/testservice", "-r", rpcType})
		err = Execute()
		assert.Equal(t, "missing .proto file path (-I)", err.Error())
	}

	RootCmd.SetArgs([]string{"generate", "github.com/vaporz/turbo/test/testservice", "-r", rpcType,
							 "-I", turbo.TurboRootPath + "/test/testservice"})
	err = Execute()
	assert.Nil(t, err)
}

func TestGrpcService(t *testing.T) {
	httpPort := "8081"
	overrideServiceYaml("8081", "50051", "development")
	turbo.ResetChans()
	go turbo.StartGRPC("github.com/vaporz/turbo/test/testservice", "service",
		50051, gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)
	time.Sleep(time.Second * 1)

	runTests(t, httpPort, "grpc")
	turbo.Stop()
}

func resetComponents() {
	turbo.ResetConvertor()
	turbo.ResetHijacker()
	turbo.ResetInterceptor()
	turbo.ResetPostprocessor()
	turbo.ResetPreprocessor()
	turbo.ResetErrorHandler()
}

func TestThriftService(t *testing.T) {
	httpPort := "8083"
	overrideServiceYaml(httpPort, "50053", "production")
	turbo.ResetChans()
	go turbo.StartTHRIFT("github.com/vaporz/turbo/test/testservice", "service",
		50053, tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
	time.Sleep(time.Second * 2)
	turbo.SetOutput(os.Stdout)

	runTests(t, httpPort, "thrift")
	turbo.Stop()
}

func runTests(t *testing.T, httpPort, rpcType string) {
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest", "{\"message\":\"["+rpcType+" server]Hello, testtest\"}")
	testPost(t, "http://localhost:"+httpPort+"/hello?your_name=testtest", "404 page not found\n")

	turbo.SetCommonInterceptor(Test1Interceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"test1_intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	// TODO test errorHandler
	turbo.WithErrorHandler(errorHandler)

	resetComponents()
	turbo.Intercept([]string{"GET"}, "/", TestInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest&yourName=testname",
		"intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	resetComponents()
	turbo.Intercept([]string{"GET"}, "/", TestInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	resetComponents()
	turbo.Intercept([]string{"GET"}, "/hello", BeforeErrorInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"interceptor_error:")

	resetComponents()
	turbo.Intercept([]string{"GET"}, "/hello", turbo.BaseInterceptor{}, BeforeErrorInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"interceptor_error:")

	resetComponents()
	turbo.Intercept([]string{"GET"}, "/hello", TestInterceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	resetComponents()
	turbo.Intercept([]string{"GET"}, "/hello", TestInterceptor{}, Test1Interceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"intercepted:test1_intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	turbo.Intercept([]string{"GET"}, "/hello", TestInterceptor{}, AfterErrorInterceptor{}, Test1Interceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"intercepted:test1_intercepted:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	resetComponents()
	turbo.Intercept([]string{"GET"}, "/hello", TestInterceptor{}, BeforeErrorInterceptor{}, Test1Interceptor{})
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"intercepted:interceptor_error:")

	resetComponents()
	turbo.Intercept([]string{"GET"}, "/hello", TestInterceptor{})
	turbo.SetPreprocessor("/hello", errorPreProcessor)
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"intercepted:error_preprocessor:")

	resetComponents()
	turbo.Intercept([]string{"GET"}, "/hello", TestInterceptor{})
	turbo.SetPreprocessor("/hello", preProcessor)
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"intercepted:preprocessor:{\"message\":\"["+rpcType+" server]Hello, testtest\"}")

	if rpcType == "thrift" {
		turbo.SetPostprocessor("/hello", thriftPostProcessor)
	} else {
		turbo.SetPostprocessor("/hello", postProcessor)
	}
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"intercepted:preprocessor:postprocessor:["+rpcType+" server]Hello, testtest")

	turbo.SetHijacker("/hello", hijacker)
	testGet(t, "http://localhost:"+httpPort+"/hello?your_name=testtest",
		"intercepted:hijacker")
	resetComponents()
}

func testGet(t *testing.T, url, expected string) {
	resp, err := http.Get(url)
	defer resp.Body.Close()
	assert.Nil(t, err)
	assert.Equal(t, expected, readResp(resp))
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

func overrideServiceYaml(httpPort, servicePort, env string) {
	writeFileWithTemplate(
		turbo.GOPATH+"/src/github.com/vaporz/turbo/test/testservice/service.yaml",
		serviceYamlFile,
		serviceYamlValues{
			HttpPort:    httpPort,
			ServiceName: "TestService",
			ServicePort: servicePort,
			Env:         env,
		},
	)
}

type serviceYamlValues struct {
	HttpPort    string
	ServiceName string
	ServicePort string
	Env         string
}

var serviceYamlFile string = `config:
  http_port: {{.HttpPort}}
  environment: {{.Env}}
  turbo_log_path: log
  grpc_service_name: {{.ServiceName}}
  grpc_service_address: 127.0.0.1:{{.ServicePort}}
  thrift_service_name: {{.ServiceName}}
  thrift_service_address: 127.0.0.1:{{.ServicePort}}

urlmapping:
  - GET /hello SayHello
`

func writeWithTemplate(wr io.Writer, text string, data interface{}) {
	tmpl, err := template.New("").Parse(text)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(wr, data)
	if err != nil {
		panic(err)
	}
}

func writeFileWithTemplate(filePath, text string, data interface{}) {
	f, err := os.Create(filePath)
	if err != nil {
		panic("fail to create file:" + filePath)
	}
	writeWithTemplate(f, text, data)
}
