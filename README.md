# Turbo  [![Circle CI](https://circleci.com/gh/vaporz/turbo.svg?style=shield)](https://circleci.com/gh/vaporz/turbo) [![Build Status](https://travis-ci.org/vaporz/turbo.svg?branch=master)](https://travis-ci.org/vaporz/turbo) [![Go Report Card](https://goreportcard.com/badge/github.com/vaporz/turbo)](https://goreportcard.com/report/github.com/vaporz/turbo) [![codebeat badge](https://codebeat.co/badges/7a166e48-dae1-454c-b925-4fbcd3f1f461)](https://codebeat.co/projects/github-com-vaporz-turbo-master) [![Coverage Status](https://coveralls.io/repos/github/vaporz/turbo/badge.svg?branch=master)](https://coveralls.io/github/vaporz/turbo?branch=master)

<b>WORK IN PROGRESS! There may be many bugs, and the README may not be synced in time as the codes changed.</b>

## Features
 * Turbo generates a reverse-proxy server which translates a HTTP request into a grpc request.
 * [Interceptor](#interceptor).
 * [PreProcessor and Hijacker](#preprocessor_and_hijacker): customizable url-grpc mapping process.
 * Support gRPC and thrift.

## Create a service on the fly
### 0, Before the start
Obviously, you have to install [Golang](https://golang.org) and [Protocol buffers](https://developers.google.com/protocol-buffers/) first.<br>
And install required packages.
```sh
go get github.com/kylelemons/go-gypsy/yaml
go get github.com/gorilla/mux
go get google.golang.org/grpc
go get git.apache.org/thrift.git/lib/go/thrift
go get github.com/spf13/cobra
go get github.com/spf13/viper
```

### 1, Install Turbo command line tools
```sh
git clone https://github.com/vaporz/turbo.git
cd turbo/turbo
go install
```

### 2, Create your service
```sh
$ turbo create package/path/to/yourservice YourService grpc
```
Directory "$GOPATH/src/package/path/to/yourservice" should appear.<br>
There're also some codes in this folder, similar with this example:<br>
https://github.com/vaporz/turbo/tree/master/example/yourservice

### 3, Run
That's it! Now let's Play!

Start gRPC server and HTTP server:
```sh
cd $GOPATH/src/yourservice
go run service/yourservice.go
go run yourserviceapi.go
```
Send a request:
```sh
$ curl "http://localhost:8081/hello?your_name=Alice"
message:"Hello, Alice"
```
## Rules and Conventions
There are some rules when you use turbo.
 * When defining a gRPC service, if the name of a gRPC method is "methodName", then the name
 of request message and response message MUST be "MethodNameRequest" and "MethodNameResponse".
 * If multiple paths are assigned to $GOPATH(e.g. devided by ':'), then the last path is used by turbo as GOPATH.
 * When parsing request parameters, values from URL path has a higher priority than those from query string or body.
<br> e.g. In a request like "GET /book/1234?id=5678", both "1234" and "5678" are values to "id", but "1234" is picked
 as value to key "id".
 * The value of a key with all lower case characters has a higher priority to the value of a key with upper case
 characters.<br>
 e.g. In a request like "GET /book?id=1234&Id=5678", "1234" is used for key "id".
 * A parameter's key is case-insensitive to turbo, in fact, internally turbo will cast keys to
 lower case characters before further use.<br>
 e.g. In a request like "GET /book?ID=1234", turbo will see this query string as "id=1234".

## Command line tools
### turbo create [package_name] [service_name] (grpc/thrift)
'turbo create' creates a project with runnable HTTP server and gRPC/Thrift server.<br>
Project structure:
```sh
yourservice
├── gen
│   ├── switcher.go
│   └── yourservice.pb.go
├── service
│   ├── impl
│   │    └── yourserviceimpl.go
│   └── yourservice.go
├── service.yaml
├── yourservice.proto
└── yourserviceapi.go
```
### turbo generate [package_name]  (grpc/thrift)
'turbo generate' generates switcher.go and [service_name].pb.go to 'gen' directory.<br>
This command is useful when either service.yaml or [service_name].proto is changed.<br>
For example, add a new API, change an existing API, change url-grpc mapping, etc.

## How to add a new API
#### 1, Define new gRPC API
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
#### 2, Add new url-grpc mapping
Modify "service.yaml":
```diff
 urlmapping:
   - GET /hello SayHello
+  - GET /eat_apple/{num:[0-9]+} EatApple
```
#### 3, Generate new codes
```sh
turbo generate package/path/to/yourservice
```

#### 4, Implement new gRPC method
Modify "yourserviceimpl.go":
```go
func (s *YourService) EatApple(ctx context.Context, req *gen.EatAppleRequest) (*gen.EatAppleResponse, error) {
	return &gen.EatAppleResponse{Message: "Good taste! Apple num=" + req.Num}, nil
}
```

Now, stop and restart both HTTP and gRPC server, then test:
```sh
$ go run service/yourservice.go
$ go run yourserviceapi.go
$ curl "http://localhost:8081/eat_apple/5"
message:"Good taste! Apple num=5"
```

## <a name="interceptro"></a>Interceptor
Interceptors provide hook functions which run before or after a request.<br>
Interceptors can be assigned to
 * 1, All URLs
 * 2, An URL path (which means a group of URLs)
 * 3, One URL
 * 4, One URL on HTTP methods

The more precise it is, the higher priority it has.<br>
If interceptor A is assigned to URL '/abc' on HTTP method "GET", and interceptor B is assigned to all URLs,
 then A is executed when "GET /abc", and B is executed when "POST /abc".

Now, let's create an interceptor for URL "/eat_apple/{num:[0-9]+}" to log some info before and after a request.<br>
Edit "yourservice/interceptor/log.go":
```go
package interceptor

import (
	"turbo"
	"log"
	"net/http"
)

type LogInterceptor struct {
	// optional, BaseInterceptor allows you to create an interceptor which implements
	// Before() or After() only, or none of them.
	// If you were to implement both, you can remove this line.
	turbo.BaseInterceptor
}

func (l LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	log.Println("[Before] Request URL:"+req.URL.Path)
	return nil
}

func (l LogInterceptor) After(resp http.ResponseWriter, req *http.Request) error {
	log.Println("[After] Request URL:"+req.URL.Path)
	return nil
}
```
Then assign this interceptor to URL "/eat_apple/{num:[0-9]+}":<br>
Edit "yourservice/yourserviceapi.go":
```diff
 func main() {
+	turbo.Intercept("/eat_apple/{num:[0-9]+}", i.LogInterceptor{})
 	turbo.StartGrpcHTTPServer("turbo/example/yourservice", grpcClient, gen.Switcher)
 }
```
Lastly, restart HTTP server and test:
```sh
$ curl "http://localhost:8081/eat_apple/5"
message:"Good taste! Apple num=5"
```
Check the server's console:
```sh
$ go run yourservice/yourserviceapi.go
2017/05/04 16:47:39 [Before] Request URL:/eat_apple/5
2017/05/04 16:47:39 [After] Request URL:/eat_apple/5
```

We usually do something like validations in an interceptor, when the validation fails,
the request halts, and returns an error message.<br>
To do this, you can simply return an error:
```go
func (l LogInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	resp.Write([]byte("Encounter an error from LogInterceptor!\n"))
	return errors.New("error!")
}
```
Test:
```sh
$ curl "http://localhost:8081/eat_apple/5"
Encounter an error from LogInterceptor!
```
## <a name="preprocessor_and_hijacker"></a>Preprocessor and Hijacker
What if I want to do something particularly for some API?<br>
Preprocessor/Hijacker comes to help!<br>
If both Preprocessors and hijackers are assigned to an URL, only the last hijacker assigned is active.

#### Preprocessor
Preprocessors are executed just after all Before() functions from interceptors, and before
sending requests to gRPC server.<br>
Preprocessors can be used to do something particularly for an API. For example, parameter value validations,
setting default values, parsing values, logging, etc.

Let's check the value of 'num' with a preprocessor:
```diff
 func main() {
 	turbo.Intercept("/eat_apple/{num:[0-9]+}", i.LogInterceptor{})
+	turbo.SetPreprocessor("/eat_apple/{num:[0-9]+}", checkNum)
 	turbo.StartGrpcHTTPServer("turbo/example/yourservice", grpcClient, gen.Switcher)
 }

+func checkNum(resp http.ResponseWriter, req *http.Request) error {
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
$ curl "http://localhost:8081/eat_apple/5"
message:"Good taste! Apple num=5"
$ curl "http://localhost:8081/eat_apple/6"
Too many apples!
```
#### Hijacker
Hijackers are similar with preprocessors. The difference is, hijackers hijack the whole mapping process.<br>
If a hijacker is assigned to an URL, it will take over the process between the last Before() and the first After() function.<br>
You can do everything, which means you also have to call gRPC method yourself.

In this example, URL "/eat_apple/{num:[0-9]+}" is hijacked, no matter what the value is in query string,
the value of parameter "num" is set to "999".
```diff
 func main() {
 	turbo.Intercept("/eat_apple/{num:[0-9]+}", i.LogInterceptor{})
 	turbo.SetPreprocessor("/eat_apple/{num:[0-9]+}", checkNum)
+	turbo.SetHijacker("/eat_apple/{num:[0-9]+}", hijackEatApple)
 	turbo.StartGrpcHTTPServer("turbo/example/yourservice", grpcClient, gen.Switcher)
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
$ curl "http://localhost:8081/eat_apple/6"
message:"Good taste! Apple num=999"
```
