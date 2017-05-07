namespace go gen

struct SayHelloResponse {
  1: string message,
}

struct EatAppleResponse {
  1: string message,
}

service YourService {
    SayHelloResponse sayHello (1:string yourName)
    EatAppleResponse eatApple (1:i32 num, 2:string stringValue, 3:bool boolValue)
}
