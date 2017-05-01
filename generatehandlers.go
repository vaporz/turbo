package turbo

import (
	"text/template"
	"bytes"
	"os"
)

func GenerateHandler(pkgPath string) {
	loadServiceConfig(pkgPath)
	var casesStr string
	for _, v := range UrlServiceMap {
		tmpl, err := template.New("cases").Parse(cases)
		if err != nil {
			panic(err)
		}
		var casesBuf bytes.Buffer
		err = tmpl.Execute(&casesBuf, method{configs[SERVICE_NAME], v[2]})
		casesStr = casesStr + casesBuf.String()
	}
	tmpl, err := template.New("handler").Parse(handlerFunc)
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(serviceRootPath + "/gen"); os.IsNotExist(err) {
		os.Mkdir(serviceRootPath+"/gen", 755)
	}
	f, _ := os.Create(serviceRootPath + "/gen/handler.go")
	err = tmpl.Execute(f, handlerContent{Cases: casesStr})
	if err != nil {
		panic(err)
	}
}

type method struct {
	ServiceName string
	MethodName  string
}

type handlerContent struct {
	Cases string
}

var handlerFunc string = `package gen

import (
	"reflect"
	"net/http"
	"turbo"
	"fmt"
	"log"
)

/*
this is a generated file, DO NOT EDIT!
 */
var Handler = func(methodName string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		turbo.ParseRequestForm(req)
		interceptors, ok := turbo.Interceptors(methodName)
		if !ok {
			interceptors, ok = turbo.CommonInterceptors()
		}
		if !ok {
			interceptors = turbo.EmptyInterceptors()
		}

		for _, i := range interceptors {
			err := i.Before(resp, req)
			if err != nil {
				log.Println("error in interceptor!" + err.Error())
				return
			}
		}
		skipSwitch := false
		if hijack := turbo.Hijacker(methodName); hijack != nil {
			hijack(resp, req)
			skipSwitch = true
		} else if preprocessor := turbo.Preprocessor(methodName); preprocessor != nil {
			if err := preprocessor(resp, req); err != nil {
				skipSwitch = true
			}
		}
		if !skipSwitch {
			doSwitch(methodName, resp, req)
		}
		l := len(interceptors)
		for i := l - 1; i > 0; i-- {
			err := interceptors[i].After(resp, req)
			if err != nil {
				log.Println("error in interceptor!")
				return
			}
		}
	}
}

func doSwitch(methodName string, resp http.ResponseWriter, req *http.Request) {
	switch methodName { {{.Cases}}
	default:
		resp.Write([]byte(fmt.Sprintf("No such grpc method[%s]", methodName)))
	}
}
`

var cases string = `
	case "{{.MethodName}}":
		request := {{.MethodName}}Request{}
		theType := reflect.TypeOf(request)
		theValue := reflect.ValueOf(&request).Elem()
		fieldNum := theType.NumField()
		for i := 0; i < fieldNum; i++ {
			fieldName := theType.Field(i).Name
			v, ok := req.Form[turbo.ToSnakeCase(fieldName)]
			if ok && len(v) > 0 {
				theValue.FieldByName(fieldName).SetString(v[0])
			}
		}
		params := make([]reflect.Value, 2)
		params[0] = reflect.ValueOf(req.Context())
		params[1] = reflect.ValueOf(&request)
		result := reflect.ValueOf(turbo.GrpcService().({{.ServiceName}}Client)).MethodByName(methodName).Call(params)

		rsp := result[0].Interface().(*{{.MethodName}}Response)
		if result[1].Interface() == nil {
			resp.Write([]byte(rsp.String() + "\n"))
		} else {
			resp.Write([]byte(result[1].Interface().(error).Error() + "\n"))
		}`
