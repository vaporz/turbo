/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
)

// Generator generates proto/thrift code
type Generator struct {
	RpcType        string
	PkgPath        string
	ConfigFileName string
	Options        string
	c              *Config
	FilePaths      []string
}

// Generate proto/thrift code

// ValidateIncludePaths checks the -I paths a caller passed. The flag takes the
// directory that holds the .proto or .thrift files; passing a file instead made
// the generator build a path like service.proto/*.proto, and protoc then
// complained about a file nobody had mentioned.
func ValidateIncludePaths(paths []string) error {
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("turbo: -I %s cannot be read: %w", path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("turbo: -I %s is a file, but -I takes the directory that contains your "+
				".proto or .thrift files; pass %s instead", path, filepath.Dir(path))
		}
	}
	return nil
}

// legacyProtocGenGo reports whether the installed protoc-gen-go is the old one,
// which certainly supports --go_out=plugins=grpc, the option turbo generates
// with. A modern binary may or may not: v1.26.0 and v1.31.0 both generate with
// it, so unlike the first version of this check the answer is only used to decide
// whether to warn.
//
// A legacy plugin answers --version with "this program should be run by protoc";
// a modern one prints "protoc-gen-go v1.x.y".
func legacyProtocGenGo(versionOutput string) bool {
	return !strings.Contains(versionOutput, "protoc-gen-go v")
}

// modernProtocGenGoRefusal is what a protoc-gen-go that dropped plugins=grpc
// answers turbo's --go_out=plugins=grpc with. The wording is identical from
// v1.31.0 on, so it is the signal a failure is recognised by.
const modernProtocGenGoRefusal = "plugins are not supported"

// legacyProtocGenGoFix is the way out when protoc refuses that option. Which
// release still accepts plugins=grpc cannot be told from the version string --
// v1.26.0 accepts it, v1.31.0 refuses it -- so the fix is spelled out wherever
// the failure can be seen rather than guessed at from a version number.
const legacyProtocGenGoFix = "install a plugin that accepts it and make sure protoc finds " +
	"that one first in PATH:\n" +
	"\tgo install github.com/golang/protobuf/protoc-gen-go@v1.5.1\n" +
	"\texport PATH=\"$GOPATH/bin:$PATH\"\n" +
	"see the README, \"Code generation needs the legacy protoc-gen-go first in PATH\""

// checkToolchain reports what generation needs before it writes anything, so a
// missing or incompatible tool produces one clear sentence instead of a wall of
// output from protoc -- and, together with the atomic writes, leaves every
// existing artifact untouched.
func (g *Generator) checkToolchain() {
	if err := ValidateIncludePaths(g.FilePaths); err != nil {
		panic(err)
	}
	tools := []string{"protoc", "protoc-gen-go", "protoc-gen-buildfields"}
	if g.RpcType == "thrift" {
		tools = []string{"thrift"}
	}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			panic(fmt.Errorf("turbo: %s is not in PATH, install it before generating (%w)", tool, err))
		}
	}
	if g.RpcType != "grpc" {
		return
	}

	version, err := exec.Command("protoc", "--version").CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("turbo: cannot run protoc --version: %w", err))
	}
	log.Infof("turbo: generating with %s", strings.TrimSpace(string(version)))

	pluginVersion, _ := exec.Command("protoc-gen-go", "--version").CombinedOutput()
	if !legacyProtocGenGo(string(pluginVersion)) {
		// Whether this release still accepts plugins=grpc cannot be told from the
		// version string: protoc-gen-go v1.26.0 and v1.31.0 both generate with it,
		// while the option is on its way out. Refusing here refused toolchains that
		// work, so say what to do if protoc objects instead of standing in the way.
		log.Warnf("turbo: protoc-gen-go reports %s. turbo generates with --go_out=plugins=grpc, "+
			"and releases differ on whether they still accept it (v1.26.0 does, v1.31.0 answers "+
			"'%s'). If protoc fails on that option, %s",
			strings.TrimSpace(string(pluginVersion)), modernProtocGenGoRefusal, legacyProtocGenGoFix)
	}
}

func (g *Generator) Generate() {
	if g.RpcType != "grpc" && g.RpcType != "thrift" {
		panic("Invalid server type, should be (grpc|thrift)")
	}
	g.c = NewConfig(g.RpcType, findFileIn(g.FilePaths, g.ConfigFileName+".yaml"))
	// generating runs without a server, and the package logger is only set up
	// when one is created -- so without this every log call on this path is a
	// nil dereference. NewConfig above only reads, so anything checkToolchain
	// refuses is still refused before a file is written.
	initLogger(g.c)
	g.checkToolchain()
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

func findFileIn(filePaths []string, file string) string {
	pathsStr := ""
	for _, p := range filePaths {
		fullPath := p + "/" + file
		_, err := os.Stat(fullPath)
		if err == nil {
			return fullPath
		}
		pathsStr += p + "\n"
	}
	panic("can not find " + file + " in any:\n" + pathsStr)
}

// writeFileWithTemplate writes a generated file atomically: the content goes to a
// temporary file beside the target and is renamed over it only once it is
// complete. os.Create used to truncate the target first, so a template that
// failed halfway -- or a generator that died -- left a half written file where a
// working one used to be, and nothing said so.
func writeFileWithTemplate(filePath string, data interface{}, text string) {
	tmpl, err := template.New("").Parse(text)
	panicIf(err)

	dir := filepath.Dir(filePath)
	f, err := os.CreateTemp(dir, filepath.Base(filePath)+".tmp")
	panicIf(err)
	temporary := f.Name()

	if err := tmpl.Execute(f, data); err != nil {
		f.Close()
		os.Remove(temporary)
		panic(err)
	}
	if err := f.Close(); err != nil {
		os.Remove(temporary)
		panic(err)
	}
	if err := os.Chmod(temporary, 0644); err != nil {
		os.Remove(temporary)
		panic(err)
	}
	if err := os.Rename(temporary, filePath); err != nil {
		os.Remove(temporary)
		panic(err)
	}
}

// GenerateGrpcSwitcher generates "grpcswither.go"
func (g *Generator) GenerateGrpcSwitcher() {
	if _, err := os.Stat(g.c.ServiceRootPath() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(g.c.ServiceRootPath()+"/gen", 0755)
	}
	serviceMethodMap := methodNames(g.c.mappings[urlServiceMaps])
	structFields := make(map[string][]string, len(serviceMethodMap))
	for s, methods := range serviceMethodMap {
		fields := make([]string, len(methods))
		for i, m := range methods {
			fields[i] = g.structFields(m + "Request")
		}
		structFields[s] = fields
	}
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/gen/grpcswitcher.go",
		struct {
			ServiceMethodMap map[string][]string
			PkgPath          string
			ServiceName      []string
			StructFields     map[string][]string
		}{
			serviceMethodMap,
			g.PkgPath,
			g.c.GrpcServiceNames(),
			structFields,
		},
		`// Code generated by turbo. DO NOT EDIT.
package gen

import (
	g "{{.PkgPath}}/gen/proto"
	"github.com/vaporz/turbo"
	"net/http"
	"errors"
)

// GrpcSwitcher is a runtime func with which a server starts.
var GrpcSwitcher = func(s turbo.Servable, serviceName, methodName string, resp http.ResponseWriter, req *http.Request) (rpcResponse interface{}, err error) {
	callOptions, header, trailer, peer := turbo.CallOptions(serviceName, methodName, req){{range $Service, $Methods := .ServiceMethodMap}}
	if serviceName == "{{$Service}}" {
		switch methodName { {{range $i, $MethodName := $Methods}}
		case "{{$MethodName}}":
			request := &g.{{$MethodName}}Request{ {{index $.StructFields $Service $i}} }
			err = turbo.BuildRequest(s, request, req)
			if err != nil {
				return nil, err
			}
			rpcResponse, err = s.Service("{{$Service}}").(g.{{$Service}}Client).{{$MethodName}}(req.Context(), request, callOptions...){{end}}
		default:
			return nil, errors.New("No such method[" + methodName + "]")
		}
	}{{end}}
	if rpcResponse==nil && err==nil {
		return nil, errors.New("No such service[" + serviceName + "]")
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
	if _, err := os.Stat(g.c.ServiceRootPath() + "/gen/proto"); os.IsNotExist(err) {
		os.MkdirAll(g.c.ServiceRootPath()+"/gen/proto", 0755)
	}
	cmd := "protoc " + g.Options + " --go_out=plugins=grpc:" + g.c.ServiceRootPath() + "/gen/proto" +
		" --buildfields_out=service_root_path=" + g.c.ServiceRootPath() + ":" + g.c.ServiceRootPath() + "/gen/proto"
	executeCmd("bash", "-c", cmd)
}

// GenerateBuildThriftParameters generates "build.go"
func (g *Generator) GenerateBuildThriftParameters() {
	writeFileWithTemplate(
		g.c.ServiceRootPath()+"/gen/thrift/build.go",
		struct {
			PkgPath         string
			ServiceNames    []string
			ServiceRootPath string
			// todo
			ServiceMethodMap map[string][]string
		}{
			g.PkgPath,
			g.c.GrpcServiceNames(),
			g.c.ServiceRootPath(),
			methodNames(g.c.mappings[urlServiceMaps])},
		buildThriftParameters,
	)
	g.runBuildThriftFields()
}

func (g *Generator) runBuildThriftFields() {
	executeCmd("bash", "-c", "go run "+g.c.ServiceRootPath()+"/gen/thrift/build.go")
}

var buildThriftParameters = `package main

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

var serviceMethodName = flag.String("n", "", "")

func main() {
	flag.Parse()
	if len(strings.TrimSpace(*serviceMethodName)) > 0 {
		names := strings.Split(*serviceMethodName, ",")
		str := buildParameterStr(names[0], names[1])
		fmt.Print(str)
	} else {
		buildFields()
	}
}

func buildFields() {
	services := []interface{}{ {{range $i, $ServiceName := .ServiceNames}}
		new(g.{{$ServiceName}}),{{end}}
	}
	var list string
	for _, i := range services {
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
		for _, s := range items {
			list += s + "\n"
		}
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

func buildParameterStr(serviceName, methodName string) string { {{range $ServiceName, $Methods := .ServiceMethodMap}}
	if serviceName == "{{- $ServiceName -}}" {
		switch methodName { {{range $i, $MethodName := $Methods}}
		case "{{$MethodName}}":
			var result string
			args := g.{{- $ServiceName -}}{{$MethodName}}Args{}
			at := reflect.TypeOf(args)
			num := at.NumField()
			for i := 0; i < num; i++ {
				result += fmt.Sprintf(
					"\n\t\t\t\tparams[%d].Interface().(%s),",
					i, at.Field(i).Type.String())
			}
			return result{{end}}
		default:
			return "error"
		}
	}{{end}}
	return "error"
}
`

// GenerateThriftSwitcher generates "thriftswitcher.go"
func (g *Generator) GenerateThriftSwitcher() {
	if _, err := os.Stat(g.c.ServiceRootPath() + "/gen"); os.IsNotExist(err) {
		os.Mkdir(g.c.ServiceRootPath()+"/gen", 0755)
	}
	serviceMethodMap := methodNames(g.c.mappings[urlServiceMaps])
	parameters := make(map[string][]string, len(serviceMethodMap))
	notEmptyParameters := make(map[string][]bool, len(serviceMethodMap))
	for s, methods := range serviceMethodMap {
		for _, v := range methods {
			p := g.thriftParameters(s, v)
			parameters[s] = append(parameters[s], p)
			notEmptyParameters[s] = append(notEmptyParameters[s], len(strings.TrimSpace(p)) > 0)
		}
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
		struct {
			PkgPath            string
			BuildArgsCases     string
			ServiceNames       []string
			ServiceMethodMap   map[string][]string
			Parameters         map[string][]string
			NotEmptyParameters map[string][]bool
			StructNames        []string
			StructFields       []string
		}{
			g.PkgPath,
			argCasesStr,
			g.c.ThriftServiceNames(),
			serviceMethodMap,
			parameters,
			notEmptyParameters,
			structNames,
			fields},
		thriftSwitcherFunc,
	)
}

func (g *Generator) thriftParameters(serviceName, methodName string) string {
	cmd := "go run " + g.c.ServiceRootPath() + "/gen/thrift/build.go -n " + serviceName + "," + methodName
	buf := &bytes.Buffer{}
	c := exec.Command("bash", "-c", cmd)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = buf
	// this output is parsed, so it cannot be streamed as it is produced; a failure
	// carries it in the panic instead, where it explains what went wrong
	if err := c.Run(); err != nil {
		panic(commandError(cmd, err, buf.String()))
	}
	return buf.String() + " "
}

func methodNames(urlServiceMaps [][4]string) map[string][]string {
	methodNamesMap := make(map[string]map[string]int)
	for _, v := range urlServiceMaps {
		if methodNamesMap[v[2]] == nil {
			methodNamesMap[v[2]] = make(map[string]int)
		}
		methodNamesMap[v[2]][v[3]] = 0 //map [ServiceName] [MethodName] = 0
	}
	methodNames := make(map[string][]string)
	for k, v := range methodNamesMap {
		methods := make([]string, 0, len(v))
		for m := range v {
			methods = append(methods, m)
		}
		// sorted, so that regenerating a service produces the same file: the map
		// above has no order, and an unsorted list reorders the generated switch
		// cases differently on every run
		sort.Strings(methods)
		methodNames[k] = methods
	}
	return methodNames
}

var thriftSwitcherFunc = `// Code generated by turbo. DO NOT EDIT.
package gen

import (
	"context"
	"errors"
	"github.com/vaporz/turbo"
	"{{.PkgPath}}/gen/thrift/gen-go/gen"
	"net/http"
	"reflect"
)

// ThriftSwitcher is a runtime func with which a server starts.
var ThriftSwitcher = func(s turbo.Servable, serviceName, methodName string, resp http.ResponseWriter, req *http.Request) (serviceResponse interface{}, err error) { {{range $Service, $Methods := .ServiceMethodMap}}
	if serviceName == "{{$Service}}" {
		ctx := context.Background()
		switch methodName { {{range $i, $MethodName := $Methods}}
		case "{{$MethodName}}":{{if index $.NotEmptyParameters $Service $i }}
			params, err := turbo.BuildThriftRequest(s, gen.{{$Service}}{{$MethodName}}Args{}, req, buildStructArg)
			if err != nil {
				return nil, err
			}{{end}}
			return s.Service("{{$Service}}").(*gen.{{$Service}}Client).{{$MethodName}}(
				ctx,{{index $.Parameters $Service $i}}){{end}}
		default:
			return nil, errors.New("No such method[" + methodName + "]")
		}
	}
	{{end}}
	if serviceResponse == nil && err == nil {
		return nil, errors.New("No such service[" + serviceName + "]")
	}
	return
}

func buildStructArg(s turbo.Servable, typeName string, req *http.Request) (v reflect.Value, err error) {
	switch typeName {
{{range $i, $StructName := .StructNames}}
	case "{{$StructName}}":
		request := &gen.{{$StructName}}{ {{index $.StructFields $i}} }
		if err := turbo.BuildStructErr(s, reflect.TypeOf(request).Elem(), reflect.ValueOf(request).Elem(), req); err != nil {
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
	nameLower := strings.ToLower(g.c.ThriftServiceNames()[0]) // todo change a thrift file name
	cmd := "thrift " + g.Options + " -r --gen go:package_prefix=" + g.PkgPath + "/gen/thrift/gen-go/ -o" +
		" " + g.c.ServiceRootPath() + "/" + "gen/thrift " + g.c.ServiceRootPath() + "/" + nameLower + ".thrift"
	executeCmd("bash", "-c", cmd)
}

// outputTailBytes bounds how much of a failing command's output is quoted back to
// the reader: enough to carry the tool's own explanation, small enough to read.
const outputTailBytes = 2048

// outputTail remembers the end of what a command printed while passing it on, so
// the output is still visible as it is produced. Only the end is kept: a failing
// protoc or thrift can print a lot, and the reason is the last thing it says.
type outputTail struct {
	mu   sync.Mutex
	data []byte
}

func (t *outputTail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data = append(t.data, p...)
	if len(t.data) > outputTailBytes {
		t.data = t.data[len(t.data)-outputTailBytes:]
	}
	return len(p), nil
}

func (t *outputTail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.data)
}

// commandError says what turbo ran and what came back. A tool that fails on its
// own terms explains itself, and that explanation is the only actionable part:
// reporting a bare "exit status 1" leaves the reader hunting for it above.
//
// One of those explanations has a fix the reader is unlikely to guess -- protoc
// refusing plugins=grpc -- so it is added to the message rather than left in the
// output, which is where it went missing the three times this project hit it.
func commandError(cmd string, err error, output string) error {
	tail := strings.TrimSpace(output)
	hint := ""
	if strings.Contains(tail, modernProtocGenGoRefusal) {
		hint = "\nturbo: " + legacyProtocGenGoFix
	}
	if tail != "" {
		return fmt.Errorf("turbo: %s failed: %w\nturbo: last lines it printed:\n%s%s", cmd, err, tail, hint)
	}
	return fmt.Errorf("turbo: %s failed: %w%s", cmd, err, hint)
}

// executeCmd runs a generation tool, streaming its output, and panics with the
// command and the tail of its output when it fails (see commandError). Generation
// depends on external tools -- protoc, its plugins, the thrift compiler -- whose
// messages are worth more than the exit status turbo used to be left with.
func executeCmd(cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	tail := &outputTail{}
	c.Stdin = os.Stdin
	c.Stderr = io.MultiWriter(os.Stderr, tail)
	c.Stdout = io.MultiWriter(os.Stdout, tail)
	if err := c.Run(); err != nil {
		panic(commandError(cmd+" "+strings.Join(args, " "), err, tail.String()))
	}
}
