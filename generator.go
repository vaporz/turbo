package turbo

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

// Generator generates proto/thrift code
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

func writeFileWithTemplate(filePath string, data interface{}, text string) {
	f, err := os.Create(filePath)
	panicIf(err)

	tmpl, err := template.New("").Parse(text)
	panicIf(err)

	err = tmpl.Execute(f, data)
	panicIf(err)
}

// GenerateGrpcSwitcher generates "grpcswither.go"
func (g *Generator) GenerateGrpcSwitcher() {
	type handlerContent struct {
		MethodNames  []string
		PkgPath      string
		ServiceName  string
		StructFields []string
	}
	if _, err := os.Stat(g.c.ServiceRootPathAbsolute() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(g.c.ServiceRootPathAbsolute()+"/gen", 0755)
	}
	methodNames := methodNames(g.c.mappings[urlServiceMaps])
	structFields := make([]string, len(methodNames))
	for i, v := range methodNames {
		structFields[i] = g.structFields(v + "Request")
	}
	writeFileWithTemplate(
		g.c.ServiceRootPathAbsolute()+"/gen/grpcswitcher.go",
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
	"net/http"
	"errors"
)

/*
this is a generated file, DO NOT EDIT!
 */
// GrpcSwitcher is a runtime func with which a server starts.
var GrpcSwitcher = func(s turbo.Servable, methodName string, resp http.ResponseWriter, req *http.Request) (rpcResponse interface{}, err error) {
	callOptions, header, trailer, peer := turbo.CallOptions(methodName, req)
	switch methodName { {{range $i, $MethodName := .MethodNames}}
	case "{{$MethodName}}":
		request := &g.{{$MethodName}}Request{ {{index $.StructFields $i}} }
		err = turbo.BuildRequest(s, request, req)
		if err != nil {
			return nil, err
		}
		rpcResponse, err = s.Service().(g.{{$.ServiceName}}Client).{{$MethodName}}(req.Context(), request, callOptions...){{end}}
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
	turbo.WithCallOptions(req, header, trailer, peer)
	return
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
	if _, err := os.Stat(g.c.ServiceRootPathAbsolute() + "/gen/proto"); os.IsNotExist(err) {
		os.MkdirAll(g.c.ServiceRootPathAbsolute()+"/gen/proto", 0755)
	}
	cmd := "protoc " + g.Options + " --go_out=plugins=grpc:" + g.c.ServiceRootPathAbsolute() + "/gen/proto" +
		" --buildfields_out=service_root_path=" + g.c.ServiceRootPathAbsolute() + ":" + g.c.ServiceRootPathAbsolute() + "/gen/proto"

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
		g.c.ServiceRootPathAbsolute()+"/gen/thrift/build.go",
		buildThriftParametersValues{
			PkgPath:         g.PkgPath,
			ServiceName:     g.c.GrpcServiceName(),
			ServiceRootPath: g.c.ServiceRootPathAbsolute(),
			MethodNames:     methodNames(g.c.mappings[urlServiceMaps])},
		buildThriftParameters,
	)
	g.runBuildThriftFields()
}

func (g *Generator) runBuildThriftFields() {
	cmd := "go run " + g.c.ServiceRootPathAbsolute() + "/gen/thrift/build.go"
	c := exec.Command("bash", "-c", cmd)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	panicIf(c.Run())
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
	if _, err := os.Stat(g.c.ServiceRootPathAbsolute() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(g.c.ServiceRootPathAbsolute()+"/gen", 0755)
	}
	methodNames := methodNames(g.c.mappings[urlServiceMaps])
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
		g.c.ServiceRootPathAbsolute()+"/gen/thriftswitcher.go",
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
	cmd := "go run " + g.c.ServiceRootPathAbsolute() + "/gen/thrift/build.go -n " + methodName
	buf := &bytes.Buffer{}
	c := exec.Command("bash", "-c", cmd)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = buf
	panicIf(c.Run())
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
var ThriftSwitcher = func(s turbo.Servable, methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) {
	switch methodName {
{{range $i, $MethodName := .MethodNames}}
	case "{{$MethodName}}":{{if index $.NotEmptyParameters $i }}
		params, err := turbo.BuildThriftRequest(s, gen.{{$.ServiceName}}{{$MethodName}}Args{}, req, buildStructArg)
		if err != nil {
			return nil, err
		}{{end}}
		return s.Service().(*gen.{{$.ServiceName}}Client).{{$MethodName}}({{index $.Parameters $i}})
{{end}}
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
}

func buildStructArg(s turbo.Servable, typeName string, req *http.Request) (v reflect.Value, err error) {
	switch typeName {
{{range $i, $StructName := .StructNames}}
	case "{{$StructName}}":
		request := &gen.{{$StructName}}{ {{index $.StructFields $i}} }
		turbo.BuildStruct(s, reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req)
		return reflect.ValueOf(request), nil
{{end}}
	default:
		return v, errors.New("unknown typeName[" + typeName + "]")
	}
}
`

// GenerateThriftStub generates Thrift stub codes
func (g *Generator) GenerateThriftStub() {
	if _, err := os.Stat(g.c.ServiceRootPathAbsolute() + "/gen/thrift"); os.IsNotExist(err) {
		os.MkdirAll(g.c.ServiceRootPathAbsolute()+"/gen/thrift", 0755)
	}
	nameLower := strings.ToLower(g.c.ThriftServiceName())
	cmd := "thrift " + g.Options + " -r --gen go:package_prefix=" + g.PkgPath + "/gen/thrift/gen-go/ -o" +
		" " + g.c.ServiceRootPathAbsolute() + "/" + "gen/thrift " + g.c.ServiceRootPathAbsolute() + "/" + nameLower + ".thrift"
	executeCmd("bash", "-c", cmd)
}

func executeCmd(cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	panicIf(c.Run())
}
