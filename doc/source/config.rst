.. _config:

Configs in service.yaml
=======================

.. code-block:: yaml

 config:
 # [Optional] The runtime environment,
 # Default: development
 # Values: production | development
   environment: production
 # [Optional]The Path to which turbo logs, can be absolute or relative,
 # Default: log
   turbo_log_path: log
 # The port http server listens
   http_port: 8081
 
 # The grpc service name, MUST be a CamelCase name, usually end with "Service"
   grpc_service_name: YourService
 
 # The grpc server entry point
   grpc_service_address: 127.0.0.1:50051
 
 # The thrift service name, MUST be a CamelCase name, usually end with "Service"
   thrift_service_name: YourService
 
 # The thrift server entry point
   thrift_service_address: 127.0.0.1:50052
 
 # Only valid for grpc service.
 # By default, the Response message is marshaled by jsonpb.Marshaler, and returned directly.
 # There're some protobuf "problem" with this json:
 # (a) protobuf parse int64 as string: e.g. {"int64_value":"123"}
 # (b) a Key with a nil Ptr value is missing in the json.
 # If this option is set to "true", then Turbo will change the json by:
 # 1, if struct field type is 'int64', then change the value in Json into a number
 # 2, if field type is 'Ptr', and field value is 'nil', then set "[key_name]":null in Json
 # 3, if any key in json is missing, set zero value to that key
 # Notice: 'map' is not filtered (yet).
   filter_proto_json: true
   
 # Valid only if "filter_proto_json: true",
 # Default value: true
 # If this option is set to "true", protobuf message fields with zero values will show in Json.
 # As [Golang spec](https://golang.org/ref/spec#The_zero_value) says, zero values are 
 # "false for booleans, 0 for integers, 0.0 for floats, "" for strings, 
 # and nil for pointers, functions, interfaces, slices, channels, and maps".
   filter_proto_json_emit_zerovalues: true
   
 # Valid only if "filter_proto_json: true",
 # Default value: true
 # If this option is set to "true", int64 values will be shown as number in Json, instead of string
   filter_proto_json_int64_as_number: true
 
 # This mapping is the core function of Turbo.
 # This mapping tells Turbo how to proxy a HTTP request to a grpc/thrift entry point.
 # The format is "HTTP_METHOD URL SERVICE_METHOD_NAME".
 # Trubo use that awsome [gorilla mux](github.com/gorilla/mux) as router, it also support variables in URL.
 urlmapping:
   - GET,POST /hello SayHello
   - GET /eat_apple/{num:[0-9]+} EatApple
  
