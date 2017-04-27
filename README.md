# turbo
Turbo generates a reverse-proxy server which translates a HTTP request into a grpc request.

## How to run the example
1, 
```sh
git clone https://github.com/vaporz/turbo.git
```
2,
```sh
cd turbo/example/inventoryservice
```
3, start grpc service
```sh
go run service/inventoryservice.go
```
4, start HTTP server
```sh
go run ui_api.go -p turbo/example/inventoryservice
```
Now you can make a request:
```sh
curl "http://localhost:8081/videos?network_id=122222"
```
you should have seen the response:
```sh
videos:<id:111 name:"test video" > videos:<id:222 name:"test video222" >
```
