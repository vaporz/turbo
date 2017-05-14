package interceptor

import (
	"turbo"
	"net/http"
	"fmt"
)

type LogInterceptor struct {
	// optional, BaseInterceptor allows you to create an interceptor which implements
	// Before() or After() only, or none of them.
	// If you were to implement both, you can remove this line.
	turbo.BaseInterceptor
	Msg string
}

func (l LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	fmt.Println("[Before][" + l.Msg + "] Request URL:" + req.URL.Path)
	//resp.Write([]byte("Encounter an error from LogInterceptor!\n"))
	//return req, errors.New("error!")
	return req, nil

	//fmt.Println("[Before][" + l.Msg + "] Request URL:" + req.URL.Path)
	//ctx := req.Context()
	//ctx = context.WithValue(ctx, "some_id", "1234567")
	//return req.WithContext(ctx), nil
}

func (l LogInterceptor) After(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	fmt.Println("[After] Request URL:" + req.URL.Path)
	return req, nil
}
