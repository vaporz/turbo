.. _add:

如何添加一个新API
=================

1, 定义一个gRPC API
-------------------

修改 "yourservice.proto", 添加新方法 "eatApple":

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

2, 添加一对儿url-gRPC映射
-------------------------

修改 "service.yaml":

.. code-block:: diff

 urlmapping:
    - GET /hello SayHello
 +  - GET /eat_apple/{num:[0-9]+} EatApple

3, 生成代码
-----------

::

 $ turbo generate package/path/to/yourservice -r grpc -I $GOPATH/src/package/path/to/yourservice

4, 实现新的gRPC方法
-------------------

编辑 "grpcservice/impl/yourserviceimpl.go":

.. code-block:: diff

 +func (s *YourService) EatApple(ctx context.Context, req *proto.EatAppleRequest) (*proto.EatAppleResponse, error) {
 +	return &proto.EatAppleResponse{Message: "Good taste! Apple num=" + strconv.FormatInt(int64(req.Num), 10)}, nil
 +}


然后，重启HTTP和gRPC的server，然后测试::

 # start grpc service
 $ go run grpcservice/yourservice.go
 # start http server
 $ go run grpcapi/yourserviceapi.go
 # test
 $ curl "http://localhost:8081/eat_apple/5"
 message:"Good taste! Apple num=5"

