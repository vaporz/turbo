.. _config:

配置文件service.yaml
=====================

.. code-block:: yaml

 config:
 # [可选] 运行环境类型,
 # 默认值: development
 # 合法值: production | development
   environment: production
 # [可选] Turbo记录log的位置，可以是绝对路径，也可以是相对路径
 # 默认值: log
   turbo_log_path: log
 # http server监听的端口号
   http_port: 8081
 
 # grpc服务名字，必须是驼峰格式（CamelCase），通常用"Service"结尾
   grpc_service_name: YourService
 
 # grpc server使用的地址和端口
   grpc_service_address: 127.0.0.1:50051
 
 # thrift服务名字，必须是驼峰格式（CamelCase），通常用"Service"结尾
   thrift_service_name: YourService
 
 # thrift server使用的地址和端口
   thrift_service_address: 127.0.0.1:50052
 
 # 只对grpc服务生效。
 # 默认配置下，service端返回的Message对象是用jsonpb.Marshaler反序列化并返回的。
 # 但protobuf生成的这个json有几个问题：
 # (a) int64被输出成了字符串类型，比如 {"int64_value":"123"}
 # (b) 指针类型的字段，如果值是nil，生成的json里会没有这个字段（最新版本的protobuf已经修复了这个问题:https://github.com/golang/protobuf/issues/367）。
 # 如果这个配置项被设为true，那么Turbo就会干预json的生成过程：
 # 1, 如果struct字段是一个 'int64'，那就把它按数字格式输出，
 # 2, 如果字段类型是指针（'Ptr'），并且值为nil，那就在json里加上 "[key_name]":null，
 # 3, 如果某个字段在json中不存在，那就用零值（zero value）赋值并加入到json中。
 # 注意：map类型没有被Turbo处理.
   filter_proto_json: true
   
 # 只有在"filter_proto_json: true"时才起作用，
 # 默认值: true
 # 如果被设为true，Protobuf Message中值为零值（zero value）的字段会出现在json中。
 # [Golang语言规范](https://golang.org/ref/spec#The_zero_value) 中对"零值"的定义是
 # "false for booleans, 0 for integers, 0.0 for floats, "" for strings, 
 # and nil for pointers, functions, interfaces, slices, channels, and maps".
   filter_proto_json_emit_zerovalues: true
   
 # 只有在"filter_proto_json: true"时才起作用，
 # 默认值: true
 # 如果被设为true，int64类型的值会作为数字类型在json中输出，而不是字符串类型
   filter_proto_json_int64_as_number: true
 
 # 这个映射是Turbo最核心的配置！
 # 这项配置告诉Turbo怎么把一个HTTP请求代理到后端的service方法上。
 # 配置的格式是 "HTTP_METHOD URL SERVICE_METHOD_NAME".
 # Turbo是借助于好用的 [gorilla mux](github.com/gorilla/mux) 来实现这个功能的，它还支持获取URL里的参数。
 urlmapping:
   - GET,POST /hello SayHello
   - GET /eat_apple/{num:[0-9]+} EatApple
  
