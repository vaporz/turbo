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
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	initPkgPath(pkgPath)
	initConfigFileName("service")
	validateServiceRootPath()
	createRootFolder()
	createServiceYaml(serviceName)
	initRpcType(serverType)
	loadServiceConfig()
	if serverType == "grpc" {
		createGrpcProject(serviceName)
	} else if serverType == "thrift" {
		createThriftProject(serviceName)
	}
}

func validateServiceRootPath() {
	_, err := os.Stat(Config.ServiceRootPath())
	if os.IsNotExist(err) {
		return
	}
	fmt.Print("Path '" + Config.ServiceRootPath() + " already exist!\n" +
		"Do you want to remove this directory before creating a new project? (type 'y' to remove):'")
	var input string
	fmt.Scanln(&input)
	if input != "y" {
		return
	}
	fmt.Print("All files in that directory will be lost, are you sure? (type 'y' to continue):'")
	fmt.Scanln(&input)
	if input != "y" {
		panic("aborted")
	}
	os.RemoveAll(Config.ServiceRootPath())
}

func createGrpcProject(serviceName string) {
	createGrpcFolders()
	createProto(serviceName)
	GenerateProtobufStub(" -I " + Config.ServiceRootPath() + " " + Config.ServiceRootPath() + "/" + strings.ToLower(serviceName) + ".proto ")
	GenerateGrpcSwitcher()
	generateGrpcServiceMain()
	generateGrpcServiceImpl()
	generateGrpcHTTPMain()
	generateGrpcHTTPComponent()
	generateServiceMain("grpc")
}

func createThriftProject(serviceName string) {
	createThriftFolders()
	createThrift(serviceName)
	GenerateThriftStub(" -I " + Config.ServiceRootPath() + " ")
	GenerateBuildThriftParameters()
	GenerateThriftSwitcher()
	generateThriftServiceMain()
	generateThriftServiceImpl()
	generateThriftHTTPMain()
	generateThriftHTTPComponent()
	generateServiceMain("thrift")
}

func createRootFolder() {
	os.MkdirAll(Config.ServiceRootPath()+"/gen", 0755)
}

func createGrpcFolders() {
	os.MkdirAll(Config.ServiceRootPath()+"/gen/proto", 0755)
	os.MkdirAll(Config.ServiceRootPath()+"/grpcapi/component", 0755)
	os.MkdirAll(Config.ServiceRootPath()+"/grpcservice/impl", 0755)
}

func createThriftFolders() {
	os.MkdirAll(Config.ServiceRootPath()+"/gen/thrift", 0755)
	os.MkdirAll(Config.ServiceRootPath()+"/thriftapi/component", 0755)
	os.MkdirAll(Config.ServiceRootPath()+"/thriftservice/impl", 0755)
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
	if _, err := os.Stat(Config.ServiceRootPath() + "/" + Config.ConfigFileName() + ".yaml"); err == nil {
		return
	}
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/"+Config.ConfigFileName()+".yaml",
		serviceYamlFile,
		serviceYamlValues{ServiceName: serviceName},
	)
}

type serviceYamlValues struct {
	ServiceName string
}

var serviceYamlFile string = `config:
  http_port: 8081
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
		Config.ServiceRootPath()+"/"+nameLower+".proto",
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
		Config.ServiceRootPath()+"/"+nameLower+".thrift",
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
	if _, err := os.Stat(Config.ServiceRootPath() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(Config.ServiceRootPath()+"/gen", 0755)
	}
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range Config.urlServiceMaps {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		var casesBuf bytes.Buffer
		writeWithTemplate(
			&casesBuf,
			grpcCases,
			method{
				ServiceName:  Config.GrpcServiceName(),
				MethodName:   v,
				StructFields: structFields(v + "Request")},
		)
		casesStr = casesStr + casesBuf.String()
	}
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/gen/grpcswitcher.go",
		grpcSwitcherFunc,
		handlerContent{
			Cases:   casesStr,
			PkgPath: Config.ServicePkgPath()},
	)
}

func structFields(structName string) string {
	fields, ok := Config.fieldMappings[structName]
	if !ok {
		return ""
	}
	var fieldStr string
	for _, field := range fields {
		pair := strings.Split(field, " ")
		nameSlice := []rune(pair[1])
		name := strings.ToUpper(string(nameSlice[0])) + string(nameSlice[1:])
		typeName := pair[0]
		fieldStr = fieldStr + name + ": &g." + typeName + "{" + structFields(typeName) + "},"
	}
	return fieldStr
}

type method struct {
	ServiceName  string
	MethodName   string
	StructFields string
}

type handlerContent struct {
	Cases   string
	PkgPath string
}

var grpcSwitcherFunc string = `package gen

import (
	g "{{.PkgPath}}/gen/proto"
	"github.com/vaporz/turbo"
	"reflect"
	"net/http"
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
		request := &g.{{.MethodName}}Request{ {{.StructFields}} }
		err = turbo.BuildStruct(reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return nil, err
		}
		return turbo.GrpcService().(g.{{.ServiceName}}Client).{{.MethodName}}(req.Context(), request)`

// GenerateProtobufStub generates protobuf stub codes
func GenerateProtobufStub(options string) {
	if _, err := os.Stat(Config.ServiceRootPath() + "/gen/proto"); os.IsNotExist(err) {
		fmt.Println("proto folder missing, create one")
		os.MkdirAll(Config.ServiceRootPath()+"/gen/proto", 0755)
	}
	cmd := "protoc " + options + " --go_out=plugins=grpc:" + Config.ServiceRootPath() + "/gen/proto"
	executeCmd("bash", "-c", cmd)
}

func GenerateBuildThriftParameters() {
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range Config.urlServiceMaps {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		var casesBuf bytes.Buffer
		writeWithTemplate(
			&casesBuf,
			thriftParameterCases,
			thriftParameterCasesValues{
				ServiceName: Config.ThriftServiceName(),
				MethodName:  v},
		)
		casesStr = casesStr + casesBuf.String()
	}

	writeFileWithTemplate(
		Config.ServiceRootPath()+"/gen/thrift/buildparameter.go",
		buildThriftParameters,
		buildThriftPamametersValues{
			PkgPath:     Config.ServicePkgPath(),
			ServiceName: Config.GrpcServiceName(),
			Cases:       casesStr},
	)
}

type thriftParameterCasesValues struct {
	MethodName  string
	ServiceName string
}

var thriftParameterCases string = `
	case "{{.MethodName}}":
		var result string
		args := g.{{.ServiceName}}{{.MethodName}}Args{}
		at := reflect.TypeOf(args)
		num := at.NumField()
		for i := 0; i < num; i++ {
			result += fmt.Sprintf(
			"\n\t\t\tparams[%d].Interface().(%s),",
			i, at.Field(i).Type.String())
		}
		return result`

type buildThriftPamametersValues struct {
	PkgPath     string
	ServiceName string
	Cases       string
}

var buildThriftParameters string = `package main

import (
	g "{{.PkgPath}}/gen/thrift/gen-go/gen"
	"reflect"
	"fmt"
	"flag"
)

var methodName = flag.String("n", "", "")

func main() {
	flag.Parse()
	str := buildParameterStr(*methodName)
	fmt.Print(str)
}

func buildParameterStr(methodName string) string {
	switch methodName { {{.Cases}}
	default:
		return "error"
	}
}
`

// GenerateThriftSwitcher generates "thriftswitcher.go"
func GenerateThriftSwitcher() {
	if _, err := os.Stat(Config.ServiceRootPath() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(Config.ServiceRootPath()+"/gen", 0755)
	}
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range Config.urlServiceMaps {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		parametersStr := thriftParameters(v)
		var casesBuf bytes.Buffer
		writeWithTemplate(
			&casesBuf,
			thriftCases,
			thriftMethod{
				ServiceName:        Config.ThriftServiceName(),
				MethodName:         v,
				Parameters:         parametersStr,
				NotEmptyParameters: len(strings.TrimSpace(parametersStr)) > 0},
		)
		casesStr = casesStr + casesBuf.String()
	}

	var argCasesStr string
	for k := range Config.fieldMappings {
		var argCasesBuf bytes.Buffer
		writeWithTemplate(
			&argCasesBuf,
			buildArgCases,
			buildArgParams{k, structFields(k)},
		)
		argCasesStr = argCasesStr + argCasesBuf.String()
	}
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/gen/thriftswitcher.go",
		thriftSwitcherFunc,
		thriftHandlerContent{
			Cases:          casesStr,
			PkgPath:        Config.ServicePkgPath(),
			BuildArgsCases: argCasesStr},
	)
}

func thriftParameters(methodName string) string {
	cmd := "go run " + Config.ServiceRootPath() + "/gen/thrift/buildparameter.go -n " + methodName
	buf := &bytes.Buffer{}
	c := exec.Command("bash", "-c", cmd)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = buf
	if err := c.Run(); err != nil {
		panic(err)
	}
	return buf.String() + " "
}

type thriftMethod struct {
	ServiceName        string
	MethodName         string
	Parameters         string
	NotEmptyParameters bool
}

type thriftHandlerContent struct {
	Cases          string
	PkgPath        string
	BuildArgsCases string
}

type buildArgParams struct {
	StructName   string
	StructFields string
}

var thriftSwitcherFunc string = `package gen

import (
	"{{.PkgPath}}/gen/thrift/gen-go/gen"
	"github.com/vaporz/turbo"
	"reflect"
	"net/http"
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

func buildStructArg(typeName string, req *http.Request) (v reflect.Value, err error) {
	switch typeName { {{.BuildArgsCases}}
	default:
		return v, errors.New("unknown typeName[" + typeName + "]")
	}
}
`

var thriftCases string = `
	case "{{.MethodName}}":{{if .NotEmptyParameters }}
		args := gen.{{.ServiceName}}{{.MethodName}}Args{}
		params, err := turbo.BuildArgs(reflect.TypeOf(args), reflect.ValueOf(args), req, buildStructArg)
		if err != nil {
			return nil, err
		}{{end}}
		return turbo.ThriftService().(*gen.{{.ServiceName}}Client).{{.MethodName}}({{.Parameters}})`

var buildArgCases string = `
	case "{{.StructName}}":
		request := &gen.{{.StructName}}{ {{.StructFields}} }
		err = turbo.BuildStruct(reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return v, err
		}
		return reflect.ValueOf(request), nil`

// GenerateThriftStub generates Thrift stub codes
func GenerateThriftStub(options string) {
	if _, err := os.Stat(Config.ServiceRootPath() + "/gen/thrift"); os.IsNotExist(err) {
		fmt.Println("thrift folder missing, create one")
		os.MkdirAll(Config.ServiceRootPath()+"/gen/thrift", 0755)
	}
	nameLower := strings.ToLower(Config.ThriftServiceName())
	cmd := "thrift " + options + " -r --gen go:package_prefix=" + Config.ServicePkgPath() + "/gen/thrift/gen-go/ -o" +
		" " + Config.ServiceRootPath() + "/" + "gen/thrift " + Config.ServiceRootPath() + "/" + nameLower + ".thrift"
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
	nameLower := strings.ToLower(Config.GrpcServiceName())
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/grpcservice/"+nameLower+".go",
		serviceMain,
		serviceMainValues{PkgPath: Config.ServicePkgPath(), Port: "50051", ServiceName: Config.GrpcServiceName()},
	)
}

type serviceMainValues struct {
	PkgPath     string
	Port        string
	ServiceName string
}

var serviceMain string = `package main

import (
	"{{.PkgPath}}/grpcservice/impl"
	"github.com/vaporz/turbo"
)

func main() {
	turbo.StartGrpcService({{.Port}}, impl.RegisterServer)
}
`

func generateThriftServiceMain() {
	nameLower := strings.ToLower(Config.ThriftServiceName())
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/thriftservice/"+nameLower+".go",
		thriftServiceMain,
		thriftServiceMainValues{PkgPath: Config.ServicePkgPath(), Port: "50052", ServiceName: Config.ThriftServiceName()},
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
	"github.com/vaporz/turbo"
)

func main() {
	turbo.StartThriftService({{.Port}}, impl.TProcessor)
}
`

func generateGrpcServiceImpl() {
	nameLower := strings.ToLower(Config.GrpcServiceName())
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/grpcservice/impl/"+nameLower+"impl.go",
		serviceImpl,
		serviceImplValues{PkgPath: Config.ServicePkgPath(), ServiceName: Config.GrpcServiceName()},
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
	"google.golang.org/grpc"
)

func RegisterServer(s *grpc.Server) {
	proto.RegisterTestServiceServer(s, &{{.ServiceName}}{})
}

type {{.ServiceName}} struct {
}

func (s *{{.ServiceName}}) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
}
`

func generateThriftServiceImpl() {
	nameLower := strings.ToLower(Config.ThriftServiceName())
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/thriftservice/impl/"+nameLower+"impl.go",
		thriftServiceImpl,
		thriftServiceImplValues{PkgPath: Config.ServicePkgPath(), ServiceName: Config.ThriftServiceName()},
	)
}

type thriftServiceImplValues struct {
	PkgPath     string
	ServiceName string
}

var thriftServiceImpl string = `package impl

import (
	"{{.PkgPath}}/gen/thrift/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func TProcessor() thrift.TProcessor {
	return gen.New{{.ServiceName}}Processor({{.ServiceName}}{})
}

type {{.ServiceName}} struct {
}

func (s {{.ServiceName}}) SayHello(yourName string) (r *gen.SayHelloResponse, err error) {
	return &gen.SayHelloResponse{Message: "[thrift server]Hello, " + yourName}, nil
}
`

func generateGrpcHTTPMain() {
	nameLower := strings.ToLower(Config.GrpcServiceName())
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/grpcapi/"+nameLower+"api.go",
		_HTTPMain,
		_HTTPMainValues{ServiceName: Config.GrpcServiceName(), PkgPath: Config.ServicePkgPath()},
	)
}

type _HTTPMainValues struct {
	ServiceName string
	PkgPath     string
}

var _HTTPMain string = `package main

import (
	"{{.PkgPath}}/gen"
	"{{.PkgPath}}/grpcapi/component"
	"github.com/vaporz/turbo"
)

func main() {
	component.InitComponents()
	turbo.StartGrpcHTTPServer("{{.PkgPath}}", "service", component.GrpcClient, gen.GrpcSwitcher)
}
`

func generateGrpcHTTPComponent() {
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/grpcapi/component/components.go",
		_HTTPComponent,
		_HTTPComponentValues{ServiceName: Config.GrpcServiceName(), PkgPath: Config.ServicePkgPath()},
	)
}

type _HTTPComponentValues struct {
	ServiceName string
	PkgPath     string
}

var _HTTPComponent string = `package component

import (
	"{{.PkgPath}}/gen/proto"
	"google.golang.org/grpc"
)

func GrpcClient(conn *grpc.ClientConn) interface{} {
	return proto.New{{.ServiceName}}Client(conn)
}

func InitComponents() {
}
`

func generateThriftHTTPComponent() {
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/thriftapi/component/components.go",
		thrfitHTTPComponent,
		thriftHTTPComponentValues{ServiceName: Config.GrpcServiceName(), PkgPath: Config.ServicePkgPath()},
	)
}

type thriftHTTPComponentValues struct {
	ServiceName string
	PkgPath     string
}

var thrfitHTTPComponent string = `package component

import (
	t "{{.PkgPath}}/gen/thrift/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.New{{.ServiceName}}ClientFactory(trans, f)
}

func InitComponents() {
}
`

func generateThriftHTTPMain() {
	nameLower := strings.ToLower(Config.ThriftServiceName())
	writeFileWithTemplate(
		Config.ServiceRootPath()+"/thriftapi/"+nameLower+"api.go",
		thriftHTTPMain,
		_HTTPMainValues{ServiceName: Config.ThriftServiceName(), PkgPath: Config.ServicePkgPath()},
	)
}

var thriftHTTPMain string = `package main

import (
	"github.com/vaporz/turbo"
	"{{.PkgPath}}/gen"
	"{{.PkgPath}}/thriftapi/component"
)

func main() {
	component.InitComponents()
	turbo.StartThriftHTTPServer("{{.PkgPath}}", "service",
		component.ThriftClient, gen.ThriftSwitcher)
}
`

func generateServiceMain(rpcType string) {
	if rpcType == "grpc" {
		writeFileWithTemplate(
			Config.ServiceRootPath()+"/main.go",
			rootMain_grpc,
			rootMainValues{ServiceName: Config.ThriftServiceName(), PkgPath: Config.ServicePkgPath()},
		)
	} else if rpcType == "thrift" {
		writeFileWithTemplate(
			Config.ServiceRootPath()+"/main.go",
			rootMain_thrift,
			rootMainValues{ServiceName: Config.ThriftServiceName(), PkgPath: Config.ServicePkgPath()},
		)
	}
}

type rootMainValues struct {
	ServiceName string
	PkgPath     string
}

var rootMain_grpc string = `package main

import (
	"github.com/vaporz/turbo"
	"{{.PkgPath}}/gen"
	gcomponent "{{.PkgPath}}/grpcapi/component"
	gimpl "{{.PkgPath}}/grpcservice/impl"
	//tcompoent "{{.PkgPath}}/thriftapi/component"
	//timpl "{{.PkgPath}}/thriftservice/impl"
)

func main() {
	gcomponent.InitComponents()
	turbo.StartGRPC("{{.PkgPath}}", "service",
		50051, gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	//tcompoent.InitComponents()
	//turbo.StartTHRIFT("{{.PkgPath}}", "service",
	//	50052, tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
`

var rootMain_thrift string = `package main

import (
	"github.com/vaporz/turbo"
	"{{.PkgPath}}/gen"
	//gcomponent "{{.PkgPath}}/grpcapi/component"
	//gimpl "{{.PkgPath}}/grpcservice/impl"
	tcompoent "{{.PkgPath}}/thriftapi/component"
	timpl "{{.PkgPath}}/thriftservice/impl"
)

func main() {
	//gcomponent.InitComponents()
	//turbo.StartGRPC("{{.PkgPath}}", "service",
	//	50051, gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	tcompoent.InitComponents()
	turbo.StartTHRIFT("{{.PkgPath}}", "service",
		50052, tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
`
