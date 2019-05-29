/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// TODO support plugin for e.g. customize folder structure

// Creator creates new projects
type Creator struct {
	RpcType string
	PkgPath string
	c       *Config
}

// CreateProject creates a whole new project!
func (c *Creator) CreateProject(serviceName string, force bool) {
	if !force {
		c.validateServiceRootPath(nil)
	}
	c.createRootFolder(GOPATH() + "/src/" + c.PkgPath)
	c.createServiceYaml(GOPATH()+"/src/"+c.PkgPath, serviceName, "service")
	c.c = NewConfig(c.RpcType, GOPATH()+"/src/"+c.PkgPath+"/service.yaml")
	if c.RpcType == "grpc" {
		c.createGrpcProject(serviceName)
	} else if c.RpcType == "thrift" {
		c.createThriftProject(serviceName)
	}
}

func (c *Creator) validateServiceRootPath(in io.Reader) {
	if in == nil {
		in = os.Stdin
	}
	if len(strings.TrimSpace(c.PkgPath)) == 0 {
		panic("pkgPath is blank")
	}
	p := GOPATH() + "/src/" + c.PkgPath
	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		return
	}
	fmt.Print("Path '" + p + "' already exist!\n" +
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
	os.RemoveAll(p)
}

func (c *Creator) createGrpcProject(serviceName string) {
	c.createGrpcFolders()
	c.createProto(serviceName)
	c.generateGrpcServiceMain()
	c.generateGrpcServiceImpl()
	c.generateGrpcHTTPMain()
	c.generateGrpcHTTPComponent()
	c.generateServiceMain("grpc")

	g := Generator{
		RpcType:        c.RpcType,
		PkgPath:        c.PkgPath,
		ConfigFileName: "service",
	}
	g.c = NewConfig(g.RpcType, c.c.ServiceRootPathAbsolute()+"/"+g.ConfigFileName+".yaml")
	g.Options = " -I " + c.c.ServiceRootPathAbsolute() + " " + c.c.ServiceRootPathAbsolute() + "/" + strings.ToLower(serviceName) + ".proto "
	g.GenerateProtobufStub()
	g.c.loadFieldMapping()
	g.GenerateGrpcSwitcher()
}

func (c *Creator) createThriftProject(serviceName string) {
	c.createThriftFolders()
	c.createThrift(serviceName)
	c.generateThriftServiceMain()
	c.generateThriftServiceImpl()
	c.generateThriftHTTPMain()
	c.generateThriftHTTPComponent()
	c.generateServiceMain("thrift")

	g := Generator{
		RpcType:        c.RpcType,
		PkgPath:        c.PkgPath,
		ConfigFileName: "service",
	}
	g.c = NewConfig(g.RpcType, c.c.ServiceRootPathAbsolute()+"/"+g.ConfigFileName+".yaml")
	g.Options = " -I " + c.c.ServiceRootPathAbsolute() + " "
	g.GenerateThriftStub()
	g.GenerateBuildThriftParameters()
	g.c.loadFieldMapping()
	g.GenerateThriftSwitcher()

}

func (c *Creator) createRootFolder(serviceRootPath string) {
	os.MkdirAll(serviceRootPath+"/gen", 0755)
}

func (c *Creator) createGrpcFolders() {
	os.MkdirAll(c.c.ServiceRootPathAbsolute()+"/gen/proto", 0755)
	os.MkdirAll(c.c.ServiceRootPathAbsolute()+"/grpcapi/component", 0755)
	os.MkdirAll(c.c.ServiceRootPathAbsolute()+"/grpcservice/impl", 0755)
}

func (c *Creator) createThriftFolders() {
	os.MkdirAll(c.c.ServiceRootPathAbsolute()+"/gen/thrift", 0755)
	os.MkdirAll(c.c.ServiceRootPathAbsolute()+"/thriftapi/component", 0755)
	os.MkdirAll(c.c.ServiceRootPathAbsolute()+"/thriftservice/impl", 0755)
}

func (c *Creator) createServiceYaml(serviceRootPath, serviceName, configFileName string) {
	if _, err := os.Stat(serviceRootPath + "/" + configFileName + ".yaml"); err == nil {
		return
	}
	writeFileWithTemplate(
		serviceRootPath+"/"+configFileName+".yaml",
		struct {
			ServiceRoot string
			ServiceName string
		}{serviceRootPath, serviceName},
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
  - GET /hello {{.ServiceName}} SayHello
`)
}

func (c *Creator) createProto(serviceName string) {
	nameLower := strings.ToLower(serviceName)
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/"+nameLower+".proto",
		struct {
			ServiceName string
		}{serviceName},
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

func (c *Creator) createThrift(serviceName string) {
	nameLower := strings.ToLower(serviceName)
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/"+nameLower+".thrift",
		struct {
			ServiceName string
		}{serviceName},
		`namespace go services

struct SayHelloResponse {
  1: string message,
}

service {{.ServiceName}} {
    SayHelloResponse sayHello (1:string yourName)
}
`,
	)
}

func (c *Creator) generateGrpcServiceMain() {
	nameLower := strings.ToLower(c.c.GrpcServiceNames()[0])
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/grpcservice/"+nameLower+".go",
		struct {
			PkgPath        string
			ConfigFilePath string
		}{c.PkgPath, c.c.ServiceRootPathAbsolute() + "/service.yaml"},
		`package main

import (
	"{{.PkgPath}}/grpcservice/impl"
	"github.com/vaporz/turbo"
	"os/signal"
	"os"
	"syscall"
	"fmt"
)

func main() {
	s := turbo.NewGrpcServer(nil, "{{.ConfigFilePath}}")
	s.StartGrpcService(impl.RegisterServer)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
`,
	)
}

func (c *Creator) generateThriftServiceMain() {
	nameLower := strings.ToLower(c.c.ThriftServiceNames()[0])
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/thriftservice/"+nameLower+".go",
		struct {
			PkgPath        string
			ConfigFilePath string
		}{c.PkgPath, c.c.ServiceRootPathAbsolute() + "/service.yaml"},
		`package main

import (
	"{{.PkgPath}}/thriftservice/impl"
	"github.com/vaporz/turbo"
	"os/signal"
	"os"
	"syscall"
	"fmt"
)

func main() {
	s := turbo.NewThriftServer(nil, "{{.ConfigFilePath}}")
	s.StartThriftService(impl.TProcessor)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
`,
	)
}

func (c *Creator) generateGrpcServiceImpl() {
	nameLower := strings.ToLower(c.c.GrpcServiceNames()[0])
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/grpcservice/impl/"+nameLower+"impl.go",
		struct {
			PkgPath     string
			ServiceName string
		}{c.PkgPath, c.c.GrpcServiceNames()[0]},
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

func (c *Creator) generateThriftServiceImpl() {
	nameLower := strings.ToLower(c.c.ThriftServiceNames()[0])
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/thriftservice/impl/"+nameLower+"impl.go",
		struct {
			PkgPath     string
			ServiceName string
		}{c.PkgPath, c.c.ThriftServiceNames()[0]},
		`package impl

import (
	"{{.PkgPath}}/gen/thrift/gen-go/services"
	"git.apache.org/thrift.git/lib/go/thrift"
)

// TProcessor returns TProcessor
func TProcessor() map[string]thrift.TProcessor {
	return map[string]thrift.TProcessor{
		"{{.ServiceName}}": services.New{{.ServiceName}}Processor({{.ServiceName}}{}),
	}
}

// {{.ServiceName}} is the struct which implements generated interface
type {{.ServiceName}} struct {
}

// SayHello is an example entry point
func (s {{.ServiceName}}) SayHello(yourName string) (r *services.SayHelloResponse, err error) {
	return &services.SayHelloResponse{Message: "[thrift server]Hello, " + yourName}, nil
}
`,
	)
}

func (c *Creator) generateGrpcHTTPMain() {
	nameLower := strings.ToLower(c.c.GrpcServiceNames()[0])
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/grpcapi/"+nameLower+"api.go",
		struct {
			PkgPath        string
			ConfigFilePath string
		}{
			c.PkgPath,
			c.c.ServiceRootPathAbsolute() + "/service.yaml"},
		`package main

import (
	"{{.PkgPath}}/gen"
	"{{.PkgPath}}/grpcapi/component"
	"github.com/vaporz/turbo"
	"os/signal"
	"os"
	"syscall"
	"fmt"
)

func main() {
	s := turbo.NewGrpcServer(&component.ServiceInitializer{}, "{{.ConfigFilePath}}")
	s.StartHTTPServer(component.GrpcClient, gen.GrpcSwitcher)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
`,
	)
}

// todo reuse created client instance
func (c *Creator) generateGrpcHTTPComponent() {
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/grpcapi/component/components.go",
		struct {
			ServiceNames []string
			PkgPath      string
		}{c.c.GrpcServiceNames(), c.PkgPath},
		`package component

import (
	"{{.PkgPath}}/gen/proto"
	"google.golang.org/grpc"
	"github.com/vaporz/turbo"
)

// GrpcClient returns a grpc client
func GrpcClient(conn *grpc.ClientConn) map[string]interface{} {
	return map[string]interface{}{
{{range $i, $ServiceName := .ServiceNames}}
		"{{- $ServiceName -}}": proto.New{{- $ServiceName -}}Client(conn),
{{end}}
	}
}

type ServiceInitializer struct {
}

// InitService is run before the service is started, do initializing staffs for your service here.
// For example, init turbo components, such as interceptors, pre/postprocessors, errorHandlers, etc.
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	// TODO
	return nil
}

// StopService is run after both grpc server and http server are stopped,
// do your cleaning up work here.
func (i *ServiceInitializer) StopService(s turbo.Servable) {
	// TODO
}
`,
	)
}

func (c *Creator) generateThriftHTTPComponent() {
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/thriftapi/component/components.go",
		struct {
			ServiceName string
			PkgPath      string
		}{c.c.GrpcServiceNames()[0], c.PkgPath},
		`package component

import (
	t "{{.PkgPath}}/gen/thrift/gen-go/services"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/vaporz/turbo"
)

// ThriftClient returns a thrift client
func ThriftClient(trans thrift.TTransport, f thrift.TProtocolFactory) map[string]interface{} {
	iprot := f.GetProtocol(trans)
	return map[string]interface{}{
		"{{- .ServiceName -}}": t.New{{- .ServiceName -}}ClientProtocol(trans, iprot, thrift.NewTMultiplexedProtocol(iprot, "{{- .ServiceName -}}")),
	}
}

type ServiceInitializer struct {
}

// InitService is run before the service is started, do initializing staffs for your service here.
// For example, init turbo components, such as interceptors, pre/postprocessors, errorHandlers, etc.
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	// TODO
	return nil
}

// StopService is run after both grpc server and http server are stopped,
// do your cleaning up work here.
func (i *ServiceInitializer) StopService(s turbo.Servable) {
	// TODO
}
`,
	)
}

func (c *Creator) generateThriftHTTPMain() {
	nameLower := strings.ToLower(c.c.ThriftServiceNames()[0])
	writeFileWithTemplate(
		c.c.ServiceRootPathAbsolute()+"/thriftapi/"+nameLower+"api.go",
		struct {
			ServiceName    string
			PkgPath        string
			ConfigFilePath string
		}{
			c.c.ThriftServiceNames()[0],
			c.PkgPath,
			c.c.ServiceRootPathAbsolute() + "/service.yaml"},
		`package main

import (
	"github.com/vaporz/turbo"
	"{{.PkgPath}}/gen"
	"{{.PkgPath}}/thriftapi/component"
	"os/signal"
	"os"
	"syscall"
	"fmt"
)

func main() {
	s := turbo.NewThriftServer(&component.ServiceInitializer{}, "{{.ConfigFilePath}}")
	s.StartHTTPServer(component.ThriftClient, gen.ThriftSwitcher)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
`,
	)
}

func (c *Creator) generateServiceMain(rpcType string) {
	type rootMainValues struct {
		PkgPath        string
		ConfigFilePath string
	}
	if rpcType == "grpc" {
		writeFileWithTemplate(
			c.c.ServiceRootPathAbsolute()+"/main.go",
			rootMainValues{PkgPath: c.PkgPath, ConfigFilePath: c.c.ServiceRootPathAbsolute() + "/service.yaml"},
			rootMainGrpc,
		)
	} else if rpcType == "thrift" {
		writeFileWithTemplate(
			c.c.ServiceRootPathAbsolute()+"/main.go",
			rootMainValues{PkgPath: c.PkgPath, ConfigFilePath: c.c.ServiceRootPathAbsolute() + "/service.yaml"},
			rootMainThrift,
		)
	}
}

var rootMainGrpc = `package main

import (
	"github.com/vaporz/turbo"
	"{{.PkgPath}}/gen"
	gcomponent "{{.PkgPath}}/grpcapi/component"
	gimpl "{{.PkgPath}}/grpcservice/impl"
	//tcomponent "{{.PkgPath}}/thriftapi/component"
	//timpl "{{.PkgPath}}/thriftservice/impl"
	"os/signal"
	"os"
	"syscall"
	"fmt"
)

func main() {
	s := turbo.NewGrpcServer(&gcomponent.ServiceInitializer{}, "{{.ConfigFilePath}}")
	s.Start(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	//s := turbo.NewThriftServer(&tcomponent.ServiceInitializer{}, "{{.ConfigFilePath}}")
	//s.Start(tcomponent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
`

var rootMainThrift = `package main

import (
	"github.com/vaporz/turbo"
	"{{.PkgPath}}/gen"
	//gcomponent "{{.PkgPath}}/grpcapi/component"
	//gimpl "{{.PkgPath}}/grpcservice/impl"
	tcomponent "{{.PkgPath}}/thriftapi/component"
	timpl "{{.PkgPath}}/thriftservice/impl"
	"os/signal"
	"os"
	"syscall"
	"fmt"
)

func main() {
	//s := turbo.NewGrpcServer(&gcomponent.ServiceInitializer{}, "{{.ConfigFilePath}}")
	//s.Start(gcomponent.GrpcClient, gen.GrpcSwitcher, gimpl.RegisterServer)

	s := turbo.NewThriftServer(&tcomponent.ServiceInitializer{}, "{{.ConfigFilePath}}")
	s.Start(tcomponent.ThriftClient, gen.ThriftSwitcher, timpl.TProcessor)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Service is stopping...")
	}

	s.Stop()
	fmt.Println("Service stopped")
}
`
