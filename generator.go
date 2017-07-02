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
	g.c = NewConfig(g.RpcType, GOPATH()+"/src/"+g.PkgPath+"/"+g.ConfigFileName+".yaml")
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
	g.c = NewConfig(g.RpcType, GOPATH()+"/src/"+g.PkgPath+"/service.yaml")
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

func writeFileWithTemplate(filePath string, data interface{}, text string) {
	f, err := os.Create(filePath)
	if err != nil {
		panic("fail to create file:" + filePath)
	}
	tmpl, err := template.New("").Parse(text)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(f, data)
	if err != nil {
		panic(err)
	}
}

func (g *Generator) createServiceYaml(serviceRootPath, serviceName, configFileName string) {
	type serviceYamlValues struct {
		ServiceRoot string
		ServiceName string
	}
	if _, err := os.Stat(serviceRootPath + "/" + configFileName + ".yaml"); err == nil {
		return
	}
	writeFileWithTemplate(
		serviceRootPath+"/"+configFileName+".yaml",
		serviceYamlValues{ServiceRoot: serviceRootPath, ServiceName: serviceName},
		`config:
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
`)
}

func (g *Generator) createProto(serviceName string) {
	type protoValues struct {
		ServiceName string
	}
	nameLower := strings.ToLower(serviceName)
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/"+nameLower+".proto",
		protoValues{ServiceName: serviceName},
		`syntax = "proto3";
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
`,
	)
}

func (g *Generator) createThrift(serviceName string) {
	type thriftValues struct {
		ServiceName string
	}
	nameLower := strings.ToLower(serviceName)
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/"+nameLower+".thrift",
		thriftValues{ServiceName: serviceName},
		`namespace go gen

struct SayHelloResponse {
  1: string message,
}

service {{.ServiceName}} {
    SayHelloResponse sayHello (1:string yourName)
}
`,
	)
}

// GenerateGrpcSwitcher generates "grpcswither.go"
func (g *Generator) GenerateGrpcSwitcher() {
	type handlerContent struct {
		MethodNames  []string
		PkgPath      string
		ServiceName  string
		StructFields []string
	}
	if _, err := os.Stat(g.c.ServiceRootPath() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(g.c.ServiceRootPath()+"/gen", 0755)
	}
	methodNames := methodNames(g.c.urlServiceMaps)
	structFields := make([]string, len(methodNames))
	for i, v := range methodNames {
		structFields[i] = g.structFields(v + "Request")
	}
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/gen/grpcswitcher.go",
		handlerContent{
			MethodNames:  methodNames,
			PkgPath:      g.PkgPath,
			ServiceName:  g.c.GrpcServiceName(),
			StructFields: structFields,
		},
		`package gen

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
var GrpcSwitcher = func(s *turbo.Server, methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName {
{{range $i, $MethodName := .MethodNames}}
	case "{{$MethodName}}":
		request := &g.{{$MethodName}}Request{ {{index $.StructFields $i}} }
		err = turbo.BuildStruct(s, reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return nil, err
		}
		return turbo.GrpcService(s).(g.{{$.ServiceName}}Client).{{$MethodName}}(req.Context(), request)
{{end}}
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}
`)
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
	type buildThriftParametersValues struct {
		PkgPath         string
		ServiceName     string
		ServiceRootPath string
		MethodNames     []string
	}
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/gen/thrift/build.go",
		buildThriftParametersValues{
			PkgPath:         g.PkgPath,
			ServiceName:     g.c.GrpcServiceName(),
			ServiceRootPath: g.c.ServiceRootPath(),
			MethodNames:     methodNames(g.c.urlServiceMaps)},
		buildThriftParameters,
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
{{range $i, $MethodName := .MethodNames}}
	case "{{$MethodName}}":
		var result string
		args := g.{{$.ServiceName}}{{$MethodName}}Args{}
		at := reflect.TypeOf(args)
		num := at.NumField()
		for i := 0; i < num; i++ {
			result += fmt.Sprintf(
				"\n\t\t\tparams[%d].Interface().(%s),",
				i, at.Field(i).Type.String())
		}
		return result
{{end}}
	default:
		return "error"
	}
}
`

// GenerateThriftSwitcher generates "thriftswitcher.go"
func (g *Generator) GenerateThriftSwitcher() {
	type thriftHandlerContent struct {
		PkgPath            string
		BuildArgsCases     string
		ServiceName        string
		MethodNames        []string
		Parameters         []string
		NotEmptyParameters []bool
		StructNames        []string
		StructFields       []string
	}
	if _, err := os.Stat(g.c.ServiceRootPath() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(g.c.ServiceRootPath()+"/gen", 0755)
	}
	methodNames := methodNames(g.c.urlServiceMaps)
	parameters := make([]string, 0, len(methodNames))
	notEmptyParameters := make([]bool, 0, len(methodNames))
	for _, v := range methodNames {
		p := g.thriftParameters(v)
		parameters = append(parameters, p)
		notEmptyParameters = append(notEmptyParameters, len(strings.TrimSpace(p)) > 0)
	}

	var argCasesStr string
	fields := make([]string, 0, len(g.c.fieldMappings))
	structNames := make([]string, 0, len(g.c.fieldMappings))
	for k := range g.c.fieldMappings {
		structNames = append(structNames, k)
		fields = append(fields, g.structFields(k))
	}
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/gen/thriftswitcher.go",
		thriftHandlerContent{
			PkgPath:            g.PkgPath,
			BuildArgsCases:     argCasesStr,
			ServiceName:        g.c.ThriftServiceName(),
			MethodNames:        methodNames,
			Parameters:         parameters,
			NotEmptyParameters: notEmptyParameters,
			StructNames:        structNames,
			StructFields:       fields},
		thriftSwitcherFunc,
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

func methodNames(urlServiceMaps [][3]string) []string {
	methodNamesMap := make(map[string]int)
	for _, v := range urlServiceMaps {
		methodNamesMap[v[2]] = 0
	}
	methodNames := make([]string, 0, len(methodNamesMap))
	for k := range methodNamesMap {
		methodNames = append(methodNames, k)
	}
	return methodNames
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
var ThriftSwitcher = func(s *turbo.Server, methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName {
{{range $i, $MethodName := .MethodNames}}
	case "{{$MethodName}}":{{if index $.NotEmptyParameters $i }}
		args := gen.{{$.ServiceName}}{{$MethodName}}Args{}
		params, err := turbo.BuildArgs(s, reflect.TypeOf(args), reflect.ValueOf(args), req, buildStructArg)
		if err != nil {
			return nil, err
		}{{end}}
		return turbo.ThriftService(s).(*gen.{{$.ServiceName}}Client).{{$MethodName}}({{index $.Parameters $i}})
{{end}}
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}

func buildStructArg(s *turbo.Server, typeName string, req *http.Request) (v reflect.Value, err error) {
	switch typeName {
{{range $i, $StructName := .StructNames}}
	case "{{$StructName}}":
		request := &gen.{{$StructName}}{ {{index $.StructFields $i}} }
		err = turbo.BuildStruct(s, reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		if err != nil {
			return v, err
		}
		return reflect.ValueOf(request), nil
{{end}}
	default:
		return v, errors.New("unknown typeName[" + typeName + "]")
	}
}
`

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
	type serviceMainValues struct {
		PkgPath        string
		ConfigFilePath string
	}
	nameLower := strings.ToLower(g.c.GrpcServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/grpcservice/"+nameLower+".go",
		serviceMainValues{PkgPath: g.PkgPath, ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
		`package main

import (
	"{{.PkgPath}}/grpcservice/impl"
	"github.com/vaporz/turbo"
)

func main() {
	s := turbo.NewGprcServer("{{.ConfigFilePath}}")
	s.StartGrpcService(impl.RegisterServer)
}
`,
	)
}

func (g *Generator) generateThriftServiceMain() {
	type thriftServiceMainValues struct {
		PkgPath        string
		ConfigFilePath string
	}
	nameLower := strings.ToLower(g.c.ThriftServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/thriftservice/"+nameLower+".go",
		thriftServiceMainValues{PkgPath: g.PkgPath, ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
		`package main

import (
	"{{.PkgPath}}/thriftservice/impl"
	"github.com/vaporz/turbo"
)

func main() {
	s := turbo.NewThriftServer("{{.ConfigFilePath}}")
	s.StartThriftService(impl.TProcessor)
}
`,
	)
}

func (g *Generator) generateGrpcServiceImpl() {
	type serviceImplValues struct {
		PkgPath     string
		ServiceName string
	}
	nameLower := strings.ToLower(g.c.GrpcServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/grpcservice/impl/"+nameLower+"impl.go",
		serviceImplValues{PkgPath: g.PkgPath, ServiceName: g.c.GrpcServiceName()},
		`package impl

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
`,
	)
}

func (g *Generator) generateThriftServiceImpl() {
	type thriftServiceImplValues struct {
		PkgPath     string
		ServiceName string
	}
	nameLower := strings.ToLower(g.c.ThriftServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/thriftservice/impl/"+nameLower+"impl.go",
		thriftServiceImplValues{PkgPath: g.PkgPath, ServiceName: g.c.ThriftServiceName()},
		`package impl

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
`,
	)
}

func (g *Generator) generateGrpcHTTPMain() {
	type HTTPMainValues struct {
		ServiceName    string
		PkgPath        string
		ConfigFilePath string
	}
	nameLower := strings.ToLower(g.c.GrpcServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/grpcapi/"+nameLower+"api.go",
		HTTPMainValues{
			ServiceName:    g.c.GrpcServiceName(),
			PkgPath:        g.PkgPath,
			ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
		`package main

import (
	"{{.PkgPath}}/gen"
	"{{.PkgPath}}/grpcapi/component"
	"github.com/vaporz/turbo"
)

func main() {
	s := turbo.NewGrpcServer("{{.ConfigFilePath}}")
	component.InitComponents(s)
	s.StartGrpcHTTPServer(component.GrpcClient, gen.GrpcSwitcher)
}
`,
	)
}

func (g *Generator) generateGrpcHTTPComponent() {
	type HTTPComponentValues struct {
		ServiceName string
		PkgPath     string
	}
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/grpcapi/component/components.go",
		HTTPComponentValues{ServiceName: g.c.GrpcServiceName(), PkgPath: g.PkgPath},
		`package component

import (
	"{{.PkgPath}}/gen/proto"
	"google.golang.org/grpc"
	"github.com/vaporz/turbo"
)

// GrpcClient returns a grpc client
func GrpcClient(conn *grpc.ClientConn) interface{} {
	return proto.New{{.ServiceName}}Client(conn)
}

// InitComponents inits turbo components, such as interceptors, pre/postprocessors, errorHandlers, etc.
func InitComponents(s *turbo.GrpcServer) {
}
`,
	)
}

func (g *Generator) generateThriftHTTPComponent() {
	type thriftHTTPComponentValues struct {
		ServiceName string
		PkgPath     string
	}
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/thriftapi/component/components.go",
		thriftHTTPComponentValues{ServiceName: g.c.GrpcServiceName(), PkgPath: g.PkgPath},
		`package component

import (
	t "{{.PkgPath}}/gen/thrift/gen-go/gen"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/vaporz/turbo"
)

// ThriftClient returns a thrift client
func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) interface{} {
	return t.New{{.ServiceName}}ClientFactory(trans, f)
}

// InitComponents inits turbo components, such as interceptors, pre/postprocessors, errorHandlers, etc.
func InitComponents(s *turbo.ThriftServer) {
}
`,
	)
}

func (g *Generator) generateThriftHTTPMain() {
	type HTTPMainValues struct {
		ServiceName    string
		PkgPath        string
		ConfigFilePath string
	}
	nameLower := strings.ToLower(g.c.ThriftServiceName())
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/thriftapi/"+nameLower+"api.go",
		HTTPMainValues{
			ServiceName:    g.c.ThriftServiceName(),
			PkgPath:        g.PkgPath,
			ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
		`package main

import (
	"github.com/vaporz/turbo"
	"{{.PkgPath}}/gen"
	"{{.PkgPath}}/thriftapi/component"
)

func main() {
	s := turbo.NewThriftServer({{.ConfigFilePath}}")
	component.InitComponents(s)
	s.StartThriftHTTPServer(component.ThriftClient, gen.ThriftSwitcher)
}
`,
	)
}

func (g *Generator) generateServiceMain(rpcType string) {
	type rootMainValues struct {
		PkgPath        string
		ConfigFilePath string
	}
	if rpcType == "grpc" {
		writeFileWithTemplate(
			g.c.ServiceRootPath()+"/main.go",
			rootMainValues{PkgPath: g.PkgPath, ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
			rootMainGrpc,
		)
	} else if rpcType == "thrift" {
		writeFileWithTemplate(
			g.c.ServiceRootPath()+"/main.go",
			rootMainValues{PkgPath: g.PkgPath, ConfigFilePath: g.c.ServiceRootPath() + "/service.yaml"},
			rootMainThrift,
		)
	}
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
	s := turbo.NewGrpcServer("{{.ConfigFilePath}}")
	gcomponent.InitComponents(s)
	s.StartGRPC(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	//s := turbo.NewThriftServer("{{.ConfigFilePath}}")
	//tcompoent.InitComponents(s)
	//s.StartTHRIFT(tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
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
	//s := turbo.NewGrpcServer("{{.ConfigFilePath}}")
	//gcomponent.InitComponents(s)
	//s.StartGRPC(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	s := turbo.NewThriftServer("{{.ConfigFilePath}}")
	tcompoent.InitComponents(s)
	s.StartTHRIFT(tcompoent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)
}
`
