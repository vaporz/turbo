package component

import (
	"github.com/gorilla/mux"
	"zx/demo/inventoryservice/http/gen"
)

func Router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/videos", gen.Handler("getVideoList")).Methods("GET")
	r.HandleFunc("/videos/{id:[0-9]+}", gen.Handler("GetVideo")).Methods("GET")
	return r
}
