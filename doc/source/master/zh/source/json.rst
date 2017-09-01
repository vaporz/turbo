.. _json:

RESTFULL JSON API
=================

GRPC
----

举一个例子，接口定义如下：

.. code-block:: proto

 message SayHelloRequest {
     string stringValue = 1;
     int64 int64Value = 2;
 }

 message SayHelloResponse {
     string message = 1;
 }

 service TestService {
     rpc sayHello (SayHelloRequest) returns (SayHelloResponse) {}
 }

protobuf生成的struct代码为：

.. code-block:: go

 type SayHelloRequest struct {
 	StringValue  string        `protobuf:"bytes,2,opt,name=stringValue" json:"stringValue,omitempty"`
 	Int64Value   int64         `protobuf:"varint,3,opt,name=int64Value" json:"int64Value,omitempty"`
 }

参考 :ref:`如何添加新API <add>`，实现该接口。

接下来启动服务，发起POST请求（Content-Type=application／json），body数据为：

.. code-block:: json

 {
 	"stringValue":"a string value",
	"int64Value":1234
 }

**注意**： json中的key的名字， **必须** 与protobuf生成struct中对应变量的 tag "protobuf"里的name参数的值相同，比如"stringValue"。

在你实现的grpc server端接口中，应该能够读取到body中的数据。

Thrift
------

举相同的例子：

.. code-block:: thrift

 struct SayHelloRequest {
   1: string stringValue,
   2: i32 int32Value,
   3: bool boolValue,
 }

 struct SayHelloResponse {
   1: string message,
 }

 service TestService {
     SayHelloResponse testJson (1:SayHelloRequest request)
 }

thrift生成的struct代码：

.. code-block:: go

 type TestJsonRequest struct {
   StringValue string `thrift:"stringValue,1" db:"stringValue" json:"stringValue"`
   Int32Value int32 `thrift:"int32Value,2" db:"int32Value" json:"int32Value"`
   BoolValue bool `thrift:"boolValue,3" db:"boolValue" json:"boolValue"`
 }

参考 :ref:`如何添加新API <add>`，实现该接口。

接下来启动服务，发起POST请求（Content-Type=application／json），body数据为：

.. code-block:: json

 {
 	"StringValue":"a string value",
 	"int32Value":1234,
	"boolvalue":true
 }

**注意**： 与grpc不同，通过thrift发出的json请求数据的key是大小写不敏感的。

在你实现的thrift server端接口中，应该能够读取到body中的数据。
