namespace go gen
include "shared.thrift"

struct SayHelloResponse {
  1: string message,
}

struct EatAppleResponse {
  1: string message,
}

service YourService {
    SayHelloResponse sayHello (1:string yourName, 2:shared.CommonValues values, 3:shared.HelloValues helloValues)
    EatAppleResponse eatApple (1:i32 num, 2:string stringValue, 3:bool boolValue)
}
