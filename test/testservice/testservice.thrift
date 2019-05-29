namespace go services
include "shared.thrift"

struct SayHelloResponse {
  1: string message,
}

struct TestJsonRequest {
  1: string stringValue,
  2: i32 int32Value,
  3: bool boolValue,
}

struct TestJsonResponse {
  1: string message,
}

service TestService {
    SayHelloResponse sayHello (1:shared.CommonValues values, 2:string yourName, 3:i64 int64Value, 4:bool boolValue,
     5:double float64Value, 6:i64 uint64Value, 7:i32 int32Value, 8:i16 int16Value, 9:list<string> stringList,
     10:list<i32> i32List, 11:list<bool> boolList, 12:list<double> doubleList)

    TestJsonResponse testJson (1:TestJsonRequest request)
}

struct EatResponse {
  1: string message,
}

service MinionsService {
    EatResponse Eat (1:string food)
}
