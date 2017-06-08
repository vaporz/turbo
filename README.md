# Turbo  [![Build Status](https://travis-ci.org/vaporz/turbo.svg?branch=master)](https://travis-ci.org/vaporz/turbo) [![Coverage Status](https://coveralls.io/repos/github/vaporz/turbo/badge.svg?branch=master)](https://coveralls.io/github/vaporz/turbo?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/vaporz/turbo)](https://goreportcard.com/report/github.com/vaporz/turbo) [![codebeat badge](https://codebeat.co/badges/7a166e48-dae1-454c-b925-4fbcd3f1f461)](https://codebeat.co/projects/github-com-vaporz-turbo-master) [![Code Climate](https://codeclimate.com/github/vaporz/turbo/badges/gpa.svg)](https://codeclimate.com/github/vaporz/turbo)

<b>WORK IN PROGRESS! There may be many bugs, and the README may not be synced in time as the codes changed.</b>

## Features
 * Turbo generates a reverse-proxy server which translates a HTTP request into a grpc/Thrift request.
 * Support gRPC and [Thrift](#support_thrift).
 * [Interceptor](#interceptor).
 * [PreProcessor and PostProcessor](#preprocessor_and_postprocessor): customizable URL-RPC mapping process.
 * [Hijacker](#hijacker): Take over requests, do anything you want!
 * [MessageFieldConvertor](#message_field_convertor): Tell Turbo how to set a struct field.
 * Modify and reload configuration file at runtime! Without restarting service.
## Index
 * [Create a service on the fly](#create_a_service)
 * [Command line tools](#command_line_tools)
 * [Rules and Conventions](#rules_and_conventions)
 * [How to add a new API](#add_a_new_api)
 * [Use a shared struct](#use_a_shared_struct)
 * [Interceptor](#interceptor)
 * [PreProcessor and PostProcessor](#preprocessor_and_postprocessor)
 * [Hijacker](#hijacker)
 * [MessageFieldConvertor](#message_field_convertor)
 * [Error Handler](#error_handler)
 * [Thrift support](#support_thrift)
 * [Configs in service.yaml](#service_yaml)
## <a name="create_a_service"></a>Create a service on the fly
### 0, Before the start
Obviously, you have to install [Golang](https://golang.org) and [Protocol buffers](https://developers.google.com/protocol-buffers/) first.  
(Recommended) And install [glide](https://github.com/Masterminds/glide) for dependency management.  
(Not Recommended) Or install required packages manually.
```sh
go get google.golang.org/grpc
go get git.apache.org/thrift.git/lib/go/thrift
go get github.com/kylelemons/go-gypsy/yaml
go get github.com/gorilla/mux
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/bitly/go-simplejson
```

### 1, Install Turbo command line tools
```sh
go get github.com/vaporz/turbo
cd github.com/vaporz/turbo/turbo
glide install
go install
cd github.com/vaporz/turbo/protoc-gen-buildfields
go install
```

### 2, Create your service
```sh
$ turbo create package/path/to/yourservice YourService -r grpc
```
Directory "$GOPATH/src/package/path/to/yourservice" should appear.  
There're also some generated files in this folder.  
Example project:  
https://github.com/vaporz/turbo-example/tree/master/yourservice

### 3, Run
That's it! Now let's Play!

Start both gRPC server and HTTP server:
```sh
cd $GOPATH/src/package/path/to/yourservice
go run main.go
```
Send a request:
```sh
$ curl -w "\n" "http://localhost:8081/hello?your_name=Alice"
message:"Hello, Alice"
```
Or you can start gRPC server and HTTP server separately:
```sh
$ cd $GOPATH/src/package/path/to/yourservice
# start grpc service
$ go run grpcservice/yourservice.go
# start http server
$ go run grpcapi/yourserviceapi.go
```

## <a name="command_line_tools"></a>Command line tools
### turbo create package_path ServiceName -r (grpc|thrift)
'turbo create' creates a project with runnable HTTP server and gRPC/Thrift server.  
'ServiceName' **MUST** be a CamelCase string.  
Project structure:
```sh
$ turbo create package/path/to/yourservice YourService -r grpc
$ cd $GOPATH/src/package/path/to/yourservice
$ tree
.
|-- gen
|   |-- grpcswitcher.go
|   `-- proto
|       `-- yourservice.pb.go
|-- grpcapi
|   |-- component
|   |   `-- components.go
|   `-- yourserviceapi.go
|-- grpcservice
|   |-- impl
|   |   `-- yourserviceimpl.go
|   `-- yourservice.go
|-- main.go
|-- service.yaml
`-- yourservice.proto
```
### turbo generate package_path -r (grpc/thrift) -I (absolute_path_to_proto/thrift_files) -I ...
'turbo generate' generates switcher.go and [service_name].pb.go to 'gen' directory.  
This command is useful when either service.yaml or [service_name].proto|.thrift is changed.  
For example, add a new API, change an existing API, change url-grpc mapping, etc.  
Example:
```sh
$ turbo generate package/path/to/yourservice -r grpc 
-I $GOPATH/src/package/path/to/yourservice -I $GOPATH/src/shared
```
"-I" can appear more than one time, if you have a shared file like "shared.proto" imported from other path.

## <a name="rules_and_conventions"></a>Rules and Conventions
There are some rules when you use turbo.
 * When defining a gRPC service, if the name of a gRPC method is "methodName", then the name
 of request message and response message **MUST** be "MethodNameRequest" and "MethodNameResponse".  
   When defining a thrift service, the response message's name **MUST** be "MethodNameResponse".
 * If multiple paths are assigned to $GOPATH(divided by ':'), then the first path is used by turbo as GOPATH.
 * When parsing request parameters, values from URL path has a higher priority than those from query string, body or context.Context.  
e.g. In a request like "GET /book/1234?id=5678", both "1234" and "5678" are values to "id", but "1234" is picked as value to key "id".
 * The value of a key with all lower case characters has a higher priority to the value of a key with upper case
 characters.  
 e.g. In a request like "GET /book?id=1234&ID=5678", "1234" is used for key "id".
 * A parameter's key is case-insensitive to turbo, in fact, internally turbo will cast keys to lower case characters before further use.  
 e.g. In a request like "GET /book?ID=1234", turbo will see this query string as "id=1234".

## <a name="add_a_new_api"></a>How to add a new API
### 1, Define new gRPC API
Modify "yourservice.proto", add new method "eatApple":
```diff
+message EatAppleRequest {
+    string num = 1;
+}
+
+message EatAppleResponse {
+    string message = 1;
+}
 
 service YourService {
     rpc sayHello (SayHelloRequest) returns (SayHelloResponse) {}
+    rpc eatApple (EatAppleRequest) returns (EatAppleResponse) {}
 }
```
### 2, Add new url-grpc mapping
Modify "service.yaml":
```diff
 urlmapping:
   - GET /hello SayHello
+  - GET /eat_apple/{num:[0-9]+} EatApple
```
### 3, Generate new codes
```sh
$ turbo generate package/path/to/yourservice -r grpc -I $GOPATH/src/package/path/to/yourservice
```

### 4, Implement new gRPC method
Edit "grpcservice/impl/yourserviceimpl.go":
```diff
+func (s *YourService) EatApple(ctx context.Context, req *proto.EatAppleRequest) (*proto.EatAppleResponse, error) {
+	return &proto.EatAppleResponse{Message: "Good taste! Apple num=" + strconv.FormatInt(int64(req.Num), 10)}, nil
+}
```

Now, restart both HTTP and gRPC server, then test:
```sh
# start grpc service
$ go run grpcservice/yourservice.go
# start http server
$ go run grpcapi/yourserviceapi.go
# test
$ curl "http://localhost:8081/eat_apple/5"
message:"Good taste! Apple num=5"
```
## <a name="use_a_shared_struct"></a>Use a shared struct
Sometimes we want to add some info to all requests from frontend server to backend server.  
So we define a new message in a file like "shared.proto" in a separate path.  
Then we add this new message to other message as a struct field.

Let me show you how to do this in grpc with Turbo.
### 1, Create "shared.proto" and define a new message
Create folder "common"(or any other name)
```sh
$ mkdir -p $GOPATH/src/package/path/to/common
```
Create file "shared.proto" in folder "common":
```proto
syntax = "proto3";
package proto;

message CommonValues {
    int64 someId = 1;
}
```
### 2, Add new field to a request message
Edit "yourservice.proto":
```diff
syntax = "proto3";
package proto;
+
+import "shared.proto";

message SayHelloRequest {
    string yourName = 1;
+    CommonValues values = 2;
}
```
### 3, Generate codes
```sh
$ turbo generate package/path/to/yourservice -r grpc -I $GOPATH/src/package/path/to/yourservice
 -I $GOPATH/src/package/path/to/common
```
Done!  
Before test, edit youserviceimpl.go to return "someId":
```diff
func (s *YourService) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
+	someId := strconv.FormatInt(req.Values.SomeId, 10)
-	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
+	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName + ", someId=" + someId}, nil
}
```
Restart grpc server and test:
```sh
$ curl -w "\n" -X GET "http://localhost:8081/hello?your_name=Alice&some_id=12345"
{"message":"[grpc server]Hello, Alice, someId=12345"}
```
Turbo is smart, query string param "some_id" is mapped into SayHelloRequest.CommonValues.SomeId automatically.  
Besides query string, You can also map "SomeId" from URL route params, or context.Context which is set from interceptors.

## <a name="interceptro"></a>Interceptor
Interceptors provide hook functions which run before or after a request.  
Interceptors can be assigned to
 * 1, All URLs
 * 2, An URL path (which means a group of URLs)
 * 3, One URL
 * 4, One URL on HTTP methods

The more precise it is, the higher priority it has.  
If interceptor A is assigned to URL '/abc' on HTTP method "GET", and interceptor B is assigned to all URLs, then A is executed when "GET /abc", and B is executed when "POST /abc".

Now, let's create an interceptor for URL "/eat_apple/{num:[0-9]+}" to log some info before and after a request.  
Edit "yourservice/interceptor/log.go":
```go
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
```
Then assign this interceptor to URL "/hello":<br>
Edit "yourservice/grpcapi/component/components.go":
```diff
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
```
Lastly, restart HTTP server and test:
```sh
$ curl -w "\n" -X GET "http://localhost:8081/hello?your_name=Alice"
{"message":"[grpc server]Hello, Alice"}
```
Check the server's console:
```sh
$ go run yourservice/yourserviceapi.go
[Before] Request URL:/hello
[After] Request URL:/hello
```

We usually do something like validations in an interceptor, when the validation fails, the request halts, and returns an error message.  
To do this, you can simply return an error:
```diff
func (l LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) (*http.Request, error) {
	fmt.Println("[Before] Request URL:"+req.URL.Path)
+	resp.Write([]byte("Encounter an error from LogInterceptor!\n"))
-	return req, nil
+	return req, errors.New("error!")
}
```
Test:
```sh
$ curl -w "\n" "http://localhost:8081/hello"
Encounter an error from LogInterceptor!
```
## <a name="preprocessor_and_postprocessor"></a>Preprocessor and Postprocessor
What if I want to do something particularly for some API?<br>
Preprocessor/Hijacker comes to help!<br>
If both Preprocessors and hijackers are assigned to an URL, only the last hijacker assigned is active.

### Preprocessor
Preprocessors are executed just after all Before() functions from interceptors, and before
sending requests to gRPC server.  
Preprocessors can be used to do something particularly for an API. For example, parameter value validations,
setting default values, parsing values, logging, etc.

Let's check the value of 'num' with a preprocessor:
```diff
func InitComponents() {
+	turbo.SetPreprocessor("/eat_apple/{num:[0-9]+}", preEatApple)
}

+func preEatApple(resp http.ResponseWriter, req *http.Request) error {
+	num,err := strconv.Atoi(req.Form["num"][0])
+	if err!=nil {
+		resp.Write([]byte("'num' is not numberic"))
+		return errors.New("invalid num")
+	}
+	if num > 5 {
+		resp.Write([]byte("Too many apples!"))
+		return errors.New("Too many apples")
+	}
+	return nil
+}
```
As usual, restart HTTP server, and test:
```sh
$ curl -w "\n" "http://localhost:8081/eat_apple/5"
message:"Good taste! Apple num=5"
$ curl -w "\n" "http://localhost:8081/eat_apple/6"
Too many apples!
```
### Postprocessor
By default, RPC response objects are format into a JSON string, and returned as API response.  
Postprocessors handle responses from backend service. You can change default behavior by assigning a postprocessor.

Let's change the response of API "/eat_apple/{num:[0-9]+}":  
Edit "yourservice/grpcapi/component/components.go":
```diff
func InitComponents() {
+	turbo.SetPostprocessor("/eat_apple/{num:[0-9]+}", postEatApple)
}

+func postEatApple(resp http.ResponseWriter, req *http.Request, serviceResp interface{}) {
+	sr := serviceResp.(*proto.EatAppleResponse)
+	resp.Write([]byte("this is from postprocesser, message=" + sr.Message))
+}
```
Restart HTTP server and test:
```sh
$ curl -w "\n" "http://localhost:8081/eat_apple/5"
this is from postprocesser, message=Good taste! Apple num=5
```

## <a name="hijacker"></a>Hijacker
Hijackers are similar with preprocessors. The difference is, hijackers hijack the whole mapping process.  
If a hijacker is assigned to an URL, it will take over the process between the last Before() and the first After() function.  
You can do everything, which means you also have to call gRPC method yourself.

In this example, URL "/eat_apple/{num:[0-9]+}" is hijacked, no matter what the value is in query string,
the value of parameter "num" is set to "999".
```diff
func InitComponents() {
+	turbo.SetHijacker("/eat_apple/{num:[0-9]+}", hijackEatApple)
}

+func hijackEatApple(resp http.ResponseWriter, req *http.Request) {
+	client := turbo.GrpcService().(gen.YourServiceClient)
+	r := new(gen.EatAppleRequest)
+	r.Num = "999"
+	res, err := client.EatApple(req.Context(), r)
+	if err == nil {
+		resp.Write([]byte(res.String() + "\n"))
+	} else {
+		resp.Write([]byte(err.Error() + "\n"))
+	}
+}
```
Restart and test:
```sh
$ curl -w "\n" "http://localhost:8081/eat_apple/6"
message:"Good taste! Apple num=999"
```

## <a name="message_field_convertor"></a> MessageFieldConvertor
Turbo automatically finds from URL route, query string and context.Context, and sets value into a request by struct field name.  
Turbo also gives you a chance to manually construct a struct.  
Edit "yourservice/grpcapi/component/components.go":
```diff
func InitComponents() {
+	turbo.RegisterMessageFieldConvertor(new(proto.CommonValues), convertCommonValues)
}

+func convertCommonValues(req *http.Request) reflect.Value {
+	result := &proto.CommonValues{}
+	result.SomeId = 123456789
+	return reflect.ValueOf(result)
+}
```
OK, func "convertCommonValues" is registered on type "proto.CommonValues" and "SomeId" is changed into "123456789".  
Restart and test:
```sh
$ curl -w "\n" -X GET "http://localhost:8081/hello?your_name=Alice&some_id=123"
{"message":"[grpc server]Hello, Alice, someId=123456789"}
```
## <a name="error_handler"></a> Error Handler
By default, a HTTP code 500 error is returned if any error occurred.  
You can customize this behavior via ErrorHandler:
```diff
func InitComponents() {
+	turbo.WithErrorHandler(errorHandler)
}

+func errorHandler(resp http.ResponseWriter, req *http.Request, err error) {
+  	resp.Write([]byte("from errorHandler:" + err.Error()))
+}
```
Restart and test(Modify "SayHello" to make it return an error):
```sh
$ curl -w "\n" "http://localhost:8081/hello?your_name=zx"
from errorHandler:rpc error: code = Unknown desc = error!
```
## <a name="support_thrift"></a> Thrift support
Turbo supports thrift as well.  
Similar with grpc, you can create a thrift project or generate thrift codes with "-r thrift" in command line.  
Just change "grpc" into "thrift" when you want to do something in thrift.
## <a name="service_yaml"></a> Configs in service.yaml
```yaml
config:
# [Optional] The runtime environment,
# Default: development
# Values: production | development
  environment: production
# [Optional]The Path to which turbo logs, can be absolute or relative,
# Default: log
  turbo_log_path: log
# The port http server listens
  http_port: 8081

# The grpc service name, MUST be a CamelCase name, usually end with "Service"
  grpc_service_name: YourService

# The grpc server entry point
  grpc_service_address: 127.0.0.1:50051

# The thrift service name, MUST be a CamelCase name, usually end with "Service"
  thrift_service_name: YourService

# The thrift server entry point
  thrift_service_address: 127.0.0.1:50052

# Only valid for grpc service.
# By default, the Response message is marshaled by jsonpb.Marshaler, and returned directly.
# There're some protobuf "problem" with this json:
# (a) protobuf parse int64 as string: e.g. {"int64_value":"123"}
# (b) a Key with a nil Ptr value is missing in the json.
# If this option is set to "true", then Turbo will change the json by:
# 1, if struct field type is 'int64', then change the value in Json into a number
# 2, if field type is 'Ptr', and field value is 'nil', then set "[key_name]":null in Json
# 3, if any key in json is missing, set zero value to that key
# Notice: 'map' is not filtered (yet).
  filter_proto_json: true
  
# Valid only if "filter_proto_json: true",
# Default value: true
# If this option is set to "true", protobuf message fields with zero values will show in Json.
# As [Golang spec](https://golang.org/ref/spec#The_zero_value) says, zero values are 
# "false for booleans, 0 for integers, 0.0 for floats, "" for strings, 
# and nil for pointers, functions, interfaces, slices, channels, and maps".
  filter_proto_json_emit_zerovalues: true
  
# Valid only if "filter_proto_json: true",
# Default value: true
# If this option is set to "true", int64 values will be shown as number in Json, instead of string
  filter_proto_json_int64_as_number: true

# This mapping is the core function of Turbo.
# This mapping tells Turbo how to proxy a HTTP request to a grpc/thrift entry point.
# The format is "HTTP_METHOD URL SERVICE_METHOD_NAME".
# Trubo use that awsome [gorilla mux](github.com/gorilla/mux) as router, it also support variables in URL.
urlmapping:
  - GET,POST /hello SayHello
  - GET /eat_apple/{num:[0-9]+} EatApple
```
