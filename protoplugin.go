package turbo

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
)

func main() {
	request := new(plugin_go.CodeGeneratorRequest)
	response := new(plugin_go.CodeGeneratorResponse)
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Println("reading input error:", err)
	}
	if err = proto.Unmarshal(data, request); err != nil {
		log.Println("parsing input proto:", err)
	}
	generateValidator(request, response)
	reflect.String.String()
}

func generateValidator(req *plugin_go.CodeGeneratorRequest, resp *plugin_go.CodeGeneratorResponse) {
	// 得到文件列表
	files := req.ProtoFile
	// 对每个文件循环
	items := make([]string, 0)
	for _, f := range files {
		// 得到message列表
		messages := f.MessageType
		// 对每个message循环
		for _, m := range messages {
			argStr := *m.Name
			if strings.HasSuffix(argStr, "Request") {
				arr := strings.Split(argStr, ".")
				name := arr[len(arr)-1:][0]
				items = findItem(items, name, *m)
			}
		}

	}
	var list string
	for _, s := range items {
		list += s + "\n"
	}
	writeFileWithTemplate(
		"/Users/xiaozhang/goworkspace/src/github.com/vaporz/turbo-example/yourservice/gen/grpcfields.yaml",
		fieldsYaml,
		fieldsYamlValues{List: list},
	)
}

func findItem(items []string, name string, structType descriptor.DescriptorProto) []string {
	numField := len(structType.Field)
	item := "  - " + name + "["
	for i := 0; i < numField; i++ {
		fieldType := structType.Field[i]
		if *fieldType.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			arr := strings.Split(fieldType.Type.String(), ".")
			typeName := arr[len(arr)-1:][0]
			argName := fieldType.Name
			item += fmt.Sprintf("%s %s,", typeName, argName)
			items = findItem(items, typeName, *structType.NestedType[i])
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
