package main

import (
	"text/template"
	"bytes"
	"os"
	"bufio"
	"strings"
	"io"
	"log"
)

func main() {
	loadServiceConfig()
	var casesStr string
	for _, v := range methodNames {
		tmpl, err := template.New("cases").Parse(cases)
		if err != nil {
			panic(err)
		}
		var casesBuf bytes.Buffer
		err = tmpl.Execute(&casesBuf, method{v})
		casesStr = casesStr + casesBuf.String()
	}
	tmpl, err := template.New("handler").Parse(handler)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create("/Users/xiaozhang/goworkspace/src/zx/demo/generator/gen/handler.go")
	err = tmpl.Execute(f, casesContent{casesStr})
	if err != nil {
		panic(err)
	}

}

var methodNames []string

func loadServiceConfig() {
	//currentDir, err := filepath.Abs(filepath.Dir("."))
	//if err != nil {
	//	log.Fatal("load config fail")
	//}
	//log.Println(currentDir)
	//TODO get filepath
	f, err := os.Open("/Users/xiaozhang/goworkspace/src/zx/demo/inventoryservice/http/service.config")
	if err != nil {
		log.Println(err)
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		appendUrlServiceMap(line)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Println(err)
			break
		}
	}
}

func appendUrlServiceMap(line string) {
	pair := strings.Split(line, "=")
	methodName := strings.TrimSpace(pair[1])
	methodNames = append(methodNames, methodName)
}

type method struct {
	MethodName string
}

type casesContent struct {
	Cases string
}

var handler string = `package gen

import (
	"reflect"
	"net/http"
	cm "zx/demo/common"
	pb "zx/demo/proto/inventoryservice"
	client "zx/demo/inventoryservice/http/component/clients"
	"fmt"
)

var Handler = func(methodName string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		switch methodName {
		{{.Cases}}
		default:
			resp.Write([]byte(fmt.Sprintf("No such grpc method[%s]", methodName)))
		}
	}
}`

var cases string = `case "{{.MethodName}}":
			cm.ParseRequestForm(req)
			request := pb.{{.MethodName}}Request{}
			theType := reflect.TypeOf(request)
			theValue := reflect.ValueOf(&request).Elem()
			fieldNum := theType.NumField()
			for i := 0; i < fieldNum; i++ {
				fieldName := theType.Field(i).Name
				v, ok := req.Form[cm.ToSnakeCase(fieldName)]
				if ok && len(v) > 0 {
					theValue.FieldByName(fieldName).SetString(v[0])
				}
			}
			params := make([]reflect.Value, 2)
			params[0] = reflect.ValueOf(req.Context())
			params[1] = reflect.ValueOf(&request)
			result := reflect.ValueOf(client.InventoryService()).MethodByName(methodName).Call(params)

			rsp := result[0].Interface().(*pb.{{.MethodName}}Response)
			if result[1].Interface() == nil {
				resp.Write([]byte(rsp.String() + "\n"))
			} else {
				resp.Write([]byte(result[1].Interface().(error).Error() + "\n"))
			}
			`
