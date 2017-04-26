package main

import (
	client "zx/demo/inventoryservice/http/component/clients"
	c "zx/demo/inventoryservice/http/component"
	"net/http"
	"log"
)

func main() {
	client.InitGrpcConnection()
	defer client.CloseGrpcConnection()
	s := &http.Server{
		Addr:    ":8081",
		Handler: c.Router(), // TODO interceptors: loginRequired, loggerContext, formatter
	}
	log.Fatal(s.ListenAndServe())
}
