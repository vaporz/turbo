package test

import (
	"io"
	"os"
	"github.com/vaporz/turbo"
	"text/template"
)

func ResetComponents() {
	turbo.ResetConvertor()
	turbo.ResetHijacker()
	turbo.ResetInterceptor()
	turbo.ResetPostprocessor()
	turbo.ResetPreprocessor()
	turbo.ResetErrorHandler()
}

func OverrideServiceYaml(httpPort, servicePort, env string) {
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
  - GET /hello/{your_name:[a-zA-Z0-9]+} SayHello
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
