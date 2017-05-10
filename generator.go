package turbo

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

func CreateProject(pkgPath, serviceName, serverType string) {
	InitPkgPath(pkgPath)
	// TODO what if serviceName is start with a lower case character?
	// TODO panic if root folder already exists
	createRootFolder()
	createServiceYaml(serviceName)
	LoadServiceConfig()
	if serverType == "grpc" {
		CreateGrpcProject(serviceName)
	} else if serverType == "thrift" {
		CreateThriftProject(serviceName)
	}
}

func CreateGrpcProject(serviceName string) {
	createGrpcFolders()
	createProto(serviceName)
	GenerateGrpcSwitcher()
	GenerateProtobufStub()
	generateGrpcServiceMain()
	generateGrpcServiceImpl()
	generateGrpcHTTPMain()
}

func CreateThriftProject(serviceName string) {
	createThriftFolders()
	createThrift(serviceName)
	GenerateThriftSwitcher()
	GenerateThriftStub()
	generateThriftServiceMain()
	generateThriftServiceImpl()
	generateThriftHTTPMain()
}

func createRootFolder() {
	os.MkdirAll(serviceRootPath+"/gen", 0755)
}

func createGrpcFolders() {
	os.MkdirAll(serviceRootPath+"/grpcapi", 0755)
	os.MkdirAll(serviceRootPath+"/grpcservice/impl", 0755)
}

func createThriftFolders() {
	os.MkdirAll(serviceRootPath+"/thriftapi", 0755)
	os.MkdirAll(serviceRootPath+"/thriftservice/impl", 0755)
}

func createServiceYaml(serviceName string) {
	tmpl, err := template.New("yaml").Parse(serviceYaml)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create(serviceRootPath + "/service.yaml")
	err = tmpl.Execute(f, serviceYamlValues{ServiceName: serviceName})
	if err != nil {
		panic(err)
	}
}

type serviceYamlValues struct {
	ServiceName string
}

var serviceYaml string = `config:
  port: 8081
  grpc_service_name: {{.ServiceName}}
  grpc_service_address: 127.0.0.1:50051
  thrift_service_name: {{.ServiceName}}
  thrift_service_address: 127.0.0.1:50052

urlmapping:
  - GET /hello SayHello
`

func createProto(serviceName string) {
	nameLower := strings.ToLower(serviceName)
	tmpl, err := template.New("proto").Parse(proto)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create(serviceRootPath + "/" + nameLower + ".proto")
	err = tmpl.Execute(f, protoValues{ServiceName: serviceName})
	if err != nil {
		panic(err)
	}
}

type protoValues struct {
	ServiceName string
}

var proto string = `syntax = "proto3";
package gen;

message SayHelloRequest {
    string yourName = 1;
}

message SayHelloResponse {
    string message = 1;
}

service {{.ServiceName}} {
    rpc sayHello (SayHelloRequest) returns (SayHelloResponse) {}
}
`

func createThrift(serviceName string) {
	nameLower := strings.ToLower(serviceName)
	tmpl, err := template.New("thrift").Parse(thriftFile)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create(serviceRootPath + "/" + nameLower + ".thrift")
	err = tmpl.Execute(f, thriftValues{ServiceName: serviceName})
	if err != nil {
		panic(err)
	}
}

type thriftValues struct {
	ServiceName string
}

var thriftFile string = `namespace go gen

struct SayHelloResponse {
  1: string message,
}

service {{.ServiceName}} {
    SayHelloResponse sayHello (1:string yourName)
}
`

/*
generate grpcswitcher.go, [service_name].pb.go, grpcservice/[service_name].go, " +
		"grpcservice/impl/[service_name]impl.go
*/
func GenerateGrpcSwitcher() {
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range UrlServiceMap {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		tmpl, err := template.New("cases").Parse(grpcCases)
		if err != nil {
			panic(err)
		}
		var casesBuf bytes.Buffer
		err = tmpl.Execute(&casesBuf, method{configs[GRPC_SERVICE_NAME], v})
		casesStr = casesStr + casesBuf.String()
	}
	tmpl, err := template.New("switcher").Parse(grpcSwitcherFunc)
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(serviceRootPath + "/gen"); os.IsNotExist(err) {
		os.Mkdir(serviceRootPath+"/gen", 0755)
	}
	f, _ := os.Create(serviceRootPath + "/gen/grpcswitcher.go")
	err = tmpl.Execute(f, handlerContent{Cases: casesStr})
	if err != nil {
		panic(err)
	}
}

type method struct {
	ServiceName string
	MethodName  string
}

type handlerContent struct {
	Cases   string
	PkgPath string
}

var grpcSwitcherFunc string = `package gen

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
	switch methodName { {{.Cases}}
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}
`

var grpcCases string = `
	case "{{.MethodName}}":
		request := {{.MethodName}}Request{}
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
		result := reflect.ValueOf(turbo.GrpcService().({{.ServiceName}}Client)).MethodByName(methodName).Call(params)
		if result[1].Interface() == nil {
			return result[0].Interface(), nil
		} else {
			return nil, result[1].Interface().(error)
		}`

func GenerateProtobufStub() {
	nameLower := strings.ToLower(configs[GRPC_SERVICE_NAME])
	cmd := "protoc -I " + serviceRootPath + " " + serviceRootPath + "/" + nameLower + ".proto --go_out=plugins=grpc:" + serviceRootPath + "/gen"
	excuteCmd("bash", "-c", cmd)
}

func GenerateThriftSwitcher() {
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range UrlServiceMap {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		tmpl, err := template.New("cases").Parse(thriftCases)
		if err != nil {
			panic(err)
		}
		var casesBuf bytes.Buffer
		err = tmpl.Execute(&casesBuf, method{configs[THRIFT_SERVICE_NAME], v})
		casesStr = casesStr + casesBuf.String()
	}
	tmpl, err := template.New("switcher").Parse(thriftSwitcherFunc)
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(serviceRootPath + "/gen"); os.IsNotExist(err) {
		os.Mkdir(serviceRootPath+"/gen", 0755)
	}
	f, _ := os.Create(serviceRootPath + "/gen/thriftswitcher.go")
	err = tmpl.Execute(f, handlerContent{Cases: casesStr, PkgPath: servicePkgPath})
	if err != nil {
		panic(err)
	}
}

var thriftSwitcherFunc string = `package gen

import (
	"{{.PkgPath}}/gen/gen-go/gen"
	"reflect"
	"net/http"
	"turbo"
	"errors"
)

/*
this is a generated file, DO NOT EDIT!
 */
var ThriftSwitcher = func(methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName { {{.Cases}}
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}
`

var thriftCases string = `
	case "{{.MethodName}}":
		args := gen.{{.ServiceName}}{{.MethodName}}Args{}
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
		result := reflect.ValueOf(turbo.ThriftService().(*gen.{{.ServiceName}}Client)).MethodByName(methodName).Call(params)
		if result[1].Interface() == nil {
			return result[0].Interface(), nil
		} else {
			return nil, result[1].Interface().(error)
		}`

func GenerateThriftStub() {
	nameLower := strings.ToLower(configs[THRIFT_SERVICE_NAME])
	cmd := "thrift -r --gen go -o" + " " + serviceRootPath + "/" + "gen " + serviceRootPath + "/" + nameLower + ".thrift"
	excuteCmd("bash", "-c", cmd)
}

func excuteCmd(cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	if err := c.Run(); err != nil {
		panic(err)
	}
}

func generateGrpcServiceMain() {
	nameLower := strings.ToLower(configs[GRPC_SERVICE_NAME])
	tmpl, err := template.New("main").Parse(serviceMain)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create(serviceRootPath + "/grpcservice/" + nameLower + ".go")
	err = tmpl.Execute(f, serviceMainValues{PkgPath: servicePkgPath, Port: "50051", ServiceName: configs[GRPC_SERVICE_NAME]})
	if err != nil {
		panic(err)
	}
}

type serviceMainValues struct {
	PkgPath     string
	Port        string
	ServiceName string
}

var serviceMain string = `package main

import (
	"net"
	"log"
	"google.golang.org/grpc"
	"{{.PkgPath}}/grpcservice/impl"
	"{{.PkgPath}}/gen"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":{{.Port}}")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	gen.Register{{.ServiceName}}Server(grpcServer, &impl.{{.ServiceName}}{})

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
`

func generateThriftServiceMain() {
	nameLower := strings.ToLower(configs[THRIFT_SERVICE_NAME])
	tmpl, err := template.New("main").Parse(thriftServiceMain)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create(serviceRootPath + "/thriftservice/" + nameLower + ".go")
	err = tmpl.Execute(f, thriftServiceMainValues{PkgPath: servicePkgPath, Port: "50052", ServiceName: configs[THRIFT_SERVICE_NAME]})
	if err != nil {
		panic(err)
	}
}

type thriftServiceMainValues struct {
	PkgPath     string
	Port        string
	ServiceName string
}

var thriftServiceMain string = `package main

import (
	"{{.PkgPath}}/thriftservice/impl"
	"{{.PkgPath}}/gen/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
	"log"
	"os"
)

func main() {
	transport, err := thrift.NewTServerSocket(":{{.Port}}")
	if err != nil {
		log.Println("socket error")
		os.Exit(1)
	}

	server := thrift.NewTSimpleServer4(gen.New{{.ServiceName}}Processor(impl.{{.ServiceName}}{}), transport,
		thrift.NewTTransportFactory(),thrift.NewTBinaryProtocolFactoryDefault())
	server.Serve()
}
`

func generateGrpcServiceImpl() {
	nameLower := strings.ToLower(configs[GRPC_SERVICE_NAME])
	tmpl, err := template.New("impl").Parse(serviceImpl)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create(serviceRootPath + "/grpcservice/impl/" + nameLower + "impl.go")
	err = tmpl.Execute(f, serviceImplValues{PkgPath: servicePkgPath, ServiceName: configs[GRPC_SERVICE_NAME]})
	if err != nil {
		panic(err)
	}
}

type serviceImplValues struct {
	PkgPath     string
	ServiceName string
}

var serviceImpl string = `package impl

import (
	"golang.org/x/net/context"
	"{{.PkgPath}}/gen"
)

type {{.ServiceName}} struct {
}

func (s *{{.ServiceName}}) SayHello(ctx context.Context, req *gen.SayHelloRequest) (*gen.SayHelloResponse, error) {
	return &gen.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
}
`

func generateThriftServiceImpl() {
	nameLower := strings.ToLower(configs[THRIFT_SERVICE_NAME])
	tmpl, err := template.New("impl").Parse(thriftServiceImpl)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create(serviceRootPath + "/thriftservice/impl/" + nameLower + "impl.go")
	err = tmpl.Execute(f, thriftServiceImplValues{PkgPath: servicePkgPath, ServiceName: configs[THRIFT_SERVICE_NAME]})
	if err != nil {
		panic(err)
	}
}

type thriftServiceImplValues struct {
	PkgPath     string
	ServiceName string
}

var thriftServiceImpl string = `package impl

import (
	"{{.PkgPath}}/gen/gen-go/gen"
)

type {{.ServiceName}} struct {
}

func (s {{.ServiceName}}) SayHello(yourName string) (r *gen.SayHelloResponse, err error) {
	return &gen.SayHelloResponse{Message: "[thrift server]Hello, " + yourName}, nil
}
`

func generateGrpcHTTPMain() {
	nameLower := strings.ToLower(configs[GRPC_SERVICE_NAME])
	tmpl, err := template.New("httpmain").Parse(_HTTPMain)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create(serviceRootPath + "/grpcapi/" + nameLower + "api.go")
	err = tmpl.Execute(f, _HTTPMainValues{ServiceName: configs[GRPC_SERVICE_NAME], PkgPath: servicePkgPath})
	if err != nil {
		panic(err)
	}
}

type _HTTPMainValues struct {
	ServiceName string
	PkgPath     string
}

var _HTTPMain string = `package main

import (
	"turbo"
	"google.golang.org/grpc"
	"{{.PkgPath}}/gen"
)

func main() {
	turbo.StartGrpcHTTPServer("{{.PkgPath}}", grpcClient, gen.GrpcSwitcher)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return gen.New{{.ServiceName}}Client(conn)
}
`

func generateThriftHTTPMain() {
	nameLower := strings.ToLower(configs[THRIFT_SERVICE_NAME])
	tmpl, err := template.New("httpmain").Parse(thriftHTTPMain)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create(serviceRootPath + "/thriftapi/" + nameLower + "api.go")
	err = tmpl.Execute(f, _HTTPMainValues{ServiceName: configs[THRIFT_SERVICE_NAME], PkgPath: servicePkgPath})
	if err != nil {
		panic(err)
	}
}

var thriftHTTPMain string = `package main

import (
	"turbo"
	"{{.PkgPath}}/gen"
	t "{{.PkgPath}}/gen/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func main() {
	turbo.StartThriftHTTPServer("{{.PkgPath}}", thriftClient, gen.ThriftSwitcher)
}

func thriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.New{{.ServiceName}}ClientFactory(trans, f)
}
`
