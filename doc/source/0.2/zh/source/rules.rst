.. _rules:

规则和约定
==========

使用Turbo时，有一些约定需要注意。

*    定义grpc服务的接口时，如果方法名是"methodName"，那么request和response对象的名字 **必须** 是 "MethodNameRequest" 和 "MethodNameResponse"。
*    定义thrift服务的接口时，如果方法名是"methodName"，那么response对象的名字 **必须** 是 "MethodNameResponse"。
*    如果$GOPATH中设置了多个path，Turbo会用其中的第一个path作为GOPATH。
*    处理request参数时，URL路径中的参数的优先级更高，高于queryString，body或者context.Context中的参数值。
    比如，对于请求"GET /book/1234?id=5678"，"1234"和"5678"都是参数"id"的值，但"1234"会被选中。
*    对相同名字的参数，key全部为小写的参数的优先级，高于包含大写字母的参数，
    比如，对于请求"GET /book?id=1234&ID=5678", "1234"会被选中。
*    在Turbo中，参数的名字是大小写不敏感的，实际上，在内部，Turbo会先把所有参数的名字都转换为小写格式，然后再继续使用。
    比如，对于请求"GET /book?ID=1234"，queryString实际等价于"?id=1234"。
