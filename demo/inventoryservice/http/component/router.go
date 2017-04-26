package component

import (
	"github.com/gorilla/mux"
	"zx/demo/generator/gen"
)

func Router() *mux.Router {
	r := mux.NewRouter()
	for _, v := range UrlServiceMap {
		r.HandleFunc(v[1], gen.Handler(v[2])).Methods(v[0])
	}
	return r
}
