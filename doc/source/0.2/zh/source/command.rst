.. _command:

命令行工具
==========

turbo create package_path ServiceName -r (grpc|thrift)
------------------------------------------------------

'turbo create' 用来创建一个新项目，包含了可直接运行的HTTP server和gRPC／Thrfit server。
'ServiceName'**必须**是一个驼峰（CamelCase）字符串。
项目目录结构::

 $ turbo create package/path/to/yourservice YourService -r grpc
 $ cd $GOPATH/src/package/path/to/yourservice
 $ tree
 .
 |-- gen
 |   |-- grpcswitcher.go
 |   `-- proto
 |       `-- yourservice.pb.go
 |-- grpcapi
 |   |-- component
 |   |   `-- components.go
 |   `-- yourserviceapi.go
 |-- grpcservice
 |   |-- impl
 |   |   `-- yourserviceimpl.go
 |   `-- yourservice.go
 |-- main.go
 |-- service.yaml
 `-- yourservice.proto

turbo generate package_path -r (grpc/thrift) -I (absolute_path_to_proto/thrift_files) -I ...
--------------------------------------------------------------------------------------------

'turbo generate' 用来生成 switcher.go 和 [service_name].pb.go，生在文件放在'gen'目录下.

当 service.yaml 或者 [service_name].proto|.thrift有改动时，这个命令很有用，
比如添加了新的API，修改了已有的API定义，修改了url到service接口的映射关系等。
例子::

 $ turbo generate package/path/to/yourservice -r grpc 
  -I $GOPATH/src/package/path/to/yourservice -I $GOPATH/src/shared

"-I" 可以重复多次, 比如你一个公共的IDL文件 "shared.proto" 的话，公共文件的绝对路径可以用"-I"加进来。
