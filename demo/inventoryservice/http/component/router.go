package component

import (
	pb "zx/demo/proto/inventoryservice"
	"github.com/gorilla/mux"
	"net/http"
	"golang.org/x/net/context"
	"log"
	"reflect"
	cm "zx/demo/common"
)

func Router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/videos", getVideoList).Methods("GET")
	r.HandleFunc("/videos/{id:[0-9]+}", handler(pb.GetVideoRequest{})).Methods("GET")
	return r
}

var getVideoList = func(resp http.ResponseWriter, req *http.Request) {
	request := &pb.GetVideoListRequest{"111222"}
	result, err := InventoryService().GetVideoList(context.Background(), request)
	if err != nil {
		log.Println(result)
	}
	resp.Write([]byte(result.String() + "\n"))
}

var handler = func(r interface{}) func(http.ResponseWriter, *http.Request) {

	return func(resp http.ResponseWriter, req *http.Request) { // TODO 用模版生成
		mergeMuxVars(req)
		request := pb.GetVideoRequest{}
		theType := reflect.TypeOf(request)
		log.Println(theType.String())
		theValue := reflect.ValueOf(&request).Elem()
		log.Println(theValue.String())
		fieldNum := theType.NumField()
		for i := 0; i < fieldNum; i++ {
			fieldName := theType.Field(i).Name
			v, ok := req.Form[cm.ToSnakeCase(fieldName)]
			if ok && len(v) > 0 {
				theValue.FieldByName(fieldName).SetString(v[0])
			}
		}
		params := make([]reflect.Value, 2)
		params[0] = reflect.ValueOf(req.Context())
		params[1] = reflect.ValueOf(&request)
		result := reflect.ValueOf(InventoryService()).MethodByName("GetVideo").Call(params)

		video := result[0].Interface().(*pb.GetVideoResponse)
		if result[1].Interface() != nil {
			log.Println(result)
		}
		resp.Write([]byte(video.String() + "\n"))
	}
}

func mergeMuxVars(req *http.Request) {
	muxVars := mux.Vars(req)
	if muxVars == nil {
		return
	}
	req.ParseForm()
	for key, valueArr := range req.Form {
		if v, ok := muxVars[key]; ok {
			// route params comes first
			req.Form[key] = append([]string{v}, valueArr...)
			delete(muxVars, key)
		}
	}
	for key, value := range muxVars {
		req.Form[key] = []string{value}
	}
}
