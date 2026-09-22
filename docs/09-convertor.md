# 转换器 Convertor

这篇讲什么：`Convertor` 的签名、它在绑定流程里什么时候被查表命中，为什么需要它，以及为某个结构体写转换器并在 `convertor:` 段注册的完整写法。

## 签名与职责

```go
type Convertor func(r *http.Request) reflect.Value
```

它定义在 `component.go`：给一个 `*http.Request`，返回某个类型的一个值。框架按结构体类型名注册、按结构体类型名查表，命中后就不再按字段名逐个绑定，直接采用你返回的值。

公开的注册与查询方法在同一文件：

```go
// SetConvertor registers a Convertor on a type
// usage: SetConvertor(new(SomeInterface), convertorFunc)
func (c *Components) SetConvertor(field string, convertorFunc Convertor)

// Convertor returns the Convertor for this type
func (c *Components) Convertor(theType string) Convertor
```

注释里那句 `usage: SetConvertor(new(SomeInterface), convertorFunc)` 与实现不符：参数 `field` 是 `string`，`registerConvertor` 直接把它当 map 的 key，所以实际传的是类型名，不是 `new(...)`。以代码为准。

### 解决什么问题

默认绑定是逐字段做的：`BuildStructErr`（`runtime.go`）遍历结构体导出的标量字段，用 `findValue` 从请求里按字段名找值，再用 `setValue` 按字段 Kind 转换。当值不是“能被字段名直接取到的标量”时，这条路走不通：

- 值要从请求头、原始 body、多个参数组合或外部系统算出来；
- 值要按自定义规则解析，例如一个签名字符串里编码了多个字段；
- 字段类型是嵌套消息，而请求方用完全不同的形状表达它。

转换器的粒度是“一个类型”：命中后整个结构体由你的函数负责，框架不再往里看。

## 查表逻辑

`BuildStructErr` 是 grpc 表单请求的绑定入口，它的开头就查表：

```go
convertor := components(req).Convertor(theValue.Type().Name())
if convertor != nil {
	theValue.Set(convertor(req).Elem())
	return nil
}
```

两处细节要记住：

- 查表的 key 是 `theValue.Type().Name()`，也就是 Go 结构体的类型名，例如 `CommonValues`、`SayHelloRequest`；这是普通字符串比较，大小写敏感。
- 这里对返回值调了 `.Elem()`，所以转换器要返回指向该结构体的指针的 `reflect.Value`，通常写成 `reflect.ValueOf(&proto.CommonValues{})`。返回结构体值本身会让 `Elem()` panic。

嵌套的指针字段是第二个命中点：

```go
fieldValue := theValue.FieldByName(fieldName)
if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct {
	convertor := components(req).Convertor(fieldValue.Type().Elem().Name())
	if convertor != nil {
		fieldValue.Set(convertor(req))
		continue
	}
	if err := BuildStructErr(s, fieldValue.Type().Elem(), fieldValue.Elem(), req); err != nil {
		return err
	}
	continue
}
```

嵌套字段这里直接 `Set(convertor(req))`，没有 `.Elem()`：返回的指针赋给这个指针字段。返回“指向结构体的指针”的 `reflect.Value` 能同时满足这两处。

thrift 表单请求的入口 `BuildArgs` 对“指向结构体的参数”做同一件事，同样按 `valueType.Elem().Name()` 查表：

```go
if field.Type.Kind() == reflect.Ptr && valueType.Elem().Kind() == reflect.Struct {
	convertor := components(req).Convertor(valueType.Elem().Name())
	if convertor != nil {
		params[i] = convertor(req)
		continue
	}
	structName := valueType.Elem().Name()
	...
}
```

调用链上：`handler` 先 `parseRequestForm` 准备参数，`switcherFunc` 再调 `BuildRequest`（thrift 是 `BuildThriftRequest`），`BuildRequest` 在非 JSON 分支里调 `BuildStructErr`，查表就发生在那里。

### 一个必须知道的边界

JSON 请求不走转换器。`BuildRequest` 在 `Content-Type` 包含 `application/json` 时直接 `jsonpb.Unmarshaler.Unmarshal`，之后依次做 `bindJSONGaps`、`setPathParams`、`bindJSONInjected`，全程没有 `Convertor` 调用；thrift 的 JSON 分支走 `bindThriftArgsFromJSON`，也没有。测试夹具的注释把这点说得很直白：注册 `SayHelloRequest` 的转换器时用的是 `testGet` 而不是 `testPostWithContentType`。

所以用转换器替代默认绑定前，请确认这些请求不是 `application/json`。

## 什么时候需要它

- 某个结构体的值来自请求头或环境，而不是参数，例如把设备令牌解析成 `ClientInfo`。
- 值需要按自定义格式解析，例如一个签名字符串里编码了多个字段。
- 某个嵌套消息在请求里没有对应字段名，默认绑定只能给它留零值。
- 想在绑定阶段统一给某个类型填默认值。

不要用它做带业务校验的逻辑：签名里没有 error，失败只能靠返回零值表达。

## 完整示例

测试夹具里这两个消息分别定义在 `test/testservice/shared.proto` 与 `test/testservice/testservice.proto`，同属 `package proto`：

```protobuf
// shared.proto
message CommonValues {
    int64 someId = 1;
}

// testservice.proto
message SayHelloRequest {
    CommonValues values = 1;
    string yourName = 2;
}
```

为 `CommonValues` 写一个转换器，让它的值不依赖请求参数：

```go
package component

import (
	"net/http"
	"reflect"

	"github.com/vaporz/turbo"
	"github.com/vaporz/turbo/test/testservice/gen/proto"
)

// convertProtoCommonValues 把 CommonValues 整个交给它构造。
// 必须返回指针的 reflect.Value：BuildStructErr 会对它调 Elem()。
var convertProtoCommonValues turbo.Convertor = func(req *http.Request) reflect.Value {
	result := &proto.CommonValues{}
	result.SomeId = 1111111
	return reflect.ValueOf(result)
}
```

注册后，请求 `/hello/testtest?bool_value=true` 的响应里 `values.someId` 是 1111111，这正是 `TestGrpcService` 的断言：

```go
s.Components.SetConvertor("CommonValues", component(s.Server, "convertProtoCommonValues").(turbo.Convertor))
testGet(t, "http://localhost:"+httpPort+"/hello/testtest?bool_value=true",
	`{"message":"{\"values\":{\"someId\":1111111},\"yourName\":\"testtest\",\"boolValue\":true}"}`)
```

也可以转换顶层消息本身。夹具里的 `convertProtoSayHelloRequest` 把 `YourName` 写死：

```go
var convertProtoSayHelloRequest turbo.Convertor = func(req *http.Request) reflect.Value {
	result := &proto.SayHelloRequest{}
	result.YourName = "from convertor"
	return reflect.ValueOf(result)
}
```

注册 `SayHelloRequest` 后，请求 `/hello/testtest` 返回 `[grpc server]Hello, from convertor`。

注册写在 `InitService` 里（服务启动前）：

```go
func (i *ServiceInitializer) InitService(s turbo.Servable) error {
	s.RegisterComponent("convertProtoCommonValues", convertProtoCommonValues)
	s.RegisterComponent("convertProtoSayHelloRequest", convertProtoSayHelloRequest)
	return nil
}
```

`service.yaml` 的 `convertor:` 段只有两列，第一列是类型名，第二列是组件名：

```yaml
urlmapping:
  - GET /hello/{your_Name:[a-zA-Z0-9]+} TestService SayHello

convertor:
  - CommonValues convertProtoCommonValues
```

它和其它组件段不同，不走 `loadMappings` 的四列格式，`config.go` 的 `loadConvertor` 单独解析：

```go
func (c *Config) loadConvertor() [][4]string {
	mapping := make([][4]string, 0)
	lines := c.GetStringSlice("convertor")
	for _, line := range lines {
		values := strings.Split(strings.TrimSpace(line), " ")
		name := strings.TrimSpace(values[0])
		convertorName := strings.TrimSpace(values[1])
		mapping = append(mapping, [4]string{name, convertorName})
	}
	return mapping
}
```

装载时 `loadComponents` 用第一列做类型名、第二列取组件：

```go
for _, m := range s.Config.mappings[convertors] {
	c.SetConvertor(m[0], getComponentByName(s, m[1]).(Convertor))
	log.Info("convertor:", m)
}
```

类型名没注册过时 `getComponentByName` 会 panic（`no such component: <name>, forget to register?`）。

## 与参数绑定的关系

转换器是绑定优先级之外的一条旁路：命中类型就直接返回，整个结构体不再逐字段走 `injected > path > body > query/form` 的优先级；没命中的部分仍按 `11-binding.md` 的规则绑定。`SayHelloRequest.Values` 这种指针字段会先查 `CommonValues` 的转换器，命中就填该字段，否则递归进去按字段名绑定；`BuildArgs` 对 thrift 参数同理。

## 常见坑

- 忘记返回指针：`theValue.Set(convertor(req).Elem())` 会 panic。
- 类型名写错（例如写成带包路径的 `proto.CommonValues`）：查不到就静默走默认绑定，字段留零值，不报错。
- 用 JSON 请求测转换器：JSON 分支不查表，看起来像“转换器没生效”。
- 转换器返回 nil 指针：后续 RPC 可能在序列化阶段 panic，请保证返回可用的值。
- `Components.Reset()` 会清空 `convertorMap`，运行时手工调整组件后记得重新 `SetConvertor`。

## 相关阅读

- [05-components.md](05-components.md)
- [07-preprocessor-postprocessor.md](07-preprocessor-postprocessor.md)
- [11-binding.md](11-binding.md)
- [13-grpc-thrift.md](13-grpc-thrift.md)
- [17-testing.md](17-testing.md)
