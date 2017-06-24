.. _convertor:

Message Field Convertor
=======================

Turbo automatically finds from URL route, query string and context.Context, and sets value into a request by struct field name.

Turbo also gives you a chance to manually construct a struct.

Edit "yourservice/grpcapi/component/components.go":

.. code-block:: diff

 func InitComponents() {
 +	turbo.RegisterMessageFieldConvertor(new(proto.CommonValues), convertCommonValues)
 }
 
 +func convertCommonValues(req *http.Request) reflect.Value {
 +	result := &proto.CommonValues{}
 +	result.SomeId = 123456789
 +	return reflect.ValueOf(result)
 +}

OK, func "convertCommonValues" is registered on type "proto.CommonValues" and "SomeId" is changed into "123456789".

Restart and test::

 $ curl -w "\n" -X GET "http://localhost:8081/hello?your_name=Alice&some_id=123"
 {"message":"[grpc server]Hello, Alice, someId=123456789"}

