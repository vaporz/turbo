/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

func main() {
	request := new(plugin_go.CodeGeneratorRequest)
	response := new(plugin_go.CodeGeneratorResponse)
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println("reading input error:", err)
	}
	if err = proto.Unmarshal(data, request); err != nil {
		fmt.Println("parsing input proto:", err)
	}
	generateBuildFields(request, response)
}

func generateBuildFields(req *plugin_go.CodeGeneratorRequest, resp *plugin_go.CodeGeneratorResponse) {
	files := req.ProtoFile
	items := make([]string, 0)
	for _, f := range files {
		messages := f.MessageType
		for _, m := range messages {
			argStr := *m.Name
			if strings.HasSuffix(argStr, "Request") {
				arr := strings.Split(argStr, ".")
				name := arr[len(arr)-1:][0]
				items = findItem(items, name, *m)
			}
		}
	}
	m := parameterMap(*req.Parameter)
	var list string
	for _, s := range items {
		list += s + "\n"
	}
	writeFileWithTemplate(
		m["service_root_path"]+"/gen/grpcfields.yaml",
		fieldsYaml,
		fieldsYamlValues{List: list},
	)
}

func parameterMap(parameter string) map[string]string {
	m := make(map[string]string, 1)
	items := strings.Split(parameter, ",")
	for _, item := range items {
		pair := strings.SplitN(item, "=", 2)
		m[pair[0]] = pair[1]
	}
	return m
}

func findItem(items []string, name string, structType descriptor.DescriptorProto) []string {
	numField := len(structType.Field)
	item := "  - " + name + "["
	for i := 0; i < numField; i++ {
		fieldType := structType.Field[i]
		if *fieldType.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			arr := strings.Split(*fieldType.TypeName, ".")
			typeName := arr[len(arr)-1:][0]
			argName := *fieldType.Name
			item += fmt.Sprintf("%v %s,", typeName, argName)
		}
	}
	item += "]"
	return append(items, item)
}

type fieldsYamlValues struct {
	List string
}

var fieldsYaml string = `grpc-fieldmapping:
{{.List}}
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
