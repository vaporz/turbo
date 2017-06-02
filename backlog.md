1, What is a Go-style framework?  
2, Unit test, performance test(vs wheels), test test test.
3, Logging: log level, log configuration, there should be some nice tools.  
4, Better documentation.  
5, Learn from and integrate with other open source projects, such as go-kit, go-micro, etc.  
6, Those TODOs among codes...  
7, proto3 issues:
 * keys with a "nil" ptr value are missing in JSON:
   https://github.com/google/protobuf/issues/3132
 * int64 values are parsed as strings, not number

8, Request param value validations with protobuf field options(thrift?).  
9, Start both http and grpc server as go routine.  
10, Generate a const list of registered urls.  
11, Reload config at runtime.


update readme for starter.go change  
modify generator.go for new main.go  
error handling mechanism  
