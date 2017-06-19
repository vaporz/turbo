.. _add:

How to add a new API
====================

1, Define new gRPC API
----------------------

Modify "yourservice.proto", add new method "eatApple":

.. code-block:: diff

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

2, Add new url-grpc mapping
---------------------------

Modify "service.yaml":

.. code-block:: diff

 urlmapping:
    - GET /hello SayHello
 +  - GET /eat_apple/{num:[0-9]+} EatApple

3, Generate new codes
---------------------

::

 $ turbo generate package/path/to/yourservice -r grpc -I $GOPATH/src/package/path/to/yourservice

4, Implement new gRPC method
----------------------------

Edit "grpcservice/impl/yourserviceimpl.go":

.. code-block:: diff

 +func (s *YourService) EatApple(ctx context.Context, req *proto.EatAppleRequest) (*proto.EatAppleResponse, error) {
 +	return &proto.EatAppleResponse{Message: "Good taste! Apple num=" + strconv.FormatInt(int64(req.Num), 10)}, nil
 +}


Now, restart both HTTP and gRPC server, then test::

 # start grpc service
 $ go run grpcservice/yourservice.go
 # start http server
 $ go run grpcapi/yourserviceapi.go
 # test
 $ curl "http://localhost:8081/eat_apple/5"
 message:"Good taste! Apple num=5"

