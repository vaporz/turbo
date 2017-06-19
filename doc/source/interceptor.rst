.. _interceptor:

Interceptor
===========

Interceptors provide hook functions which run before or after a request.

Interceptors can be assigned to

    1. All URLs
    2. An URL path (which means a group of URLs)
    3. One URL
    4. One URL on HTTP methods

The more precise it is, the higher priority it has.

If interceptor A is assigned to URL '/abc' on HTTP method "GET", and interceptor B is assigned to all URLs, then A is executed when "GET /abc", and B is executed when "POST /abc".

Now, let's create an interceptor for URL "/eat_apple/{num:[0-9]+}" to log some info before and after a request.

Edit "yourservice/interceptor/log.go":

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

Then assign this interceptor to URL "/hello".

Edit "yourservice/grpcapi/component/components.go":

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
 
 func InitComponents() {
 +	turbo.Intercept([]string{"GET"}, "/hello", i.LogInterceptor{})
 }

Lastly, restart HTTP server and test::

 $ curl -w "\n" -X GET "http://localhost:8081/hello?your_name=Alice"
 {"message":"[grpc server]Hello, Alice"}

Check the server's console::

 $ go run yourservice/yourserviceapi.go
 [Before] Request URL:/hello
 [After] Request URL:/hello

We usually do something like validations in an interceptor, when the validation fails, the request halts, and returns an error message.

To do this, you can simply return an error:

.. code-block:: diff

 func (l LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
 	fmt.Println("[Before] Request URL:"+req.URL.Path)
 +	resp.Write([]byte("Encounter an error from LogInterceptor!\n"))
 -	return req, nil
 +	return req, errors.New("error!")
 }

Test::

 $ curl -w "\n" "http://localhost:8081/hello"
 Encounter an error from LogInterceptor!

