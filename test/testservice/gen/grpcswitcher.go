package gen

import (
	g "github.com/vaporz/turbo/test/testservice/gen/proto"
	"github.com/vaporz/turbo"
	"net/http"
	"errors"
)

/*
this is a generated file, DO NOT EDIT!
 */
// GrpcSwitcher is a runtime func with which a server starts.
var GrpcSwitcher = func(s turbo.Servable, methodName string, resp http.ResponseWriter, req *http.Request) (rpcResponse interface{}, err error) {
	callOptions, header, trailer, peer := turbo.CallOptions(methodName, req)
	switch methodName {
	case "SayHello":
		request := &g.SayHelloRequest{Values: &g.CommonValues{}, }
		err = turbo.BuildRequest(s, request, req)
		if err != nil {
			return nil, err
		}
		rpcResponse, err = s.Service().(g.TestServiceClient).SayHello(req.Context(), request, callOptions...)
	default:
		return nil, errors.New("No such method[" + methodName + "]")
	}
	turbo.WithCallOptions(req, header, trailer, peer)
	return
}
