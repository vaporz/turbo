package turbo

import (
	"errors"
	// TODO support logging levels, log file path, etc.
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type switcher func(methodName string, resp http.ResponseWriter, req *http.Request) (interface{}, error)

var switcherFunc switcher

func router(s switcher) *mux.Router {
	switcherFunc = s
	r := mux.NewRouter()
	for _, v := range Config.urlServiceMaps {
		httpMethods := strings.Split(v[0], ",")
		path := v[1]
		methodName := v[2]
		r.HandleFunc(path, handler(methodName)).Methods(httpMethods...)
	}
	return r
}

var handler = func(methodName string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		ParseRequestForm(req)
		interceptors := getInterceptors(req)
		skipSwitch := false
		req, err := doBefore(&interceptors, resp, req)
		if err != nil {
			log.Println(err.Error())
			skipSwitch = true
		}
		if !skipSwitch {
			skipSwitch = doHijackerPreprocessor(resp, req)
		}
		if !skipSwitch {
			serviceResp, err := switcherFunc(methodName, resp, req)
			if err == nil {
				doPostprocessor(resp, req, serviceResp)
			} else {
				log.Println(err.Error())
				// do not 'return' here, this is not a bug
			}
		}
		err = doAfter(interceptors, resp, req)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}
}

func getInterceptors(req *http.Request) []Interceptor {
	interceptors := Interceptors(req)
	if len(interceptors) == 0 {
		interceptors = CommonInterceptors()
	}
	return interceptors
}

func doBefore(interceptors *[]Interceptor, resp http.ResponseWriter, req *http.Request) (request *http.Request, err error) {
	for index, i := range *interceptors {
		req, err = i.Before(resp, req)
		if err != nil {
			log.Println("error in interceptor!" + err.Error())
			*interceptors = (*interceptors)[0:index]
			return nil, err
		}
	}
	return req, nil
}

func doHijackerPreprocessor(resp http.ResponseWriter, req *http.Request) bool {
	pre := Preprocessor(req)
	if hijack := Hijacker(req); hijack != nil {
		if pre != nil {
			log.Printf("Warning: PreProcessor ignored, URL: %s", req.URL.String())
		}
		hijack(resp, req)
		return true
	} else if pre != nil {
		if err := pre(resp, req); err != nil {
			log.Println(err.Error())
			return true
		}
	}
	return false
}

func doPostprocessor(resp http.ResponseWriter, req *http.Request, serviceResponse interface{}) {
	// 1, run postprocessor, if any
	post := Postprocessor(req)
	if post != nil {
		post(resp, req, serviceResponse)
		return
	}

	// 2, parse serviceResponse with registered struct
	//if user defined struct registered {
	// TODO user can define a struct, which defines how data is mapped
	// from response to this struct, and how this struct is parsed into xml/json
	// return
	//}

	//3, return as json
	jsonBytes, err := JSON(serviceResponse)
	if err != nil {
		log.Println(err.Error())
	}
	resp.Write(jsonBytes)
}

func doAfter(interceptors []Interceptor, resp http.ResponseWriter, req *http.Request) (err error) {
	l := len(interceptors)
	for i := l - 1; i >= 0; i-- {
		req, err = interceptors[i].After(resp, req)
		if err != nil {
			log.Println("error in interceptor!")
			// if a doBefore() has run, then the corresponding doAfter() func
			// should run too, so don't return here.
			// Or, may be not?
			//return err
		}
	}
	return nil
}

// SetValue sets v to fieldValue according to fieldValue's Kind
func SetValue(fieldValue reflect.Value, v string) error {
	switch k := fieldValue.Kind(); k {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return errors.New("error int")
		}
		fieldValue.SetInt(i)
	case reflect.String:
		fieldValue.SetString(v)
	case reflect.Bool:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return errors.New("error bool")
		}
		fieldValue.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return errors.New("error float")
		}
		fieldValue.SetFloat(f)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return errors.New("error uint")
		}
		fieldValue.SetUint(u)
	default:
		return errors.New("not supported kind[" + k.String() + "]")
	}
	return nil
}

// ReflectValue returns a reflect.Value with v according to fieldValue's Kind
func ReflectValue(fieldValue reflect.Value, v string) (reflect.Value, error) {
	switch k := fieldValue.Kind(); k {
	case reflect.Int16:
		i, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return reflect.ValueOf(int16(0)), err
		}
		return reflect.ValueOf(int16(i)), nil
	case reflect.Int32:
		i, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return reflect.ValueOf(int32(0)), err
		}
		return reflect.ValueOf(int32(i)), nil
	case reflect.Int64:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return reflect.ValueOf(int64(0)), err
		}
		return reflect.ValueOf(int64(i)), nil
	case reflect.String:
		return reflect.ValueOf(v), nil
	case reflect.Bool:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return reflect.ValueOf(false), err
		}
		return reflect.ValueOf(bool(b)), nil
	case reflect.Float32:
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return reflect.ValueOf(0), errors.New("error float")
		}
		return reflect.ValueOf(float32(f)), nil
	case reflect.Float64:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return reflect.ValueOf(0), errors.New("error float")
		}
		return reflect.ValueOf(float64(f)), nil
		//case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		//	var u uint64
		//	if v == "" {
		//		u = 0
		//	} else {
		//		u, err := strconv.ParseUint(v, 10, 64)
		//		if err != nil {
		//			return reflect.ValueOf(u), errors.New("error uint")
		//		}
		//	}
		//	return reflect.ValueOf(u), nil
	default:
		return reflect.ValueOf(0), errors.New("not supported kind[" + k.String() + "]")
	}
}

//BuildStruct finds values from request, and set them to struct fields recursively
func BuildStruct(theType reflect.Type, theValue reflect.Value, req *http.Request) error {
	if theValue.Kind() == reflect.Invalid {
		fmt.Println("value is invalid, please check grpc-fieldmapping")
		return nil
	}
	fieldNum := theType.NumField()
	for i := 0; i < fieldNum; i++ {
		fieldName := theType.Field(i).Name
		fieldValue := theValue.FieldByName(fieldName)
		if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct {
			convertor := MessageFieldConvertor(fieldValue.Type().Elem())
			if convertor != nil {
				fieldValue.Set(convertor(req))
				continue
			}
			err := BuildStruct(fieldValue.Type().Elem(), fieldValue.Elem(), req)
			if err != nil {
				return err
			}
			continue
		}
		v, ok := findValue(fieldName, req)
		if !ok {
			continue
		}
		err := SetValue(fieldValue, v)
		if err != nil {
			fmt.Println(err)
		}
	}
	return nil
}

func findValue(fieldName string, req *http.Request) (string, bool) {
	snakeCaseName := ToSnakeCase(fieldName)
	v, ok := req.Form[snakeCaseName]
	if ok && len(v) > 0 {
		return v[0], true
	}
	ctxValue := req.Context().Value(fieldName)
	if ctxValue != nil {
		return ctxValue.(string), true
	}
	ctxValue = req.Context().Value(snakeCaseName)
	if ctxValue != nil {
		return ctxValue.(string), true
	}
	return "", false
}

// MakeParams returns a reflect.Value slice for grpc request
func MakeParams(req *http.Request, requestValue reflect.Value) []reflect.Value {
	params := make([]reflect.Value, 2)
	params[0] = reflect.ValueOf(req.Context())
	params[1] = requestValue
	return params
}

// ParseResult checks if error is nil, will returns an error if any
func ParseResult(result []reflect.Value) (serviceResponse interface{}, err error) {
	if result[1].Interface() == nil {
		return result[0].Interface(), nil
	}
	return nil, result[1].Interface().(error)
}

// BuildArgs returns a list of reflect.Value for thrift request
func BuildArgs(argsType reflect.Type, argsValue reflect.Value, req *http.Request, buildStructArg func(typeName string, req *http.Request) (v reflect.Value, err error)) ([]reflect.Value, error) {
	fieldNum := argsType.NumField()
	params := make([]reflect.Value, fieldNum)
	for i := 0; i < fieldNum; i++ {
		field := argsType.Field(i)
		fieldName := field.Name
		valueType := argsValue.FieldByName(fieldName).Type()
		if field.Type.Kind() == reflect.Ptr && valueType.Elem().Kind() == reflect.Struct {
			convertor := MessageFieldConvertor(valueType.Elem())
			if convertor != nil {
				params[i] = convertor(req)
				continue
			}
			structName := valueType.Elem().Name()
			v, err := buildStructArg(structName, req)
			if err != nil {
				return nil, err
			}
			params[i] = v
			continue
		}
		v, ok := findValue(fieldName, req)
		if !ok {
			fmt.Println("[info]value not found! key[" + fieldName + "], use default value[" + v + "]")
		}
		value, err := ReflectValue(argsValue.FieldByName(fieldName), v)
		if err != nil {
			fmt.Println("[info]using default value, error: " + err.Error())
		}
		params[i] = value
	}
	return params, nil
}
