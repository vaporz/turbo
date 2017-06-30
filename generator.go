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

// Generator generates new projects or proto/thrift code
type Generator struct {
	RpcType        string
	PkgPath        string
	ConfigFileName string
	Options        string
	c              *Config
}

// Generate proto/thrift code
func (g *Generator) Generate() {
	if g.RpcType != "grpc" && g.RpcType != "thrift" {
		panic("Invalid server type, should be (grpc|thrift)")
	}
	g.c = &Config{RpcType: g.RpcType, GOPATH: GOPATH()}
	g.c.loadServiceConfig(g.c.GOPATH + "/src/" + g.PkgPath + "/" + g.ConfigFileName + ".yaml")
	if g.RpcType == "grpc" {
		g.GenerateProtobufStub()
		g.c.loadFieldMapping()
		g.GenerateGrpcSwitcher()
	} else if g.RpcType == "thrift" {
		g.GenerateThriftStub()
		g.GenerateBuildThriftParameters()
		g.c.loadFieldMapping()
		g.GenerateThriftSwitcher()
	}
}

// CreateProject creates a whole new project!
func (g *Generator) CreateProject(serviceName string, force bool) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	if !force {
		g.validateServiceRootPath(nil)
	}
	g.createRootFolder(GOPATH() + "/src/" + g.PkgPath)
	g.createServiceYaml(GOPATH()+"/src/"+g.PkgPath, serviceName, "service")
	g.c = &Config{RpcType: g.RpcType, GOPATH: GOPATH()}
	g.c.loadServiceConfig(g.c.GOPATH + "/src/" + g.PkgPath + "/service.yaml")
	if g.RpcType == "grpc" {
		g.createGrpcProject(serviceName)
	} else if g.RpcType == "thrift" {
		g.createThriftProject(serviceName)
	}
}

func (g *Generator) validateServiceRootPath(in io.Reader) {
	if in == nil {
		in = os.Stdin
	}
	_, err := os.Stat(g.c.ServiceRootPath())
	if os.IsNotExist(err) {
		return
	}
	fmt.Print("Path '" + g.c.ServiceRootPath() + "' already exist!\n" +
		"Do you want to remove this directory before creating a new project? (type 'y' to remove):")
	var input string
	fmt.Fscan(in, &input)
	if input != "y" {
		return
	}
	fmt.Print("All files in that directory will be lost, are you sure? (type 'y' to continue):")
	fmt.Fscan(in, &input)
	if input != "y" {
		panic("aborted")
	}
	os.RemoveAll(g.c.ServiceRootPath())
}

func (g *Generator) createGrpcProject(serviceName string) {
	g.createGrpcFolders()
	g.createProto(serviceName)
	g.Options = " -I " + g.c.ServiceRootPath() + " " + g.c.ServiceRootPath() + "/" + strings.ToLower(serviceName) + ".proto "
	g.GenerateProtobufStub()
	g.GenerateGrpcSwitcher()
	g.generateGrpcServiceMain()
	g.generateGrpcServiceImpl()
	g.generateGrpcHTTPMain()
	g.generateGrpcHTTPComponent()
	g.generateServiceMain("grpc")
}

func (g *Generator) createThriftProject(serviceName string) {
	g.createThriftFolders()
	g.createThrift(serviceName)
	g.Options = " -I " + g.c.ServiceRootPath() + " "
	g.GenerateThriftStub()
	g.GenerateBuildThriftParameters()
	g.GenerateThriftSwitcher()
	g.generateThriftServiceMain()
	g.generateThriftServiceImpl()
	g.generateThriftHTTPMain()
	g.generateThriftHTTPComponent()
	g.generateServiceMain("thrift")
}

func (g *Generator) createRootFolder(serviceRootPath string) {
	os.MkdirAll(serviceRootPath+"/gen", 0755)
}

func (g *Generator) createGrpcFolders() {
	os.MkdirAll(g.c.ServiceRootPath()+"/gen/proto", 0755)
	os.MkdirAll(g.c.ServiceRootPath()+"/grpcapi/component", 0755)
	os.MkdirAll(g.c.ServiceRootPath()+"/grpcservice/impl", 0755)
}

func (g *Generator) createThriftFolders() {
	os.MkdirAll(g.c.ServiceRootPath()+"/gen/thrift", 0755)
	os.MkdirAll(g.c.ServiceRootPath()+"/thriftapi/component", 0755)
	os.MkdirAll(g.c.ServiceRootPath()+"/thriftservice/impl", 0755)
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

func (g *Generator) createServiceYaml(serviceRootPath, serviceName, configFileName string) {
	if _, err := os.Stat(serviceRootPath + "/" + configFileName + ".yaml"); err == nil {
		return
	}
	writeFileWithTemplate(
		serviceRootPath+"/"+configFileName+".yaml",
		serviceYamlFile,
		serviceYamlValues{ServiceRoot: serviceRootPath, ServiceName: serviceName},
	)
}

type serviceYamlValues struct {
	ServiceRoot string
	ServiceName string
}

var serviceYamlFile string = `config:
  environment: development
  service_root_path: {{.ServiceRoot}}
  turbo_log_path: log
  http_port: 8081
  grpc_service_name: {{.ServiceName}}
  grpc_service_host: 127.0.0.1
  grpc_service_port: 50051
  thrift_service_name: {{.ServiceName}}
  thrift_service_host: 127.0.0.1
  thrift_service_port: 50052

urlmapping:
  - GET /hello SayHello
`

func (g *Generator) createProto(serviceName string) {
	nameLower := strings.ToLower(serviceName)
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/"+nameLower+".proto",
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

func (g *Generator) createThrift(serviceName string) {
	nameLower := strings.ToLower(serviceName)
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/"+nameLower+".thrift",
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
func (g *Generator) GenerateGrpcSwitcher() {
	if _, err := os.Stat(g.c.ServiceRootPath() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(g.c.ServiceRootPath()+"/gen", 0755)
	}
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range g.c.urlServiceMaps {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		var casesBuf bytes.Buffer
		writeWithTemplate(
			&casesBuf,
			grpcCases,
			method{
				ServiceName:  g.c.GrpcServiceName(),
				MethodName:   v,
				StructFields: g.structFields(v + "Request")},
		)
		casesStr = casesStr + casesBuf.String()
	}
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/gen/grpcswitcher.go",
		grpcSwitcherFunc,
		handlerContent{
			Cases:   casesStr,
			PkgPath: g.PkgPath},
	)
}

func (g *Generator) structFields(structName string) string {
	fields, ok := g.c.fieldMappings[structName]
	if !ok {
		return ""
	}
	var fieldStr string
	for _, field := range fields {
		if len(strings.TrimSpace(field)) == 0 {
			continue
		}
		pair := strings.Split(field, " ")
		nameSlice := []rune(pair[1])
		name := strings.ToUpper(string(nameSlice[0])) + string(nameSlice[1:])
		typeName := pair[0]
		fieldStr = fieldStr + name + ": &g." + typeName + "{" + g.structFields(typeName) + "},"
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
// GrpcSwitcher is a runtime func with which a server starts.
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
func (g *Generator) GenerateProtobufStub() {
	if _, err := os.Stat(g.c.ServiceRootPath() + "/gen/proto"); os.IsNotExist(err) {
		os.MkdirAll(g.c.ServiceRootPath()+"/gen/proto", 0755)
	}
	cmd := "protoc " + g.Options + " --go_out=plugins=grpc:" + g.c.ServiceRootPath() + "/gen/proto" +
		" --buildfields_out=service_root_path=" + g.c.ServiceRootPath() + ":" + g.c.ServiceRootPath() + "/gen/proto"

	executeCmd("bash", "-c", cmd)
}

// GenerateBuildThriftParameters generates "build.go"
func (g *Generator) GenerateBuildThriftParameters() {
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range g.c.urlServiceMaps {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		var casesBuf bytes.Buffer
		writeWithTemplate(
			&casesBuf,
			thriftParameterCases,
			thriftParameterCasesValues{
				ServiceName: g.c.ThriftServiceName(),
				MethodName:  v},
		)
		casesStr = casesStr + casesBuf.String()
	}

	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/gen/thrift/build.go",
		buildThriftParameters,
		buildThriftParametersValues{
			PkgPath:         g.PkgPath,
			ServiceName:     g.c.GrpcServiceName(),
			Cases:           casesStr,
			ServiceRootPath: g.c.ServiceRootPath()},
	)
	g.runBuildThriftFields()
}

func (g *Generator) runBuildThriftFields() {
	cmd := "go run " + g.c.ServiceRootPath() + "/gen/thrift/build.go"
	c := exec.Command("bash", "-c", cmd)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	if err := c.Run(); err != nil {
		panic(err)
	}
}

type thriftParameterCasesValues struct {
	MethodName  string
	ServiceName string
}

var thriftParameterCases string = `	case "{{.MethodName}}":
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

type buildThriftParametersValues struct {
	PkgPath         string
	ServiceName     string
	Cases           string
	ServiceRootPath string
}

var buildThriftParameters string = `package main

import (
	"flag"
	"fmt"
	g "{{.PkgPath}}/gen/thrift/gen-go/gen"
	"io"
	"os"
	"reflect"
	"strings"
	"text/template"
)

var methodName = flag.String("n", "", "")

func main() {
	flag.Parse()
	if len(strings.TrimSpace(*methodName)) > 0 {
		str := buildParameterStr(*methodName)
		fmt.Print(str)
	} else {
		buildFields()
	}
}

func buildFields() {
	i := new(g.{{.ServiceName}})
	t := reflect.TypeOf(i).Elem()
	numMethod := t.NumMethod()
	items := make([]string, 0)
	for i := 0; i < numMethod; i++ {
		method := t.Method(i)
		numIn := method.Type.NumIn()
		for j := 0; j < numIn; j++ {
			argType := method.Type.In(j)
			argStr := argType.String()
			if argType.Kind() == reflect.Ptr && argType.Elem().Kind() == reflect.Struct {
				arr := strings.Split(argStr, ".")
				name := arr[len(arr)-1:][0]
				items = findItem(items, name, argType)
			}
		}
	}
	var list string
	for _, s := range items {
		list += s + "\n"
	}
	writeFileWithTemplate(
		"{{.ServiceRootPath}}/gen/thriftfields.yaml",
		fieldsYaml,
		fieldsYamlValues{List: list},
	)
}

func findItem(items []string, name string, structType reflect.Type) []string {
	numField := structType.Elem().NumField()
	item := "  - " + name + "["
	for i := 0; i < numField; i++ {
		fieldType := structType.Elem().Field(i)
		if fieldType.Type.Kind() == reflect.Ptr && fieldType.Type.Elem().Kind() == reflect.Struct {
			arr := strings.Split(fieldType.Type.String(), ".")
			typeName := arr[len(arr)-1:][0]
			argName := fieldType.Name
			item += fmt.Sprintf("%s %s,", typeName, argName)
			items = findItem(items, typeName, fieldType.Type)
		}
	}
	item += "]"
	return append(items, item)
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

type fieldsYamlValues struct {
	List string
}

var fieldsYaml string = ` + "`" + `thrift-fieldmapping:
{{printf "%s" "{{.List}}"}}
` + "`" + `

func buildParameterStr(methodName string) string {
	switch methodName {
{{.Cases}}
	default:
		return "error"
	}
}
`

// GenerateThriftSwitcher generates "thriftswitcher.go"
func (g *Generator) GenerateThriftSwitcher() {
	if _, err := os.Stat(g.c.ServiceRootPath() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(g.c.ServiceRootPath()+"/gen", 0755)
	}
	var casesStr string
	methodNames := make(map[string]int)
	for _, v := range g.c.urlServiceMaps {
		methodNames[v[2]] = 0
	}
	for v := range methodNames {
		parametersStr := g.thriftParameters(v)
		var casesBuf bytes.Buffer
		writeWithTemplate(
			&casesBuf,
			thriftCases,
			thriftMethod{
				ServiceName:        g.c.ThriftServiceName(),
				MethodName:         v,
				Parameters:         parametersStr,
				NotEmptyParameters: len(strings.TrimSpace(parametersStr)) > 0},
		)
		casesStr = casesStr + casesBuf.String()
	}

	var argCasesStr string
	for k := range g.c.fieldMappings {
		var argCasesBuf bytes.Buffer
		writeWithTemplate(
			&argCasesBuf,
			buildArgCases,
			buildArgParams{k, g.structFields(k)},
		)
		argCasesStr = argCasesStr + argCasesBuf.String()
	}
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/gen/thriftswitcher.go",
		thriftSwitcherFunc,
		thriftHandlerContent{
			Cases:          casesStr,
			PkgPath:        g.PkgPath,
			BuildArgsCases: argCasesStr},
	)
}

func (g *Generator) thriftParameters(methodName string) string {
	cmd := "go run " + g.c.ServiceRootPath() + "/gen/thrift/build.go -n " + methodName
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
// ThriftSwitcher is a runtime func with which a server starts.
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
func (g *Generator) GenerateThriftStub() {
	if _, err := os.Stat(g.c.ServiceRootPath() + "/gen/thrift"); os.IsNotExist(err) {
		os.MkdirAll(g.c.ServiceRootPath()+"/gen/thrift", 0755)
	}
	nameLower := strings.ToLower(g.c.ThriftServiceName())
	cmd := "thrift " + g.Options + " -r --gen go:package_prefix=" + g.PkgPath + "/gen/thrift/gen-go/ -o" +
		" " + g.c.ServiceRootPath() + "/" + "gen/thrift " + g.c.ServiceRootPath() + "/" + nameLower + ".thrift"
	executeCmd("bash", "-c", cmd)
}

func executeCmd(cmd string, args ...string) {
	// TODO learn
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	if err := c.Run(); err != nil {
		panic(err)
	}
}

func (g *Generator) generateGrpcServiceMain() {
	nameLower := strings.ToLower(g.c.GrpcServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/grpcservice/"+nameLower+".go",
		serviceMain,
		serviceMainValues{PkgPath: g.PkgPath, ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
	)
}

type serviceMainValues struct {
	PkgPath        string
	ConfigFilePath string
}

var serviceMain string = `package main

import (
	"{{.PkgPath}}/grpcservice/impl"
	"github.com/vaporz/turbo"
)

func main() {
	turbo.StartGrpcService("{{.ConfigFilePath}}", impl.RegisterServer)
}
`

func (g *Generator) generateThriftServiceMain() {
	nameLower := strings.ToLower(g.c.ThriftServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/thriftservice/"+nameLower+".go",
		thriftServiceMain,
		thriftServiceMainValues{PkgPath: g.PkgPath, ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
	)
}

type thriftServiceMainValues struct {
	PkgPath        string
	ConfigFilePath string
}

var thriftServiceMain string = `package main

import (
	"{{.PkgPath}}/thriftservice/impl"
	"github.com/vaporz/turbo"
)

func main() {
	turbo.StartThriftService("{{.ConfigFilePath}}", "service", impl.TProcessor)
}
`

func (g *Generator) generateGrpcServiceImpl() {
	nameLower := strings.ToLower(g.c.GrpcServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/grpcservice/impl/"+nameLower+"impl.go",
		serviceImpl,
		serviceImplValues{PkgPath: g.PkgPath, ServiceName: g.c.GrpcServiceName()},
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

// RegisterServer registers a service struct to a server
func RegisterServer(s *grpc.Server) {
	proto.Register{{.ServiceName}}Server(s, &{{.ServiceName}}{})
}

// {{.ServiceName}} is the struct which implements generated interface
type {{.ServiceName}} struct {
}

// SayHello is an example entry point
func (s *{{.ServiceName}}) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
}
`

func (g *Generator) generateThriftServiceImpl() {
	nameLower := strings.ToLower(g.c.ThriftServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/thriftservice/impl/"+nameLower+"impl.go",
		thriftServiceImpl,
		thriftServiceImplValues{PkgPath: g.PkgPath, ServiceName: g.c.ThriftServiceName()},
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

// TProcessor returns TProcessor
func TProcessor() thrift.TProcessor {
	return gen.New{{.ServiceName}}Processor({{.ServiceName}}{})
}

// {{.ServiceName}} is the struct which implements generated interface
type {{.ServiceName}} struct {
}

// SayHello is an example entry point
func (s {{.ServiceName}}) SayHello(yourName string) (r *gen.SayHelloResponse, err error) {
	return &gen.SayHelloResponse{Message: "[thrift server]Hello, " + yourName}, nil
}
`

func (g *Generator) generateGrpcHTTPMain() {
	nameLower := strings.ToLower(g.c.GrpcServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/grpcapi/"+nameLower+"api.go",
		_HTTPMain,
		_HTTPMainValues{
			ServiceName:    g.c.GrpcServiceName(),
			PkgPath:        g.PkgPath,
			ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
	)
}

type _HTTPMainValues struct {
	ServiceName    string
	PkgPath        string
	ConfigFilePath string
}

var _HTTPMain string = `package main

import (
	"{{.PkgPath}}/gen"
	"{{.PkgPath}}/grpcapi/component"
	"github.com/vaporz/turbo"
)

func main() {
	component.InitComponents()
	turbo.StartGrpcHTTPServer("{{.ConfigFilePath}}", component.GrpcClient, gen.GrpcSwitcher)
}
`

func (g *Generator) generateGrpcHTTPComponent() {
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/grpcapi/component/components.go",
		_HTTPComponent,
		_HTTPComponentValues{ServiceName: g.c.GrpcServiceName(), PkgPath: g.PkgPath},
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

// GrpcClient returns a grpc client
func GrpcClient(conn *grpc.ClientConn) interface{} {
	return proto.New{{.ServiceName}}Client(conn)
}

// InitComponents inits turbo components, such as interceptors, pre/postprocessors, errorHandlers, etc.
func InitComponents() {
}
`

func (g *Generator) generateThriftHTTPComponent() {
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/thriftapi/component/components.go",
		thrfitHTTPComponent,
		thriftHTTPComponentValues{ServiceName: g.c.GrpcServiceName(), PkgPath: g.PkgPath},
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

// ThriftClient returns a thrift client
func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.New{{.ServiceName}}ClientFactory(trans, f)
}

// InitComponents inits turbo components, such as interceptors, pre/postprocessors, errorHandlers, etc.
func InitComponents() {
}
`

func (g *Generator) generateThriftHTTPMain() {
	nameLower := strings.ToLower(g.c.ThriftServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/thriftapi/"+nameLower+"api.go",
		thriftHTTPMain,
		_HTTPMainValues{
			ServiceName:    g.c.ThriftServiceName(),
			PkgPath:        g.PkgPath,
			ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
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
	turbo.StartThriftHTTPServer("{{.ConfigFilePath}}", component.ThriftClient, gen.ThriftSwitcher)
}
`

func (g *Generator) generateServiceMain(rpcType string) {
	if rpcType == "grpc" {
		writeFileWithTemplate(
			g.c.ServiceRootPath()+"/main.go",
			rootMainGrpc,
			rootMainValues{PkgPath: g.PkgPath, ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
		)
	} else if rpcType == "thrift" {
		writeFileWithTemplate(
			g.c.ServiceRootPath()+"/main.go",
			rootMainThrift,
			rootMainValues{PkgPath: g.PkgPath, ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
		)
	}
}

type rootMainValues struct {
	PkgPath        string
	ConfigFilePath string
}

var rootMainGrpc string = `package main

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
	turbo.StartGRPC("{{.ConfigFilePath}}", gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	//tcompoent.InitComponents()
	//turbo.StartTHRIFT("{{.ConfigFilePath}}", tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
`

var rootMainThrift string = `package main

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
	//turbo.StartGRPC("{{.ConfigFilePath}}", gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	tcompoent.InitComponents()
	turbo.StartTHRIFT("{{.ConfigFilePath}}", tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
`
