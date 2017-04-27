package framework

import (
	"github.com/gorilla/mux"
	"net/http"
)

func Router() *mux.Router {
	r := mux.NewRouter()
	for _, v := range UrlServiceMap {
		r.HandleFunc(v[1], handler(v[2])).Methods(v[0])
	}
	return r
}

var handler func(methodName string) func(http.ResponseWriter, *http.Request)

func InitHandler(h func(methodName string) func(http.ResponseWriter, *http.Request)){
	handler = h
}