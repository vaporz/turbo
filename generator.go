package turbo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

// CreateProject creates a whole new project!
func CreateProject(pkgPath, serviceName, serverType string) {
	initPkgPath(pkgPath)
	// TODO pop alert, force user to confirm if root folder already exists
	createRootFolder()
	createServiceYaml(serviceName)
	InitRpcType(serverType)
	loadServiceConfig()
	if serverType == "grpc" {
		createGrpcProject(serviceName)
	} else if serverType == "thrift" {
		createThriftProject(serviceName)
	}
}

func createGrpcProject(serviceName string) {
	createGrpcFolders()
	createProto(serviceName)
	GenerateProtobufStub(" -I " + serviceRootPath + " " + serviceRootPath + "/" + strings.ToLower(serviceName) + ".proto ")
	GenerateGrpcSwitcher()
	generateGrpcServiceMain()
	generateGrpcServiceImpl()
	generateGrpcHTTPMain()
}

func createThriftProject(serviceName string) {
	createThriftFolders()
	createThrift(serviceName)
	GenerateThriftStub(" -I " + serviceRootPath + " ")
	GenerateThriftSwitcher()
	generateThriftServiceMain()
	generateThriftServiceImpl()
	generateThriftHTTPMain()
}

func createRootFolder() {
	os.MkdirAll(serviceRootPath+"/gen", 0755)
}

func createGrpcFolders() {
	os.MkdirAll(serviceRootPath+"/gen/proto", 0755)
	os.MkdirAll(serviceRootPath+"/grpcapi", 0755)
	os.MkdirAll(serviceRootPath+"/grpcservice/impl", 0755)
}

func createThriftFolders() {
	os.MkdirAll(serviceRootPath+"/gen/thrift", 0755)
	os.MkdirAll(serviceRootPath+"/thriftapi", 0755)
	os.MkdirAll(serviceRootPath+"/thriftservice/impl", 0755)
}

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

func createServiceYaml(serviceName string) {
	if _, err := os.Stat(serviceRootPath + "/service.yaml"); os.IsExist(err) {
		return
	}
	writeFileWithTemplate(
		serviceRootPath+"/service.yaml",
		serviceYaml,
		serviceYamlValues{ServiceName: serviceName},
	)
}

type serviceYamlValues struct {
	ServiceName string
}

var serviceYaml string = `config:
  httpPort: 8081
  grpc_service_name: {{.ServiceName}}
  grpc_service_address: 127.0.0.1:50051
  thrift_service_name: {{.ServiceName}}
  thrift_service_address: 127.0.0.1:50052

urlmapping:
  - GET /hello SayHello
`

func createProto(serviceName string) {
	nameLower := strings.ToLower(serviceName)
	writeFileWithTemplate(
		serviceRootPath+"/"+nameLower+".proto",
		protoFile,
		protoValues{ServiceName: serviceName},
	)
}

type protoValues struct {
	ServiceName string
}

var protoFile string = `syntax = "proto3";
package proto;

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
	writeFileWithTemplate(
		serviceRootPath+"/"+nameLower+".thrift",
		thriftFile,
		thriftValues{ServiceName: serviceName},
	)
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

// GenerateGrpcSwitcher generates "grpcswither.go"
func GenerateGrpcSwitcher() {
	if _, err := os.Stat(serviceRootPath + "/gen"); os.IsNotExist(err) {
		os.Mkdir(serviceRootPath+"/gen", 0755)
	}
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range urlServiceMaps {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		var casesBuf bytes.Buffer
		writeWithTemplate(
			&casesBuf,
			grpcCases,
			method{v, structFields(v)},
		)
		casesStr = casesStr + casesBuf.String()
	}
	writeFileWithTemplate(
		serviceRootPath+"/gen/grpcswitcher.go",
		grpcSwitcherFunc,
		handlerContent{
			ServiceName: configs[grpcServiceName],
			Cases:       casesStr,
			PkgPath:     servicePkgPath},
	)
}

func structFields(methodName string) string {
	requestName := methodName + "Request"
	fields := fieldMappings[requestName]
	var structFields string
	for _, field := range fields {
		pair := strings.Split(field, " ")
		nameSlice := []rune(pair[1])
		name := strings.ToUpper(string(nameSlice[0])) + string(nameSlice[1:])
		typeName := pair[0]
		structFields = structFields + name + ": &proto." + typeName + "{},"
	}
	return structFields
}

type method struct {
	MethodName   string
	StructFields string
}

type handlerContent struct {
	ServiceName string
	Cases       string
	PkgPath     string
}

var grpcSwitcherFunc string = `package gen

import (
	"reflect"
	"net/http"
	"github.com/vaporz/turbo"
	"errors"
	"{{.PkgPath}}/gen/proto"
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

func callGrpcMethod(methodName string, params []reflect.Value) []reflect.Value {
	return reflect.ValueOf(turbo.GrpcService().(proto.{{.ServiceName}}Client)).MethodByName(methodName).Call(params)
}
`

var grpcCases string = `
	case "{{.MethodName}}":
		request := &proto.{{.MethodName}}Request{ {{.StructFields}} }
		err = turbo.BuildStruct(reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return nil, err
		}
		params := turbo.MakeParams(req, reflect.ValueOf(request))
		return turbo.ParseResult(callGrpcMethod(methodName, params))`

// GenerateProtobufStub generates protobuf stub codes
func GenerateProtobufStub(options string) {
	if _, err := os.Stat(serviceRootPath + "/gen/proto"); os.IsNotExist(err) {
		fmt.Println("proto folder missing, create one")
		os.MkdirAll(serviceRootPath+"/gen/proto", 0755)
	}
	cmd := "protoc " + options + " --go_out=plugins=grpc:" + serviceRootPath + "/gen/proto"
	executeCmd("bash", "-c", cmd)
}

// GenerateThriftSwitcher generates "thriftswitcher.go"
func GenerateThriftSwitcher() {
	if _, err := os.Stat(serviceRootPath + "/gen"); os.IsNotExist(err) {
		os.Mkdir(serviceRootPath+"/gen", 0755)
	}
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range urlServiceMaps {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		var casesBuf bytes.Buffer
		writeWithTemplate(
			&casesBuf,
			thriftCases,
			thriftMethod{configs[thriftServiceName], v},
		)
		casesStr = casesStr + casesBuf.String()
	}

	var argCasesStr string
	for k := range fieldMappings {
		var argCasesBuf bytes.Buffer
		writeWithTemplate(
			&argCasesBuf,
			buildArgCases,
			buildArgParams{k},
		)
		argCasesStr = argCasesStr + argCasesBuf.String()
	}
	writeFileWithTemplate(
		serviceRootPath+"/gen/thriftswitcher.go",
		thriftSwitcherFunc,
		thriftHandlerContent{
			ServiceName:    configs[thriftServiceName],
			Cases:          casesStr,
			PkgPath:        servicePkgPath,
			BuildArgsCases: argCasesStr},
	)
}

type thriftMethod struct {
	ServiceName string
	MethodName  string
}

type thriftHandlerContent struct {
	ServiceName    string
	Cases          string
	PkgPath        string
	BuildArgsCases string
}

type buildArgParams struct {
	StructName string
}

var thriftSwitcherFunc string = `package gen

import (
	"{{.PkgPath}}/gen/thrift/gen-go/gen"
	"reflect"
	"net/http"
	"github.com/vaporz/turbo"
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

func callThriftMethod(methodName string, params []reflect.Value) []reflect.Value {
	return reflect.ValueOf(turbo.ThriftService().(*gen.{{.ServiceName}}Client)).MethodByName(methodName).Call(params)
}

func buildStructArg(typeName string, req *http.Request) (v reflect.Value, err error) {
	switch typeName { {{.BuildArgsCases}}
	default:
		return v, errors.New("unknown typeName[" + typeName + "]")
	}
}
`

var thriftCases string = `
	case "{{.MethodName}}":
		args := gen.{{.ServiceName}}{{.MethodName}}Args{}
		params, err := turbo.BuildArgs(reflect.TypeOf(args), reflect.ValueOf(args), req, buildStructArg)
		if err != nil {
			return nil, err
		}
		return turbo.ParseResult(callThriftMethod(methodName, params))`

var buildArgCases string = `
	case "{{.StructName}}":
		request := &gen.{{.StructName}}{}
		err = turbo.BuildStruct(reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return v, err
		}
		return reflect.ValueOf(request), nil`

// GenerateThriftStub generates Thrift stub codes
func GenerateThriftStub(options string) {
	if _, err := os.Stat(serviceRootPath + "/gen/thrift"); os.IsNotExist(err) {
		fmt.Println("thrift folder missing, create one")
		os.MkdirAll(serviceRootPath+"/gen/thrift", 0755)
	}
	nameLower := strings.ToLower(configs[thriftServiceName])
	cmd := "thrift " + options + " -r --gen go:package_prefix=" + servicePkgPath + "/gen/thrift/gen-go/ -o" +
		" " + serviceRootPath + "/" + "gen/thrift " + serviceRootPath + "/" + nameLower + ".thrift"
	executeCmd("bash", "-c", cmd)
}

func executeCmd(cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	if err := c.Run(); err != nil {
		panic(err)
	}
}

func generateGrpcServiceMain() {
	nameLower := strings.ToLower(configs[grpcServiceName])
	writeFileWithTemplate(
		serviceRootPath+"/grpcservice/"+nameLower+".go",
		serviceMain,
		serviceMainValues{PkgPath: servicePkgPath, Port: "50051", ServiceName: configs[grpcServiceName]},
	)
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
	"{{.PkgPath}}/gen/proto"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":{{.Port}}")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	proto.Register{{.ServiceName}}Server(grpcServer, &impl.{{.ServiceName}}{})

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
`

func generateThriftServiceMain() {
	nameLower := strings.ToLower(configs[thriftServiceName])
	writeFileWithTemplate(
		serviceRootPath+"/thriftservice/"+nameLower+".go",
		thriftServiceMain,
		thriftServiceMainValues{PkgPath: servicePkgPath, Port: "50052", ServiceName: configs[thriftServiceName]},
	)
}

type thriftServiceMainValues struct {
	PkgPath     string
	Port        string
	ServiceName string
}

var thriftServiceMain string = `package main

import (
	"{{.PkgPath}}/thriftservice/impl"
	"{{.PkgPath}}/gen/thrift/gen-go/gen"
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
	nameLower := strings.ToLower(configs[grpcServiceName])
	writeFileWithTemplate(
		serviceRootPath+"/grpcservice/impl/"+nameLower+"impl.go",
		serviceImpl,
		serviceImplValues{PkgPath: servicePkgPath, ServiceName: configs[grpcServiceName]},
	)
}

type serviceImplValues struct {
	PkgPath     string
	ServiceName string
}

var serviceImpl string = `package impl

import (
	"golang.org/x/net/context"
	"{{.PkgPath}}/gen/proto"
)

type {{.ServiceName}} struct {
}

func (s *{{.ServiceName}}) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
}
`

func generateThriftServiceImpl() {
	nameLower := strings.ToLower(configs[thriftServiceName])
	writeFileWithTemplate(
		serviceRootPath+"/thriftservice/impl/"+nameLower+"impl.go",
		thriftServiceImpl,
		thriftServiceImplValues{PkgPath: servicePkgPath, ServiceName: configs[thriftServiceName]},
	)
}

type thriftServiceImplValues struct {
	PkgPath     string
	ServiceName string
}

var thriftServiceImpl string = `package impl

import (
	"{{.PkgPath}}/gen/thrift/gen-go/gen"
)

type {{.ServiceName}} struct {
}

func (s {{.ServiceName}}) SayHello(yourName string) (r *gen.SayHelloResponse, err error) {
	return &gen.SayHelloResponse{Message: "[thrift server]Hello, " + yourName}, nil
}
`

func generateGrpcHTTPMain() {
	nameLower := strings.ToLower(configs[grpcServiceName])
	writeFileWithTemplate(
		serviceRootPath+"/grpcapi/"+nameLower+"api.go",
		_HTTPMain,
		_HTTPMainValues{ServiceName: configs[grpcServiceName], PkgPath: servicePkgPath},
	)
}

type _HTTPMainValues struct {
	ServiceName string
	PkgPath     string
}

var _HTTPMain string = `package main

import (
	"github.com/vaporz/turbo"
	"google.golang.org/grpc"
	"{{.PkgPath}}/gen"
	"{{.PkgPath}}/gen/proto"
)

func main() {
	turbo.StartGrpcHTTPServer("{{.PkgPath}}", grpcClient, gen.GrpcSwitcher)
}

func grpcClient(conn *grpc.ClientConn) interface{} {
	return proto.New{{.ServiceName}}Client(conn)
}
`

func generateThriftHTTPMain() {
	nameLower := strings.ToLower(configs[thriftServiceName])
	writeFileWithTemplate(
		serviceRootPath+"/thriftapi/"+nameLower+"api.go",
		thriftHTTPMain,
		_HTTPMainValues{ServiceName: configs[thriftServiceName], PkgPath: servicePkgPath},
	)
}

var thriftHTTPMain string = `package main

import (
	"github.com/vaporz/turbo"
	"{{.PkgPath}}/gen"
	t "{{.PkgPath}}/gen/thrift/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func main() {
	turbo.StartThriftHTTPServer("{{.PkgPath}}", thriftClient, gen.ThriftSwitcher)
}

func thriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.New{{.ServiceName}}ClientFactory(trans, f)
}
`
