.. _interceptor:

Interceptor
===========

拦截器提供了在发送request之前和之后做一些事情的机会。

拦截器可以被设置给

    1. 所有URL
    2. 某个URL根路径 (也就是一组URL)
    3. 单个URL
    4. 单个URL，并且指定 HTTP 谓语

设置的粒度越精确，拦截器的优先级就越高

比如，拦截器 A 设置在 'GET /abc'，拦截器 B 设置给了所有URL，那么，当请求"GET /abc"时，拦截器A会被执行，B不会被执行，但请求"POST /abc"时，只有B会被执行。

下面，让我们来创建一个拦截器，设置给"/hello"，用来记录一些log。

编辑 "yourservice/interceptor/log.go":

.. code-block:: go

 package interceptor
 
 import (
 	"github.com/vaporz/turbo"
 	"fmt"
 	"net/http"
 )
 
 type LogInterceptor struct {
 	// optional, BaseInterceptor allows you to create an interceptor which implements
 	// Before() or After() only, or none of them.
 	// If you were to implement both, you can remove this line.
 	turbo.BaseInterceptor
 }
 
 func (l LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
 	fmt.Println("[Before] Request URL:"+req.URL.Path)
 	return req, nil
 }
 
 func (l LogInterceptor) After(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
 	fmt.Println("[After] Request URL:"+req.URL.Path)
 	return req, nil
 }

然后把它设置给 URL "/hello"。

编辑 "yourservice/grpcapi/component/components.go":

.. code-block:: diff

 package component
 
 import (
 	"github.com/vaporz/turbo-example/yourservice/gen/proto"
 	"google.golang.org/grpc"
 +	i "github.com/vaporz/turbo-example/yourservice/interceptor"
 )
 
 func GrpcClient(conn *grpc.ClientConn) interface{} {
 	return proto.NewTestServiceClient(conn)
 }

 func RegisterComponents(s *turbo.GrpcServer) {
 +	 s.RegisterComponent("LogInterceptor", i.LogInterceptor{})
 }

编辑 "yourservice/service.yaml":

.. code-block:: diff

 +interceptor:
 +  - GET /hello LogInterceptor

最后，重启服务并测试::

 $ curl -w "\n" -X GET "http://localhost:8081/hello?your_name=Alice"
 {"message":"[grpc server]Hello, Alice"}

查看HTTP server的控制台::

 $ go run yourservice/yourserviceapi.go
 [Before] Request URL:/hello
 [After] Request URL:/hello

拦截器经常被用来做一些校验工作，当校验失败时，请求会被中止，并返回一个错误。

很简单，你只需要这样:

.. code-block:: diff

 func (l LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
 	fmt.Println("[Before] Request URL:"+req.URL.Path)
 +	resp.Write([]byte("Encounter an error from LogInterceptor!\n"))
 -	return req, nil
 +	return req, errors.New("error!")
 }

测试::

 $ curl -w "\n" "http://localhost:8081/hello"
 Encounter an error from LogInterceptor!

