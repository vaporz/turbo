# 参数绑定

这篇讲 turbo 如何把一个 HTTP 请求填进 gRPC 请求消息或 Thrift 方法参数：四种来源的优先级、参数名的拼写归一规则、JSON body 与表单/query/path 各自的绑定路径、注入值的用法，以及「参数存在但不能用」时为什么应该报错而不是留零值。

## 优先级：injected > path > body > query/form

`binding.go` 开头的注释把规则写死成一行：

```
injected > path > body > query/form
```

| 来源 | 怎么进来 | 为什么排在这个位置 |
| --- | --- | --- |
| injected | `turbo.InjectParam` | 唯一不是客户端给的来源：验签通过、设备码解析完成、用户已认证之后由服务端写下 |
| path | 路由捕获的变量 | 路由已经用这个值匹配过了 |
| body | JSON 请求体 | body 明确提到的字段优先于 URL 上的同名参数 |
| query/form | `req.Form` | 只补 body 没有提到的字段 |

按请求类型展开：非 JSON 请求走 `BuildStructErr`，逐字段调用 `findValue`，顺序是 injected、path、query/form；JSON 请求走 `BuildRequest`，顺序是 `jsonpb` 填 body、`bindJSONGaps` 用 query/form 补空缺、`setPathParams` 用 path 覆盖、`bindJSONInjected` 用注入值覆盖全部。

## 拼写归一：normalizeKey 与 lookupKeys

参数名的拼写不影响绑定。`normalizeKey` 只忽略大小写和下划线：

```go
func normalizeKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(key, "_", ""))
}
```

`deviceCode`、`device_code`、`DeviceCode`、`DEVICE_CODE` 都归一到 `devicecode`；连字符不忽略。

`lookupKeys` 给出「一个字段名会用哪些拼写去探测」：`[]string{fieldName, strings.ToLower(fieldName), ToSnakeCase(fieldName)}`。例如 `Int64Value` 得到 `Int64Value`、`int64value`、`int64_value`。`ToSnakeCase` 在 `util.go` 里用正则实现，`YourName` 到 `your_name`，`Int64Value` 到 `int64_value`。

## 每个来源的查找顺序

### query / form：formValue

`formValue` 先按 `lookupKeys` 的三种拼写精确查 `req.Form`，都没命中再做一次拼写无关的扫描（`normalisedFormValue`）。第二次扫描会先把 key 排序，结果不依赖 map 的遍历顺序。

这个「先精确、后归一」的顺序产生一个可观察的后果：两个 key 归一到同一个参数时，**字段名的小写形式优先**。对字段 `YourName`，`lookupKeys` 里 `yourname` 排在 `your_name` 前面，所以 `?your_name=turbo&yourname=xxx` 绑定到 `xxx`，与两个 key 出现的先后无关。

### path：pathValue 与 findPathParamValue

`pathValue` 从 `mux.Vars(req)` 读路由变量，`findPathParamValue` 同样先按 `lookupKeys` 精确匹配，再做一次 `normalizeKey` 扫描，所以路由写成 `{your_Name}` 也能填进 `YourName` 字段。

`parseRequestForm`（`util.go`）会把路由变量拷进 `req.Form`，并且放在同 key 已有值的前面；它不修改 `mux.Vars` 本身。

### injected：InjectedValue

`InjectedValue(fieldName, req)` 先查 `InjectParam` 建的存储，再查请求 context 里的普通字符串值（这是 `InjectParam` 出现之前唯一的机制，所以继续支持）。两种存储都按 `lookupKeys` 探测，空字符串视为没有值。

### findValue 把三者串起来

```go
func findValue(fieldName string, req *http.Request) (string, bool) {
	if injected, ok := InjectedValue(fieldName, req); ok {
		if form, ok := formValue(fieldName, req); ok && form != injected {
			log.Warnf("binding: %s <- injected %q, overriding the query/form value %q", fieldName, injected, form)
		}
		return injected, true
	}
	if value, ok := pathValue(fieldName, req); ok {
		return value, true
	}
	return formValue(fieldName, req)
}
```

path 从路由变量读，而不是从 `req.Form` 读，这样「谁赢」由来源决定，不再取决于调用方碰巧用了哪种拼写。

### parseRequestForm 对 body 的处理

`Content-Type` 含 `application/x-www-form-urlencoded` 时，`parseRequestForm` 先把 body 读下来，调用 `req.ParseForm()`，再把 body 复原。所以解析表单不会让原始 body 消失，后面的拦截器和 hijacker 仍然读得到，验签这类逻辑才做得成。其他情况（包括 JSON）根本不碰 body，只把 URL query 装进 `req.Form`。之后 `mergeUpperCaseKeysToLowerCase` 把大写 key 的值并入小写 key，`mergeMuxVars` 把路由变量按小写 key 放进 `req.Form`。

## 注入参数：turbo.InjectParam 与 turbo.InjectedValue

```go
func InjectParam(req *http.Request, key, value string)
func InjectedValue(fieldName string, req *http.Request) (string, bool)
```

为什么用它，源码注释给了三点：

- 它是写 `req.Form` 的受支持替代。`req.Form` 只对表单请求有效，JSON 绑定路径不看 `req.Form`，直接写进去的注入值会无声消失。
- 注入值优先级最高，客户端无法用 body 或 query 覆盖服务端验证过的值。
- 被覆盖时会记一条 warning，便于发现调用方在试图覆盖。

两个使用要点：`InjectParam` 就地修改 `req`（内部是 `*req = *req.WithContext(...)`），所以之后必须继续使用同一个 `*http.Request`；`Interceptor.Before` 只能返回 error，不能返回新的 request，注入必须靠这种就地修改。`key` 为空或 `req` 为 nil 时直接返回，可以重复调用，多个注入值会累积。

### gRPC 场景下为什么必须写进请求消息

turbo 的服务实现位于一次真实的 RPC 之后：HTTP 层通过 switcher 调用 `s.Service(name).(Client).Method(req.Context(), request, ...)`，服务实现拿到的是 gRPC/Thrift 服务端自己构造的 context。放在 `req.Context()` 里的值不会随 RPC 传过去（gRPC 只传 metadata，Thrift 生成的 switcher 甚至直接传 `context.Background()`）。所以注入值要生效，**必须经过绑定写进请求消息的字段**，而 `InjectParam` 正是这么做的。

```go
type DeviceInterceptor struct {
	turbo.BaseInterceptor
}

func (i *DeviceInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	turbo.InjectParam(req, "device_code", "device-from-server")
	return nil
}
```

## JSON body 绑定

`BuildRequest`（`runtime.go`）在 `Content-Type` 含 `application/json` 时用 `&jsonpb.Unmarshaler{AllowUnknownFields: true}` 把 body 解析进请求消息：空 body 视为 `{}`；body 里多出来的字段不会导致失败；解析失败返回 400，消息只报 body 字节数和解析原因（见 10-errors.md）。

解析成功后依次调用 `bindJSONGaps`、`setPathParams`、`bindJSONInjected`。`bindJSONGaps` 用 `jsonObjectKeys` 把 body 解析成「小写 key 到原始 JSON」的映射，`rawHas` 判断 body 是否提到过某个字段（会试 `lookupKeys` 的三种拼写，因为 `jsonpb` 默认按 camelCase 写而 proto 字段名可能是 snake_case），只有 body 没提到时才用 `formValue` 去 query/form 取值。`walkFields` 会递归进入嵌套的消息指针，`rawSub` 取出对应的嵌套 JSON 对象，所以嵌套结构里的字段同样遵循「body 优先、query/form 补缺」。

### filter_proto_json 与两个子选项

这三个键都在 `config:` 下面，作用方向是**响应**（`writeResponse` 构造 `Marshaler`，见 `util.go`）：

| 配置键 | 默认 | 作用 |
| --- | --- | --- |
| `filter_proto_json` | 关 | 打开后对 proto 响应做结构化修补 |
| `filter_proto_json_emit_zerovalues` | 开（仅在上一项为 `true` 时有意义） | 为 json 里缺失的 key 补零值 |
| `filter_proto_json_int64_as_number` | 开（同上） | 把 int64 从字符串写成数字 |

`FilterProtoJson` 要求值恰好等于 `true`；两个子选项只有显式写成 `false` 才关闭。`Marshaler.JSON` 只对实现了 `proto.Message` 的响应调用 `FilterJsonWithStruct`，修补内容是：int64 字段按 `Int64AsNumber` 写成数字或字符串；`Ptr` 字段为 nil 时写成 `null`；json 里缺失的 key 按零值补上。Thrift 的响应不是 `proto.Message`，所以这三个键对 Thrift 链路不生效。

```yaml
config:
  filter_proto_json: true
  filter_proto_json_emit_zerovalues: false
  filter_proto_json_int64_as_number: true
```

### json_field_names: proto|camel

`json_field_names` 也在 `config:` 下面，控制 proto 响应的 key 用哪种拼写：`proto`（默认）输出 `owner_openid`、`Int64Value`；`camel` 输出 `ownerOpenid`、`int64Value`。

> **版本**：v0.6.0 起支持 `json_field_names`。

非法值（例如 `camelCase`）在配置加载时被 `Config.validate` 拒绝，而不是悄悄退回某一种：`invalid json_field_names: "camelCase", expected "proto" or "camel"`。请求方向的绑定对大小写和下划线都不敏感，两种取值下客户端都能用 `owner_openid` 或 `ownerOpenid` 发请求，这个选项只决定响应写哪种。

## form / query / path 的绑定细节

非 JSON 请求走 `BuildStructErr`，对每个导出字段调用 `findValue`，再交给 `setValue` 按 Kind 转换：`int`/`int8`/`int16`/`int32`/`int64` 用 `strconv.ParseInt(v, 10, 64)`；`uint` 系列用 `strconv.ParseUint`；`float32`/`float64` 用 `strconv.ParseFloat`；`string` 原样；`bool` 用 `strconv.ParseBool`；slice 按逗号切分后逐个解析。

两种「值不存在」的处理：空字符串不算值，`formValue`、`pathValue`、`InjectedValue` 都检查 `len(v) > 0`，`setValue` 收到空串也直接返回，所以 `?string_list=` 不会把 slice 设成空切片；字段不在请求里就保持零值，`BuildStructErr` 不会因此报错。

`Convertor` 优先于字段绑定：`BuildStructErr` 和 `BuildArgs` 都会先 `components(req).Convertor(类型名)`，命中就整体替换这个结构，注册名是类型的 `Name()`，例如 `SetConvertor("CommonValues", f)`，见 09-convertor.md。

## Thrift 的参数绑定差异

Thrift 的方法参数是一个参数列表，不是一个请求消息。生成代码会为每个方法生成一个 `<Service><Method>Args` 结构（例如 `TestServiceSayHelloArgs`），绑定逻辑在 `bindThriftArgsFromJSON`（`binding.go`）：

- 方法只有一个参数且该参数是消息：body 就是那个参数本身，按字段名解析；body 里出现不属于该参数的名字会被拒绝（400）。
- 方法只有一个参数且是标量：body 就是一个裸 JSON 值。
- 方法有多个参数：body 是一个 JSON 对象，key 按参数名对应，例如 `{"values":{"someId":123},"yourName":"a name","int64Value":7}`。参数名匹配会同时试 Go 字段名和 `json`/`thrift` 标签名，并且忽略大小写与下划线。body 没提到的参数保持零值；命名了不存在参数时返回 400 并列出方法真正的参数名。

> **版本**：v0.6.0 起，Thrift 的 JSON body 按键名对应参数（此前多参数方法会 panic）。

解析出 body 之后，query/form 补缺、path 覆盖、注入值覆盖的顺序与 gRPC 链路一致。更多细节见 13-grpc-thrift.md。

## 严格绑定：存在但不能用就报错

规则只有一句：**参数不存在就保持零值并成功；参数存在但没法用就明确报错（400）。** 理由是零值有歧义：把一个 `abc` 传给 int64 字段、留零值加 200 响应，和一个压根没传这个参数的请求在调用方看来完全一样。只有调用方能修，所以必须让它知道；错误消息里保留字段名、来源和原因，值被替换成 `<redacted>`。

```bash
$ curl -s 'http://127.0.0.1:8081/hello?int64_value=abc'
turbo: cannot bind Int64Value from query/form parameter: strconv.ParseInt: parsing "<redacted>": invalid syntax
```

嵌套结构、指针字段、slice 的情况：JSON 路径的嵌套消息由 `jsonpb` 按 body 创建，form/query/path 路径由 `BuildStructErr` 递归进入 `*Struct` 字段；它不会替你 new 一个嵌套消息，所以嵌套指针必须已经存在。指针为 nil 时 `BuildStructErr` 只记一条日志 `value is invalid, please check grpc-fieldmapping`，不当成绑定失败上报。slice 只要有一个元素解析失败就是 400，而不是丢弃整个列表。

## 关于 protoc-gen-buildfields

`protoc-gen-buildfields` 不生成任何 `*Fields` 类型。它读取 proto 的 `CodeGeneratorRequest`，把名字以 `Request` 结尾的消息及其嵌套消息字段写成 `gen/grpcfields.yaml`，形如 `grpc-fieldmapping:` 加一行 `  - SayHelloRequest[CommonValues values,]`。

`Config.loadFieldMapping` 把它读进 `fieldMappings`，`Generator.structFields` 据此生成 `grpcswitcher.go` 里的嵌套消息字面量，例如 `request := &g.SayHelloRequest{ Values: &g.CommonValues{}, }`。这就是嵌套指针非 nil 的来源。`setPathParams` 对 nil 的嵌套消息指针没有保护，会沿 `Elem()` 继续递归，所以不要手写一个不带嵌套字段初始化的请求结构，也不要删掉 `gen/grpcfields.yaml`；生成流程见 12-code-generation.md。

## 完整示例

`order.proto`（放在 `package_path` 目录下）：

```proto
syntax = "proto3";
package proto;
option go_package = "/;proto";

message GetOrderRequest {
    string order_id = 1;
    int64 user_id = 2;
    string device_code = 3;
    string note = 4;
}

message GetOrderResponse {
    string message = 1;
}

service OrderService {
    rpc getOrder (GetOrderRequest) returns (GetOrderResponse) {}
}
```

`service.yaml`：

```yaml
config:
  environment: development
  file_root_path: /tmp/turbodemo/src
  package_path: github.com/example/orderservice
  turbo_log_path:
  http_port: 8081
  grpc_service_name: OrderService
  grpc_service_host: 127.0.0.1
  grpc_service_port: 50061
  thrift_service_name: OrderService
  thrift_service_host: 127.0.0.1
  thrift_service_port: 50062

urlmapping:
  - GET,POST /orders/{order_id:[a-zA-Z0-9]+} OrderService GetOrder

interceptor:
  - GET,POST /orders/{order_id:[a-zA-Z0-9]+} DeviceInterceptor
```

服务实现把收到的四个字段原样打出来，方便看清每个值从哪来：

```go
package impl

import (
	"context"
	"fmt"

	"github.com/example/orderservice/gen/proto"
	"google.golang.org/grpc"
)

func RegisterServer(s *grpc.Server) {
	proto.RegisterOrderServiceServer(s, &OrderService{})
}

type OrderService struct{}

func (s *OrderService) GetOrder(ctx context.Context, req *proto.GetOrderRequest) (*proto.GetOrderResponse, error) {
	return &proto.GetOrderResponse{Message: fmt.Sprintf(
		"order_id=%s user_id=%d device_code=%s note=%s",
		req.OrderId, req.UserId, req.DeviceCode, req.Note)}, nil
}
```

重新生成并启动：

```bash
ROOT=/tmp/turbodemo/src
turbo generate github.com/example/orderservice -r grpc -I "$ROOT/github.com/example/orderservice"
cd "$ROOT/github.com/example/orderservice"
go build ./...
./orderservice
```

一次请求同时带上四种来源：

```bash
curl -s -X POST 'http://127.0.0.1:8081/orders/A100?user_id=7&device_code=from-query' \
  -H 'Content-Type: application/json' \
  -d '{"note":"hi","device_code":"from-body","orderId":"from-body"}'
```

```json
{"message":"order_id=A100 user_id=7 device_code=device-from-server note=hi"}
```

逐个字段看：

| 字段 | 最终值 | 从哪来 | 为什么是它 |
| --- | --- | --- | --- |
| `order_id` | `A100` | path | body 里有 `orderId`，但 `setPathParams` 在 body 之后运行，path 覆盖 body |
| `user_id` | `7` | query | body 没提这个字段，`bindJSONGaps` 用 query 补空缺 |
| `device_code` | `device-from-server` | injected | 注入值最后写入，覆盖 body 与 query，并记一条 warning |
| `note` | `hi` | body | body 明确提到，query/form 不再参与 |

冲突时的赢家（去掉注入拦截器再试）：query 与 body 都有 `device_code` 时 body 赢，结果是 `device_code=from-body`；path 是 `A100`、body 写 `orderId` 时 path 赢，结果是 `order_id=A100`。

```bash
curl -s -X POST 'http://127.0.0.1:8081/orders/A100?device_code=from-query' \
  -H 'Content-Type: application/json' -d '{"device_code":"from-body"}'
```

## 相关阅读

- [service.yaml 配置](03-service-yaml.md)：`filter_proto_json`、`json_field_names` 放在哪一节
- [gRPC 与 Thrift](13-grpc-thrift.md)：Thrift 参数列表与前缀差异
- [错误与 status code](10-errors.md)：绑定失败的码与消息形状
- [转换器](09-convertor.md)：用 `SetConvertor` 整体替换一个结构
- [代码生成](12-code-generation.md)：`gen/grpcfields.yaml` 怎么来、什么时候要重新生成
