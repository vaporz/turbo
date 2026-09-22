# CLI 与代码生成

这篇讲 turbo 的两个命令行二进制怎么装、`turbo create` 和 `turbo generate` 的每个参数、它们各自生成哪些文件与目录，以及 `protoc-gen-go` 版本、生成物是否入库、生成失败会不会写坏已有文件这些实践问题。

## 安装

turbo 仓库是一个 Go module（`module github.com/vaporz/turbo`），里面有两个 `package main`：

| 二进制 | 源码目录 | 作用 |
| --- | --- | --- |
| `turbo` | `turbo/` | 创建项目、生成 switcher 与 pb/thrift 代码 |
| `protoc-gen-buildfields` | `protoc-gen-buildfields/` | protoc 插件，写 `gen/grpcfields.yaml` |

```bash
go install github.com/vaporz/turbo/turbo@latest
go install github.com/vaporz/turbo/protoc-gen-buildfields@latest
```

在仓库里开发时，`Makefile` 的 `install` 目标做的是同一件事：

```bash
make install   # 等价于 cd turbo && go install; cd protoc-gen-buildfields && go install
```

`protoc-gen-buildfields` 必须能在 `PATH` 里被 `protoc` 找到，否则 `turbo generate -r grpc` 会在工具链检查阶段停下。两个二进制的关系是：`turbo generate` 调 `protoc`，`protoc` 按 `--buildfields_out=...` 参数调起 `protoc-gen-buildfields`，插件把结果写进 `gen/grpcfields.yaml`。

## turbo create

```
turbo create package_path ServiceName
```

别名 `c`。`ServiceName` 必须是 CamelCase：`IsCamelCase` 用的正则是 `^([A-Z]+[a-z]*)+$`，`test_create_service` 会被拒绝。

| flag | 简写 | 默认 | 说明 |
| --- | --- | --- | --- |
| `--rpctype` | `-r` | `grpc` | 只能是 `grpc` 或 `thrift` |
| `--force` | `-f` | `false` | 直接覆盖已有文件，跳过目录检查与交互确认 |
| `--rootpath` | `-p` | `.` | 新包创建在哪个目录下，会先转成绝对路径 |

参数错误时返回的文案（来自 `turbo/cmd/create.go`）：少于两个参数是 `invalid args`；服务名不是 CamelCase 是 `[xxx] is not a CamelCase string`；`-r` 取值不对是 `invalid value for -r, should be grpc or thrift`。

项目最终落在 `<rootpath 的绝对路径>/<package_path>`。`rootpath` 是「包含 package_path 这棵目录树」的根，`file_root_path` 会被写成这个绝对路径，`package_path` 会被写成第一个参数。

### 不加 -f 时的交互

目标目录已存在时，`validateServiceRootPath` 会在 stdin 上依次问 `Path '<root>/<package_path>' already exist! Do you want to remove this directory before creating a new project? (type 'y' to remove):` 和 `All files in that directory will be lost, are you sure? (type 'y' to continue):`。

两次都输入 `y` 才会 `os.RemoveAll` 整个目录，第二次不是 `y` 会 `panic("aborted")`。第一次不是 `y` 则不删除，继续逐个写文件。删除整棵目录不可逆，建议先确认目录内容，或者用 `-f` 明确表示「只覆盖要生成的文件，不删目录」。

### 生成哪些文件

`-r grpc`：

```
<root>/<package_path>/
├── service.yaml
├── <servicename-lower>.proto
├── main.go
├── gen/
│   ├── grpcfields.yaml          # protoc-gen-buildfields 写出
│   ├── grpcswitcher.go          # turbo 写出
│   └── proto/<servicename-lower>.pb.go
├── grpcapi/
│   ├── <servicename-lower>api.go
│   └── component/components.go
└── grpcservice/
    ├── <servicename-lower>.go
    └── impl/<servicename-lower>impl.go
```

`-r thrift`：

```
<root>/<package_path>/
├── service.yaml
├── <servicename-lower>.thrift
├── main.go
├── gen/
│   ├── thriftfields.yaml        # go run gen/thrift/build.go 写出
│   ├── thriftswitcher.go        # turbo 写出
│   └── thrift/                  # build.go 与 thrift 编译器写出的 gen-go/gen/*.go
├── thriftapi/
│   ├── <servicename-lower>api.go
│   └── component/components.go
└── thriftservice/
    ├── <servicename-lower>.go
    └── impl/<servicename-lower>impl.go
```

几点需要知道：

- `create` 不只写模板，它紧接着就把 pb/thrift 代码和 switcher 也生成出来，所以创建项目时同样需要 protoc/thrift 工具链。`create` 不走 `Generate()` 的 `checkToolchain`，工具缺失时表现为外部命令失败（`turbo: bash -c <命令> failed: ...`）。
- 模板生成的 `service.yaml` 里只有一份示例路由 `GET /hello <ServiceName> SayHello`，同时写好了 `grpc_service_name` 与 `thrift_service_name`，两个值相同。
- `create` 不生成 `go.mod`。import 路径取自 `package_path`，所以要把它放在拥有该前缀的模块里，或者在服务根目录自己 `go mod init <package_path>`。
- `service.yaml` 已存在时不会被 `create` 覆盖（即使加了 `-f`），其余模板文件会被覆盖。

## turbo generate

```
turbo generate package_path -r grpc|thrift [-I path]...
```

别名 `g`。

| flag | 简写 | 必填 | 说明 |
| --- | --- | --- | --- |
| `--rpctype` | `-r` | 是（没有默认值） | 只能是 `grpc` 或 `thrift` |
| `--include-path` | `-I` | grpc 必填 | 存放 `.proto` / `.thrift` 的目录，要求绝对路径，可重复 |

参数错误时的文案（来自 `turbo/cmd/generate.go`）：

- 没有 package_path：`Usage: generate [package_path] -r [grpc|thrift] -I (absolute_paths_to_proto|thrift_files)`
- 没有 `-r`：`missing rpctype (-r)`
- `-r` 取值不对：`invalid rpctype`
- grpc 没给 `-I`：`missing .proto file path (-I)`

`-I` 的目录同时承担两个作用：传给 protoc/thrift 作为 include 路径，以及被 `findFileIn` 用来找 `service.yaml`。因此 thrift 虽然不强制要求 `-I`，不给的话 `findFileIn` 会 panic：

```
can not find service.yaml in any:
```

### -I 校验

`ValidateIncludePaths`（`generator.go`）在生成之前检查每个路径：

- 读不到：`turbo: -I <path> cannot be read: <err>`
- 传的是文件而不是目录：`turbo: -I <path> is a file, but -I takes the directory that contains your .proto or .thrift files; pass <dir> instead`

`turbo/cmd/generate.go` 会先跑一次这个校验，让命令回一句人话而不是 panic。

> **版本**：v0.6.0 起校验 `-I` 并预检工具链（此前传文件会拼出 `service.proto/*.proto` 这种路径，让 protoc 抱怨一个没人提过的文件）。

### 它怎么调 protoc / thrift

grpc（`GenerateProtobufStub`）：

```bash
protoc <-I path path/*.proto ...> \
  --go_out=plugins=grpc:<ServiceRoot>/gen/proto \
  --buildfields_out=service_root_path=<ServiceRoot>:<ServiceRoot>/gen/proto
```

`turbo create` 走的是另一份 `Options`：` -I <ServiceRoot> <ServiceRoot>/<名字小写>.proto `。

thrift（`GenerateThriftStub`）：

```bash
thrift <-I path ...> -r --gen go:package_prefix=<PkgPath>/gen/thrift/gen-go/ \
  -o <ServiceRoot>/gen/thrift <ServiceRoot>/<名字小写>.thrift
```

thrift 生成之后还会：

```bash
go run <ServiceRoot>/gen/thrift/build.go                       # 写 gen/thriftfields.yaml
go run <ServiceRoot>/gen/thrift/build.go -n <Service>,<Method> # 为每条路由取参数列表
```

生成顺序是：grpc 为 `GenerateProtobufStub`、`loadFieldMapping`、`GenerateGrpcSwitcher`；thrift 为 `GenerateThriftStub`、`GenerateBuildThriftParameters`、`loadFieldMapping`、`GenerateThriftSwitcher`。`loadFieldMapping` 读的分别是 `gen/grpcfields.yaml` 与 `gen/thriftfields.yaml`，读不到会 panic，所以不要手工删这两个文件。

### 工具链检查

`checkToolchain` 在写任何文件之前跑：

- grpc 需要 `protoc`、`protoc-gen-go`、`protoc-gen-buildfields`；thrift 只需要 `thrift`。
- 缺工具：`turbo: <tool> is not in PATH, install it before generating (<err>)`
- 会执行 `protoc --version` 并记一条日志。`protoc-gen-go --version` 的输出决定是否警告，见下一节。

> **版本**：v0.6.0 起，方法名排序后再写进 switcher，所以同一份配置重复生成得到相同文件。

## --go_out=plugins=grpc 与 protoc-gen-go 版本

turbo 用 `--go_out=plugins=grpc:...` 生成带 gRPC service 的 pb 代码。遗留的 `protoc-gen-go` 认识这个选项（到 v1.26.x 为止），现代的会拒绝并打印：

```
--go_out: protoc-gen-go: plugins are not supported; use 'protoc --go-grpc_out=...'
```

两种插件常常同时装在机器上，而 **protoc 用 PATH 里的第一个**，所以要把遗留插件放到前面：

```bash
go install github.com/golang/protobuf/protoc-gen-go@v1.5.1
export PATH="$GOPATH/bin:$PATH"
```

`gen.sh` 的推荐写法就是上面两行的顺序：先安装遗留插件，再把 `$GOPATH/bin` 提到 `PATH` 最前面，然后才调用 `turbo generate`。仓库里 v0.6.2 没有 `gen.sh` 文件，重新生成测试夹具有一份写在注释里的配方，位于 `test/testservice/service.yaml` 顶部：

```bash
# turbo generate github.com/vaporz/turbo/test/testservice -r grpc   -I $PWD/test/testservice
# turbo generate github.com/vaporz/turbo/test/testservice -r thrift -I $PWD/test/testservice
```

那份注释还说明：`file_root_path` 会与 `package_path` 拼接，必须指向包含 `package_path` 的目录；普通 checkout 下可以按注释里的办法把仓库软链到一个 GOPATH 形状的临时目录，生成完再改回 `file_root_path`。

区分两种插件：`protoc-gen-go --version` 在遗留插件上回答 `this program should be run by protoc`，在现代插件上打印 `protoc-gen-go v1.x.y`。`legacyProtocGenGo` 就是按这个判断的。

turbo 发现现代插件时**只警告不拒绝**，因为能不能生成取决于插件构建而不是版本号：v1.26.0 接受 `plugins=grpc`，v1.31.0 拒绝它。当 protoc 真的因为 `plugins are not supported` 失败时，turbo 会把修复方法附在错误里（`legacyProtocGenGoFix`）：安装 `protoc-gen-go@v1.5.1`，并确保它在 `PATH` 里排第一。错误消息里还会带上失败命令和它输出的最后几行。

## 生成物要不要入库

repo 里的实际做法是两者都有，按用途区分：

- `test/testservice/gen/` 是提交进仓库的测试夹具（14 个文件，含 `grpcfields.yaml`、`grpcswitcher.go`、pb 与 thrift 生成代码）。夹具入库才能在没有 protoc/thrift 的机器上跑测试。
- `test/testcreateservice/` 是集成测试生成出来的，被 `.gitignore` 忽略；`.gitignore` 里还有 `test/testservice/testservice`（编译出的二进制）、`turbo.log`、`.idea/`、`vendor/`、`log/`、`doc/build/`。

业务项目通常把整个 `gen/` 加进 `.gitignore`，只在构建机上跑生成。这要求构建机装好 protoc、protoc-gen-go、protoc-gen-buildfields（grpc）或 thrift，并且有一个把生成命令串起来的 `gen.sh` 放进构建流程。注意 `gen/grpcfields.yaml` 与 `gen/thriftfields.yaml` 是生成流程的中间产物，如果 `gen/` 不入库，构建时必须先跑生成再 `go build`。

`Makefile` 里和生成相关的目标：

| 目标 | 做什么 |
| --- | --- |
| `make install` | 安装 `turbo` 与 `protoc-gen-buildfields` |
| `make test` | `go test -p=1 -cover -coverpkg github.com/vaporz/turbo github.com/vaporz/turbo github.com/vaporz/turbo/test`，然后 `cd test/testcreateservice && go build ./...`、`cd test/testservice && go build ./...` |
| `make clean-gen` | `rm -rf test/testcreateservice` |
| `make tidy` | 先 `clean-gen` 再 `go mod tidy` |
| `make vuln` | 跑 `govulncheck github.com/vaporz/turbo github.com/vaporz/turbo/test` |
| `make fmt` | `gofmt -w` 仓库内的 Go 源码 |

`make tidy` 必须先清掉 `test/testcreateservice` 的原因写在 README 和 Makefile 注释里：集成测试会生成这个目录并留在原地，它是本模块的一部分，`go mod tidy` 会计入它 import 的依赖，于是「只有生成树用到」的依赖会被记成直接依赖，和干净 checkout 的答案不一致。

## 原子写

> **版本**：v0.6.0 起，生成失败不会把已有产物写坏。

`writeFileWithTemplate`（`generator.go`）不再先截断目标文件，而是：

1. 在目标文件同目录 `os.CreateTemp(dir, 文件名+".tmp")`；
2. 模板执行失败、`Close` 失败、`Chmod` 失败或 `Rename` 失败时，删掉临时文件并 panic；
3. 全部成功才 `os.Rename` 覆盖目标。

覆盖范围是 turbo 自己写出的文件：`service.yaml`、`*.proto` / `*.thrift`、`main.go`、`grpcapi`/`thriftapi`/`grpcservice`/`thriftservice` 下的模板文件、`gen/grpcswitcher.go`、`gen/thriftswitcher.go`、`gen/thrift/build.go`。由 protoc、thrift 编译器以及 `gen/thrift/build.go` 自己写出的文件（pb.go、thrift gen-go、`grpcfields.yaml`、`thriftfields.yaml`）是这些工具直接创建的，不受这层保护。

## 完整示例：从零到跑起来

```bash
# 1. 安装工具
go install github.com/vaporz/turbo/turbo@latest
go install github.com/vaporz/turbo/protoc-gen-buildfields@latest
go install github.com/golang/protobuf/protoc-gen-go@v1.5.1
export PATH="$GOPATH/bin:$PATH"

# 2. 准备一个 GOPATH 形状的根目录
export ROOT=/tmp/turbodemo/src
mkdir -p "$ROOT"

# 3. 创建项目（-p 指向包含 package_path 的根）
turbo create github.com/example/greeterservice GreeterService -r grpc -p "$ROOT"

# 4. 进入服务根目录，重新生成（改过 proto 或 service.yaml 之后都要跑）
cd "$ROOT/github.com/example/greeterservice"
turbo generate github.com/example/greeterservice -r grpc -I "$ROOT/github.com/example/greeterservice"

# 5. 编译。creator 不生成 go.mod，这里按 package_path 建一个
go mod init github.com/example/greeterservice
go mod tidy
go build ./...

# 6. 启动：main.go 同时拉起 gRPC 服务与 HTTP 网关
go run main.go

# 7. 另一个终端验证
curl -s 'http://127.0.0.1:8081/hello?your_name=turbo'
```

期望的文件树（第 4 步之后）：

```
/tmp/turbodemo/src/github.com/example/greeterservice/
├── go.mod
├── main.go
├── service.yaml
├── greeterservice.proto
├── gen/
│   ├── grpcfields.yaml
│   ├── grpcswitcher.go
│   └── proto/greeterservice.pb.go
├── grpcapi/
│   ├── greeterserviceapi.go
│   └── component/components.go
└── grpcservice/
    ├── greeterservice.go
    └── impl/greeterserviceimpl.go
```

换成 thrift 只需把 `-r grpc` 改成 `-r thrift`，并注意 `service.yaml` 里 `grpc_service_name` 与 `thrift_service_name` 都保持有值：thrift 的 `gen/thrift/build.go` 用 `grpc_service_name` 生成反射列表，thrift 的客户端组件模板用的也是 `grpc_service_name` 的第一个值。两者不一致时生成的 Thrift 代码会出错。

## 相关阅读

- [快速开始](02-getting-started.md)：最短路径跑起一个服务
- [service.yaml 配置](03-service-yaml.md)：生成后要改哪些键
- [参数绑定](11-binding.md)：`grpcfields.yaml` 与嵌套消息指针的关系
- [gRPC 与 Thrift](13-grpc-thrift.md)：两条链路的启动与客户端
- [部署](18-deployment.md)：构建机上的生成流程
