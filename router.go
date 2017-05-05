package turbo

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
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