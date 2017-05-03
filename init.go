package turbo

import (
	"os"
	"text/template"
	"strings"
)

func Init(pkgPath, serviceName string) {
	initPkgPath(pkgPath)
	initServiceName(serviceName)
	createFolders()
	createFiles()
}

func createFolders() {
	os.Mkdir(servicePkgPath, 0755)
	os.Mkdir(servicePkgPath+"/gen", 0755)
	os.Mkdir(servicePkgPath+"/service", 0755)
	os.Mkdir(servicePkgPath+"/service/impl", 0755)
}

func createFiles() {
	createServiceYaml()
	createProto()
}

func createServiceYaml() {
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
  service_name: {{.ServiceName}}
  service_address: 127.0.0.1:50051

urlmapping:
  - GET /hello SayHello
`

func createProto() {
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
