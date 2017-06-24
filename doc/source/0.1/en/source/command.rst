.. _command:

Command line tools
==================

turbo create package_path ServiceName -r (grpc|thrift)
------------------------------------------------------

'turbo create' creates a project with runnable HTTP server and gRPC/Thrift server.

'ServiceName' **MUST** be a CamelCase string.

Project structure::

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

'turbo generate' generates switcher.go and [service_name].pb.go to 'gen' directory.   

This command is useful when either service.yaml or [service_name].proto|.thrift is changed. 
For example, add a new API, change an existing API, change url-grpc mapping, etc.
Example::

 $ turbo generate package/path/to/yourservice -r grpc 
  -I $GOPATH/src/package/path/to/yourservice -I $GOPATH/src/shared

"-I" can appear more than one time, if you have a shared file like "shared.proto" imported from other path.

