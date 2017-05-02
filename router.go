package turbo

import (
	"github.com/gorilla/mux"
	"net/http"
	"log"
)

func router() *mux.Router {
	r := mux.NewRouter()
	for _, v := range UrlServiceMap {
		r.HandleFunc(v[1], handler(v[2])).Methods(v[0])
	}
	return r
}

var switcher func(methodName string, resp http.ResponseWriter, req *http.Request)

func initSwitcher(s func(string, http.ResponseWriter, *http.Request)) {
	switcher = s
}

type Interceptor interface {
	Before(http.ResponseWriter, *http.Request) error
	After(http.ResponseWriter, *http.Request) error
}

type BaseInterceptor struct{}

func (i BaseInterceptor) Before(http.ResponseWriter, *http.Request) error {
	return nil
}

func (i BaseInterceptor) After(http.ResponseWriter, *http.Request) error {
	return nil
}

var handler = func(methodName string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		ParseRequestForm(req)
		interceptors := getInterceptors(methodName)
		err := doBefore(interceptors, resp, req)
		if err != nil {
			return
		}
		skipSwitch := doHijackerPreprocessor(methodName, resp, req)
		if !skipSwitch {
			switcher(methodName, resp, req)
		}
		err = doAfter(interceptors, resp, req)
		if err != nil {
			return
		}
	}
}

func getInterceptors(methodName string) []Interceptor {
	interceptors, ok := Interceptors(methodName)
	if !ok {
		interceptors, ok = CommonInterceptors()
	}
	if !ok {
		interceptors = EmptyInterceptors()
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

func doHijackerPreprocessor(methodName string, resp http.ResponseWriter, req *http.Request) bool {
	if hijack := Hijacker(methodName); hijack != nil {
		hijack(resp, req)
		return true
	} else if preprocessor := Preprocessor(methodName); preprocessor != nil {
		if err := preprocessor(resp, req); err != nil {
			return true
		}
	}
	return false
}

func doAfter(interceptors []Interceptor, resp http.ResponseWriter, req *http.Request) error {
	l := len(interceptors)
	for i := l - 1; i > 0; i-- {
		err := interceptors[i].After(resp, req)
		if err != nil {
			log.Println("error in interceptor!")
			return err
		}
	}
	return nil
}
