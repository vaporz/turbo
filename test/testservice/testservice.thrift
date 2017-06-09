namespace go gen

struct SayHelloResponse {
  1: string message,
}

service TestService {
    SayHelloResponse sayHello (1:string yourName)
}
