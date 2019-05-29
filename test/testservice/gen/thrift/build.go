package main

import (
	"flag"
	"fmt"
	g "github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/services"
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
	services := []interface{}{ 
		new(g.TestService),
		new(g.MinionsService),
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
				if argType.Kind() == reflect.Ptr && argType.Elem().Kind() == reflect.Struct {
					items = findItem(items, argType.Elem().PkgPath()+"."+argType.Elem().Name(), argType)
				}
			}
		}
		for _, s := range items {
			list += s + "\n"
		}
	}
	writeFileWithTemplate(
		"/Users/xiaozhang/goworkspace/src/github.com/vaporz/turbo/test/testservice/gen/thriftfields.yaml",
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
			argName := fieldType.Name
			item += fmt.Sprintf("%s %s,", fieldType.Type.Elem().PkgPath()+"."+"."+fieldType.Type.Elem().Name(), argName)
			items = findItem(items, fieldType.Type.Elem().PkgPath()+"."+"."+fieldType.Type.Elem().Name(), fieldType.Type)
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

var fieldsYaml string = `thrift-fieldmapping:
{{.List}}
`

func buildParameterStr(serviceName, methodName string) string { 
	if serviceName == "MinionsService" {
		switch methodName { 
		case "Eat":
			var result string
			args := g.MinionsServiceEatArgs{}
			at := reflect.TypeOf(args)
			num := at.NumField()
			for i := 0; i < num; i++ {
				result += fmt.Sprintf(
					"\n\t\t\t\tparams[%d].Interface().(%s),",
					i, at.Field(i).Type.String())
			}
			return result
		default:
			return "error"
		}
	}
	if serviceName == "TestService" {
		switch methodName { 
		case "SayHello":
			var result string
			args := g.TestServiceSayHelloArgs{}
			at := reflect.TypeOf(args)
			num := at.NumField()
			for i := 0; i < num; i++ {
				result += fmt.Sprintf(
					"\n\t\t\t\tparams[%d].Interface().(%s),",
					i, at.Field(i).Type.String())
			}
			return result
		case "TestJson":
			var result string
			args := g.TestServiceTestJsonArgs{}
			at := reflect.TypeOf(args)
			num := at.NumField()
			for i := 0; i < num; i++ {
				result += fmt.Sprintf(
					"\n\t\t\t\tparams[%d].Interface().(%s),",
					i, at.Field(i).Type.String())
			}
			return result
		default:
			return "error"
		}
	}
	return "error"
}
