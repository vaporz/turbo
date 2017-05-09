package turbo

import (
	"errors"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"reflect"
	"strconv"
)

func router(switcherFunc func(methodName string, resp http.ResponseWriter, req *http.Request)) *mux.Router {
	switcher = switcherFunc
	r := mux.NewRouter()
	for _, v := range UrlServiceMap {
		r.HandleFunc(v[1], handler(v[2])).Methods(v[0])
	}
	return r
}

var switcher func(methodName string, resp http.ResponseWriter, req *http.Request)

var handler = func(methodName string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		ParseRequestForm(req)
		interceptors := getInterceptors(req)
		err := doBefore(interceptors, resp, req)
		if err != nil {
			return
		}
		skipSwitch := doHijackerPreprocessor(resp, req)
		if !skipSwitch {
			switcher(methodName, resp, req)
		}
		err = doAfter(interceptors, resp, req)
		if err != nil {
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

func doBefore(interceptors []Interceptor, resp http.ResponseWriter, req *http.Request) error {
	for _, i := range interceptors {
		err := i.Before(resp, req)
		if err != nil {
			log.Println("error in interceptor!" + err.Error())
			return err
		}
	}
	return nil
}

func doHijackerPreprocessor(resp http.ResponseWriter, req *http.Request) bool {
	preprocessor := Preprocessor(req)
	if hijack := Hijacker(req); hijack != nil {
		if preprocessor != nil {
			//TODO packaging warning and error lib
			log.Printf("Warning: There is a preprocessor on %s not performed because of hijacker\n", req.URL.String())
		}
		hijack(resp, req)
		return true
	} else if preprocessor != nil {
		if err := preprocessor(resp, req); err != nil {
			return true
		}
	}
	return false
}

func doAfter(interceptors []Interceptor, resp http.ResponseWriter, req *http.Request) error {
	l := len(interceptors)
	for i := l - 1; i >= 0; i-- {
		err := interceptors[i].After(resp, req)
		if err != nil {
			log.Println("error in interceptor!")
			return err
		}
	}
	return nil
}

func SetValue(fieldValue reflect.Value, v string) error {
	switch fieldValue.Kind() {
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
	case reflect.Slice, reflect.Interface, reflect.Struct:
		// only basic types supported
		return errors.New("type not supported")
	default:
		return errors.New("error")
	}
	return nil
}

func ReflectValue(fieldValue reflect.Value, v string) (value reflect.Value, err error) {
	switch fieldValue.Kind() {
	case reflect.Int16:
		var i int64
		if v == "" {
			i = 0
		} else {
			i, err = strconv.ParseInt(v, 10, 16)
			if err != nil {
				return reflect.ValueOf(i), errors.New("error int")
			}
		}
		return reflect.ValueOf(int16(i)), nil
	case reflect.Int32:
		var i int64
		if v == "" {
			i = 0
		} else {
			i, err = strconv.ParseInt(v, 10, 32)
			if err != nil {
				return reflect.ValueOf(i), errors.New("error int")
			}
		}
		return reflect.ValueOf(int32(i)), nil
	case reflect.Int64:
		var i int64
		if v == "" {
			i = 0
		} else {
			i, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return reflect.ValueOf(i), errors.New("error int")
			}
		}
		return reflect.ValueOf(int64(i)), nil
	case reflect.String:
		return reflect.ValueOf(v), nil
	case reflect.Bool:
		var b bool
		if v == "" {
			b = false
		} else {
			b, err = strconv.ParseBool(v)
			if err != nil {
				return reflect.ValueOf(b), errors.New("error bool")
			}
		}
		return reflect.ValueOf(bool(b)), nil
	case reflect.Float32:
		var f float64
		if v == "" {
			f = 0
		} else {
			f, err = strconv.ParseFloat(v, 64)
			if err != nil {
				return reflect.ValueOf(f), errors.New("error float")
			}
		}
		return reflect.ValueOf(float32(f)), nil
	case reflect.Float64:
		var f float64
		if v == "" {
			f = 0
		} else {
			f, err = strconv.ParseFloat(v, 64)
			if err != nil {
				return reflect.ValueOf(f), errors.New("error float")
			}
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
	case reflect.Slice, reflect.Interface, reflect.Struct:
		// only basic types supported
		return reflect.ValueOf(0), errors.New("type not supported")
	default:
		return reflect.ValueOf(0), errors.New("error")
	}
}
