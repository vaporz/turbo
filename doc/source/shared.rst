.. _shared:

Use a shared struct
===================

Sometimes we want to add some info to all requests from frontend server to backend server.
So we define a new message in a file like "shared.proto" in a separate path.
Then we add this new message to other message as a struct field.

Let me show you how to do this in grpc with Turbo.

1, Create "shared.proto" and define a new message
-------------------------------------------------

Create folder "common"(or any other name)::

 $ mkdir -p $GOPATH/src/package/path/to/common

Create file "shared.proto" in folder "common":

.. code-block:: proto

 syntax = "proto3";
 package proto;
 
 message CommonValues {
     int64 someId = 1;
 }

2, Add new field to a request message
-------------------------------------

Edit "yourservice.proto":

.. code-block:: diff

 syntax = "proto3";
 package proto;
 +
 +import "shared.proto";
 
 message SayHelloRequest {
     string yourName = 1;
 +    CommonValues values = 2;
 }

3, Generate codes
-----------------

::

 $ turbo generate package/path/to/yourservice -r grpc -I $GOPATH/src/package/path/to/yourservice
  -I $GOPATH/src/package/path/to/common

Done!

Before test, edit youserviceimpl.go to return "someId":

.. code-block:: diff

 func (s *YourService) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
 +	someId := strconv.FormatInt(req.Values.SomeId, 10)
 -	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
 +	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName + ", someId=" + someId}, nil
 }

Restart grpc server and test::

 $ curl -w "\n" -X GET "http://localhost:8081/hello?your_name=Alice&some_id=12345"
 {"message":"[grpc server]Hello, Alice, someId=12345"}

Turbo is smart, query string param "some_id" is mapped into SayHelloRequest.CommonValues.SomeId automatically.

Besides query string, You can also map "SomeId" from URL route params, or context.Context which is set from interceptors.

