package turbo

import (
	"errors"
	// TODO support logging levels, log file path, etc.
	"bytes"
	"encoding/json"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type switcher func(s *Server, methodName string, resp http.ResponseWriter, req *http.Request) (interface{}, error)

func router(s *Server) *mux.Router {
	r := mux.NewRouter()
	for _, v := range s.Config.mappings[urlServiceMaps] {
		httpMethods := strings.Split(v[0], ",")
		path := v[1]
		methodName := v[2]
		r.HandleFunc(path, handler(s, methodName)).Methods(httpMethods...)
	}
	return r
}

func handler(s *Server, methodName string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		parseRequestForm(req)
		interceptors := getInterceptors(s, req)
		req, err := doBefore(&interceptors, resp, req)
		if err == nil {
			doRequest(s, methodName, resp, req)
		} else {
			s.Components.errorHandlerFunc()(resp, req, err)
		}
		doAfter(interceptors, resp, req)
	}
}

func getInterceptors(s *Server, req *http.Request) []Interceptor {
	interceptors := s.Components.Interceptors(req)
	if len(interceptors) == 0 {
		interceptors = s.Components.CommonInterceptors()
	}
	return interceptors
}

func doBefore(interceptors *[]Interceptor, resp http.ResponseWriter, req *http.Request) (request *http.Request, err error) {
	for index, i := range *interceptors {
		req, err = i.Before(resp, req)
		if err != nil {
			log.Errorln("error in Before(): ", err.Error())
			*interceptors = (*interceptors)[0:index]
			return req, err
		}
	}
	return req, nil
}

func doRequest(s *Server, methodName string, resp http.ResponseWriter, req *http.Request) {
	if hijack := s.Components.Hijacker(req); hijack != nil {
		hijack(resp, req)
		return
	}
	err := doPreprocessor(s, resp, req)
	if err != nil {
		s.Components.errorHandlerFunc()(resp, req, err)
		return
	}
	serviceResp, err := s.switcherFunc(s, methodName, resp, req)
	if err != nil {
		s.Components.errorHandlerFunc()(resp, req, err)
		return
	}
	doPostprocessor(s, resp, req, serviceResp, err)
}

func doPreprocessor(s *Server, resp http.ResponseWriter, req *http.Request) error {
	if pre := s.Components.Preprocessor(req); pre != nil {
		if err := pre(resp, req); err != nil {
			log.Println(err.Error())
			return err
		}
	}
	return nil
}

func doPostprocessor(s *Server, resp http.ResponseWriter, req *http.Request, serviceResponse interface{}, err error) {
	// 1, run Postprocessor, if any
	post := s.Components.Postprocessor(req)
	if post != nil {
		post(resp, req, serviceResponse, err)
		return
	}

	// 2, parse serviceResponse with registered struct
	//if user defined struct registered {
	// TODO user can define a struct, which defines how data is mapped
	// from response to this struct, and how this struct is parsed into xml/json
	// return
	//}

	//3, return as json
	m := Marshaler{
		FilterProtoJson: s.Config.FilterProtoJson(),
		EmitZeroValues:  s.Config.FilterProtoJsonEmitZeroValues(),
		Int64AsNumber:   s.Config.FilterProtoJsonInt64AsNumber(),
	}
	jsonBytes, err := m.JSON(serviceResponse)
	if err == nil {
		resp.Write(jsonBytes)
	} else {
		log.Println(err.Error())
		resp.Write([]byte(err.Error()))
	}
}

func doAfter(interceptors []Interceptor, resp http.ResponseWriter, req *http.Request) (err error) {
	l := len(interceptors)
	for i := l - 1; i >= 0; i-- {
		req, err = interceptors[i].After(resp, req)
		if err != nil {
			log.Errorln("error in After(): ", err.Error())
		}
	}
	return nil
}

//BuildStruct finds values from request, and set them to struct fields recursively
func BuildStruct(s *Server, theType reflect.Type, theValue reflect.Value, req *http.Request) error {
	if theValue.Kind() == reflect.Invalid {
		log.Info("value is invalid, please check grpc-fieldmapping")
		return nil
	}
	fieldNum := theType.NumField()
	for i := 0; i < fieldNum; i++ {
		fieldName := theType.Field(i).Name
		fieldValue := theValue.FieldByName(fieldName)
		if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct {
			convertor := s.Components.MessageFieldConvertor(fieldValue.Type().Elem().Name())
			if convertor != nil {
				fieldValue.Set(convertor(req))
				continue
			}
			err := BuildStruct(s, fieldValue.Type().Elem(), fieldValue.Elem(), req)
			if err != nil {
				return err
			}
			continue
		}
		v, ok := findValue(fieldName, req)
		if !ok {
			continue
		}
		err := setValue(fieldValue, v)
		logErrorIf(err)
	}
	return nil
}

// setValue sets v to fieldValue according to fieldValue's Kind
func setValue(fieldValue reflect.Value, v string) error {
	var err error
	switch k := fieldValue.Kind(); k {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		var i int64
		i, err = strconv.ParseInt(v, 10, 64)
		fieldValue.SetInt(i)
	case reflect.String:
		fieldValue.SetString(v)
	case reflect.Bool:
		var b bool
		b, err = strconv.ParseBool(v)
		fieldValue.SetBool(b)
	case reflect.Float32, reflect.Float64:
		var f float64
		f, err = strconv.ParseFloat(v, 64)
		fieldValue.SetFloat(f)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var u uint64
		u, err = strconv.ParseUint(v, 10, 64)
		fieldValue.SetUint(u)
	default:
		return errors.New("not supported kind[" + k.String() + "]")
	}
	return err
}

// BuildArgs returns a list of reflect.Value for thrift request
func BuildArgs(s *Server, argsType reflect.Type, argsValue reflect.Value, req *http.Request, buildStructArg func(s *Server, typeName string, req *http.Request) (v reflect.Value, err error)) ([]reflect.Value, error) {
	fieldNum := argsType.NumField()
	params := make([]reflect.Value, fieldNum)
	for i := 0; i < fieldNum; i++ {
		field := argsType.Field(i)
		fieldName := field.Name
		valueType := argsValue.FieldByName(fieldName).Type()
		if field.Type.Kind() == reflect.Ptr && valueType.Elem().Kind() == reflect.Struct {
			convertor := s.Components.MessageFieldConvertor(valueType.Elem().Name())
			if convertor != nil {
				params[i] = convertor(req)
				continue
			}
			structName := valueType.Elem().Name()
			v, err := buildStructArg(s, structName, req)
			if err != nil {
				return nil, err
			}
			params[i] = v
			continue
		}
		v, _ := findValue(fieldName, req)
		value, _ := reflectValue(argsValue.FieldByName(fieldName), v)
		params[i] = value
	}
	return params, nil
}

// reflectValue returns a reflect.Value with v according to fieldValue's Kind
func reflectValue(fieldValue reflect.Value, v string) (reflect.Value, error) {
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
	case reflect.Float64:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return reflect.ValueOf(float64(0)), errors.New("error float")
		}
		return reflect.ValueOf(float64(f)), nil
	default:
		return reflect.ValueOf(0), errors.New("not supported kind[" + k.String() + "]")
	}
}

func findValue(fieldName string, req *http.Request) (string, bool) {
	lowerCasesName := strings.ToLower(fieldName)
	snakeCaseName := ToSnakeCase(fieldName)

	v, ok := req.Form[lowerCasesName]
	if ok && len(v) > 0 {
		return v[0], true
	}
	v, ok = req.Form[snakeCaseName]
	if ok && len(v) > 0 {
		return v[0], true
	}
	ctxValue := req.Context().Value(fieldName)
	if ctxValue != nil {
		return ctxValue.(string), true
	}
	ctxValue = req.Context().Value(lowerCasesName)
	if ctxValue != nil {
		return ctxValue.(string), true
	}
	ctxValue = req.Context().Value(snakeCaseName)
	if ctxValue != nil {
		return ctxValue.(string), true
	}
	return "", false
}

func BuildRequest(s *Server, v proto.Message, req *http.Request) error {
	var err error
	if contentTypes, ok := req.Header["Content-Type"]; ok && contentTypes[0] == "application/json" {
		unmarshaler := &jsonpb.Unmarshaler{AllowUnknownFields: true}
		err = unmarshaler.Unmarshal(req.Body, v)
		if err != nil {
			return err
		}
		setPathParams(s, reflect.TypeOf(v).Elem(), reflect.ValueOf(v).Elem(), req)
	} else {
		err = BuildStruct(s, reflect.TypeOf(v).Elem(), reflect.ValueOf(v).Elem(), req)
	}
	return err
}

func BuildThriftRequest(s *Server, args interface{}, req *http.Request, buildStructArg func(s *Server, typeName string, req *http.Request) (v reflect.Value, err error)) ([]reflect.Value, error) {
	var err error
	var params []reflect.Value
	if contentTypes, ok := req.Header["Content-Type"]; ok && contentTypes[0] == "application/json" {
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		v := reflect.New(reflect.ValueOf(args).Field(0).Type().Elem()).Interface()
		err := json.Unmarshal(buf.Bytes(), v)
		// TODO should panic?
		if err != nil {
			return params, err
		}
		setPathParams(s, reflect.TypeOf(v).Elem(), reflect.ValueOf(v).Elem(), req)
		params = make([]reflect.Value, 1)
		params[0] = reflect.ValueOf(v)
	} else {
		params, err = BuildArgs(s, reflect.TypeOf(args), reflect.ValueOf(args), req, buildStructArg)
	}
	return params, err
}

func setPathParams(s *Server, theType reflect.Type, theValue reflect.Value, req *http.Request) error {
	fieldNum := theType.NumField()
	pathParams := mux.Vars(req)
	for i := 0; i < fieldNum; i++ {
		fieldName := theType.Field(i).Name
		fieldValue := theValue.FieldByName(fieldName)
		if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct {
			err := setPathParams(s, fieldValue.Type().Elem(), fieldValue.Elem(), req)
			if err != nil {
				return err
			}
			continue
		}
		v, ok := findPathParamValue(fieldName, pathParams)
		if !ok {
			continue
		}
		err := setValue(fieldValue, v)
		logErrorIf(err)
	}
	return nil
}

func findPathParamValue(fieldName string, pathParams map[string]string) (string, bool) {
	lowerCasesName := strings.ToLower(fieldName)
	v, ok := pathParams[lowerCasesName]
	if ok && len(v) > 0 {
		return v, true
	}
	snakeCaseName := ToSnakeCase(fieldName)
	v, ok = pathParams[snakeCaseName]
	if ok && len(v) > 0 {
		return v, true
	}
	return "", false
}
