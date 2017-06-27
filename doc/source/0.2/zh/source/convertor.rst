.. _convertor:

Message类型转换器
==================

Turbo 会自动在URL，queryString和context.Context中查询参数，然后根据struct中的字段名设置到request对象中。

Turbo 也允许你"手动"组装一个struct对象，举个例子：
编辑 "yourservice/grpcapi/component/components.go":

.. code-block:: diff

 func InitComponents() {
 +	turbo.RegisterMessageFieldConvertor(new(proto.CommonValues), convertCommonValues)
 }
 
 +func convertCommonValues(req *http.Request) reflect.Value {
 +	result := &proto.CommonValues{}
 +	result.SomeId = 123456789
 +	return reflect.ValueOf(result)
 +}

OK了, 方法 "convertCommonValues" 被注册给了类型 "proto.CommonValues"，在方法里面，"SomeId"被修改为"123456789"，不管之前是什么值。

重启服务然后测试::

 $ curl -w "\n" -X GET "http://localhost:8081/hello?your_name=Alice&some_id=123"
 {"message":"[grpc server]Hello, Alice, someId=123456789"}

