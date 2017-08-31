package main

import (
	"flag"
	"fmt"
	g "github.com/vaporz/turbo/test/testservice/gen/thrift/gen-go/gen"
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
	i := new(g.TestService)
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

var fieldsYaml string = `thrift-fieldmapping:
{{.List}}
`

func buildParameterStr(methodName string) string {
	switch methodName {

	case "TestJson":
		var result string
		args := g.TestServiceTestJsonArgs{}
		at := reflect.TypeOf(args)
		num := at.NumField()
		for i := 0; i < num; i++ {
			result += fmt.Sprintf(
				"\n\t\t\tparams[%d].Interface().(%s),",
				i, at.Field(i).Type.String())
		}
		return result

	case "SayHello":
		var result string
		args := g.TestServiceSayHelloArgs{}
		at := reflect.TypeOf(args)
		num := at.NumField()
		for i := 0; i < num; i++ {
			result += fmt.Sprintf(
				"\n\t\t\tparams[%d].Interface().(%s),",
				i, at.Field(i).Type.String())
		}
		return result

	default:
		return "error"
	}
}
