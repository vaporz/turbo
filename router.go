package turbo

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"errors"
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
	if hijack := Hijacker(req); hijack != nil {
		hijack(resp, req)
		// TODO warn if there are preprocessor
		return true
	} else if preprocessor := Preprocessor(req); preprocessor != nil {
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

func SetValue(theValue reflect.Value, fieldName, v string) error {
	switch theValue.FieldByName(fieldName).Kind() {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return errors.New("error int!!")
		}
		theValue.FieldByName(fieldName).SetInt(i)
	case reflect.String:
		theValue.FieldByName(fieldName).SetString(v)
	case reflect.Bool:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return errors.New("error bool!!")
		}
		theValue.FieldByName(fieldName).SetBool(b)
	default:
		return errors.New("error!!")
	}
	return nil
}
