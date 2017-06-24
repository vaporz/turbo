.. _shared:

如何使用公共IDL文件
===================

有时，我们希望把一些所有请求都会用到的数据包含在发送给service的request中。
于是，我们会项目外的另一个地方新建一个公共的文件，比如"shared.proto"，然后定义一个新的Message。
然后，再把这个新Message添加到其他request对象中。

下面的例子介绍了如何在Turbo中这么做。

1, 新建 "shared.proto"，并且定义一个新Message
----------------------------------------------

创建文件夹"common"（或者其他名字）::

 $ mkdir -p $GOPATH/src/package/path/to/common

在里面创建文件 "shared.proto":

.. code-block:: proto

 syntax = "proto3";
 package proto;
 
 message CommonValues {
     int64 someId = 1;
 }

2, 给request对象添加新参数
---------------------------

编辑 "yourservice.proto":

.. code-block:: diff

 syntax = "proto3";
 package proto;
 +
 +import "shared.proto";
 
 message SayHelloRequest {
     string yourName = 1;
 +    CommonValues values = 2;
 }

3, 生成代码
------------

::

 $ turbo generate package/path/to/yourservice -r grpc -I $GOPATH/src/package/path/to/yourservice
  -I $GOPATH/src/package/path/to/common

完成！

测试前，编辑 "youserviceimpl.go"，返回 "someId":

.. code-block:: diff

 func (s *YourService) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
 +	someId := strconv.FormatInt(req.Values.SomeId, 10)
 -	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName}, nil
 +	return &proto.SayHelloResponse{Message: "[grpc server]Hello, " + req.YourName + ", someId=" + someId}, nil
 }

重启服务并测试::

 $ curl -w "\n" -X GET "http://localhost:8081/hello?your_name=Alice&some_id=12345"
 {"message":"[grpc server]Hello, Alice, someId=12345"}

Turbo很智能，queryString中的参数"some_id"会被自动设置给"SayHelloRequest.CommonValues.SomeId"。

除了queryString，也可以读取URL路径里的参数，或者在拦截器中设置在context.Context里的参数。
