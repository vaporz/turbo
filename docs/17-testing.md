# 测试你写的 turbo 服务

这篇讲怎么测一个用 turbo 搭起来的服务：从直接调用组件、到构造请求验证绑定、再到起一个真 server 打 HTTP 进去，以及怎么捕获 turbo 自己的日志、为什么每个测试都要用独立的配置文件。原则是能在低层次钉住的行为就不要放端到端，凡是“HTTP 层能看到的结果”（状态码、响应体、日志行）都值得一条端到端测试。

> **版本**：v0.6.2 起，`turbo.SetOutput` 是运行时接管 turbo 日志的唯一公开入口；`config.log_level`、路由审计、配置稳定等待与空配置拒绝这些行为都可以在测试里断言（下面参考的测试就是 v0.6.2 的 `test/integration_test.go`）。

## 三个层次

| 层次 | 测什么 | 手段 | 成本 |
|---|---|---|---|
| 组件单测 | 拦截器、预处理器、后处理器、转换器本身 | 直接 new 出来调 `Before` / `After` / 函数本身 | 最低 |
| 绑定测试 | 参数从 query、form、path、body、注入值怎么落到结构体字段 | `httptest` 构造请求，经 server 走一遍绑定 | 中 |
| 端到端 | 路由、绑定、RPC、序列化、状态码整条链 | 起真 server，HTTP 打进去 | 最高，但最接近生产 |

## 组件单测：直接调用

拦截器是普通接口（`component.go` 的 `Interceptor`），不需要 server 就能测：

```go
type authInterceptor struct {
	turbo.BaseInterceptor
}

func (a *authInterceptor) Before(resp http.ResponseWriter, req *http.Request) error {
	if req.Header.Get("X-Token") != "ok" {
		return turbo.Errorf(http.StatusUnauthorized, "missing token")
	}
	turbo.InjectParam(req, "user_id", "u-1")
	return nil
}

func TestAuthInterceptor(t *testing.T) {
	i := &authInterceptor{}

	bad := httptest.NewRequest(http.MethodGet, "/hello", nil)
	err := i.Before(httptest.NewRecorder(), bad)
	if err == nil || turbo.StatusOf(err) != http.StatusUnauthorized {
		t.Fatalf("err = %v, want a 401 error", err)
	}

	good := httptest.NewRequest(http.MethodGet, "/hello", nil)
	good.Header.Set("X-Token", "ok")
	if err := i.Before(httptest.NewRecorder(), good); err != nil {
		t.Fatal(err)
	}
	if v, ok := turbo.InjectedValue("UserId", good); !ok || v != "u-1" {
		t.Fatalf("injected = %q, %v; want u-1, true", v, ok)
	}
}
```

`turbo.Errorf` 给错误带上 HTTP 状态，`turbo.StatusOf` 把它读回来，两个都是公开 API。

## 构造请求测绑定

绑定要覆盖的来源优先级是 `injected > path > body > query/form`，但直接调 `BuildStructErr` / `BuildRequest` 在外面测不了：它们要从请求 context 里取 `*Components`，而那个 key 是包内未导出的：

```go
var componentsKey key = 0

func components(req *http.Request) *Components {
	return req.Context().Value(componentsKey).(*Components)
}
```

外部包塞不进这个键，`components(req)` 会对 nil 做类型断言然后 panic。所以作为使用者，绑定应该用 `httptest` 打到运行中的 server 上验证，或者写端到端测试；想直接测这些函数只能在 turbo 包内部做（`strict_binding_test.go` 就是内部测试）。

自己搭请求时注意两点，它们和 turbo 包内的测试写法一致：

- `req.Form` 要自己设：turbo 的 `formValue` 只读 `req.Form`，不看 `req.URL.Query()`。服务运行时由 `parseRequestForm` 填好，测试里写 `req.Form = req.URL.Query()`。
- path 变量用 `mux.SetURLVars` 放进请求，否则绑定看不到路由变量。

公开 API 里能独立验证的是“注入”这一半，因为它不依赖组件：

```go
func TestInjectedParamIsSpellingInsensitive(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/hello?device_code=from-query", nil)
	req.Form = req.URL.Query()
	turbo.InjectParam(req, "device_code", "A3")

	for _, name := range []string{"DeviceCode", "device_code", "DEVICE_CODE"} {
		if v, ok := turbo.InjectedValue(name, req); !ok || v != "A3" {
			t.Fatalf("%s = %q, %v; want A3, true", name, v, ok)
		}
	}
}
```

`InjectParam` 直接改传入的请求（内部会 `*req = *req.WithContext(...)`），调用之后要继续用同一个 `*http.Request`。

## 端到端测试骨架

下面这份骨架的每一步都是必要的：动态取端口、把配置写进 `t.TempDir()`、先起 RPC 服务再起 HTTP、轮询等就绪、`defer s.Stop()`。`gen`、`grpcapi/component`、`grpcservice/impl` 这三个包名和 `GrpcSwitcher`、`GrpcClient`、`RegisterServer` 这三个符号是 `turbo generate -r grpc` 生成的（见 `test/testservice` 的实际布局），换成你自己的模块路径即可。

```go
package service_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vaporz/turbo"
	"github.com/your/module/gen"
	"github.com/your/module/grpcapi/component"
	"github.com/your/module/grpcservice/impl"
)

// freePort 让内核挑一个没人监听的口，避免硬编码端口互相撞车。
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
}

// writeConfig 把配置写进本测试自己的临时目录（独立文件的原因见下文）。
func writeConfig(t *testing.T, httpPort, grpcPort string) string {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "service.yaml")
	yaml := fmt.Sprintf(`config:
  environment: development
  http_port: %s
  grpc_service_name: YourService
  grpc_service_host: 127.0.0.1
  grpc_service_port: %s

urlmapping:
  - GET /hello YourService SayHello
`, httpPort, grpcPort)
	require.NoError(t, os.WriteFile(cfg, []byte(yaml), 0o644))
	return cfg
}

// startService 起一个真的 turbo server，返回 HTTP 地址和停止函数。
func startService(t *testing.T) (string, func()) {
	t.Helper()
	httpPort := freePort(t)
	grpcPort := freePort(t)
	cfg := writeConfig(t, httpPort, grpcPort)

	s := turbo.NewGrpcServer(nil, cfg)
	s.StartGrpcService(impl.RegisterServer)
	s.StartHTTPServer(component.GrpcClient, gen.GrpcSwitcher)

	base := "http://127.0.0.1:" + httpPort
	waitReady(t, base+"/hello")
	return base, s.Stop
}

// waitReady 轮询到 server 应答为止：监听在 goroutine 里，固定 sleep 会随机失败。
func waitReady(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server did not start serving %s", url)
}

func TestSayHello(t *testing.T) {
	base, stop := startService(t)
	defer stop()

	resp, err := http.Get(base + "/hello?your_name=turbo")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, string(body), "turbo")
}
```

容易踩的点：

- `StartGrpcService` 和 `StartHTTPServer` 都会调用 `Initializer.InitService`，传了非 nil 的 initializer 就会被调两次；不需要初始化逻辑时传 `nil`。
- 配置重载的 watcher 是 `StartHTTPServer`（或 `Start`）里装的，只调 `StartGrpcService` 不会有热重载。
- `config` 段里的 `file_root_path` 和 `package_path` 只有 `turbo create` / `turbo generate` 用得到，运行期不需要；CI 里还要跑 `turbo generate` 时，`file_root_path` 必须是绝对路径。
- 想一步启动也可以用 `s.Start(component.GrpcClient, gen.GrpcSwitcher, impl.RegisterServer)`；Thrift 的 `Start` 里多一个固定 1 秒的 `time.Sleep`。

## 为什么每个测试用独立配置文件

`test/integration_test.go` 的 `testConfigPath` 注释把原因写得很直白（原文摘录）：

> ... a server's watcher is never torn down -- neither by Stop() nor by the reload that replaces its Config. When all tests share a single config file, every rewrite therefore fires a hot reload in every server that was ever started, including servers that were already stopped ...

一个测试一份配置，放在 `t.TempDir()` 里，顺便避免把仓库里签入的 `service.yaml` 改坏。

还要注意临时目录的位置。热重载靠 inotify，而 inotify 在某些挂载上不工作（同一段注释）：

> ... the hot reload under test is driven by inotify: on a DrvFs mount (for example a repo checked out under /mnt/d in WSL) events are dropped, and the watcher then never reports the change at all.

在 WSL 下把仓库放在 `/mnt/d/...` 时，`t.TempDir()` 通常落在 Linux 侧临时目录里，反而是安全的；但如果你手动把测试配置写在仓库目录下，重载测试会莫名其妙地不生效。

重载相关的断言也不要用固定 sleep：测试文件里的 `testGetEventually` 每 50 毫秒轮询一次，直到响应体等于期望值或超时。切换是异步的，慢文件系统上的等待可能比 sleep 更长。

## 捕获日志做断言

turbo 的日志是断言“服务内部发生了什么”的公共入口，用 `turbo.SetOutput` 接管。`test/integration_test.go` 里的 `syncBuffer` 就是一个加锁的 `bytes.Buffer`（请求处理器在自己的 goroutine 里写日志，所以 buffer 必须带锁）：

```go
func TestRouteTableIsLogged(t *testing.T) {
	httpPort, grpcPort := freePort(t), freePort(t)
	cfg := writeConfig(t, httpPort, grpcPort)

	s := turbo.NewGrpcServer(nil, cfg)
	s.StartGrpcService(impl.RegisterServer)

	logged := &syncBuffer{}
	turbo.SetOutput(logged)
	defer turbo.SetOutput(os.Stdout) // 全局 logger，记得还原

	// 路由表是在 StartHTTPServer 里打印的，所以 SetOutput 必须在它之前。
	s.StartHTTPServer(component.GrpcClient, gen.GrpcSwitcher)
	defer s.Stop()
	waitReady(t, "http://127.0.0.1:"+httpPort+"/hello")

	require.Contains(t, logged.String(), "turbo: 1 route(s) registered")
	require.Contains(t, logged.String(), "route audit: 1 route(s)")
}
```

注意事项：

- `turbo.SetOutput` 改的是全局 logger，而且会把 formatter 换成默认 `TextFormatter`；并发测试或依赖 JSON 输出的测试要避开它。
- 一定要在测试结束前还原，否则后面所有测试的日志都进了你已经不再读的 buffer。
- 请求处理器在各自的 goroutine 里写日志，buffer 必须带锁，或者只在所有请求结束后再读。

## 拿 turbo 自己的测试当参考

`test/integration_test.go` 里这些测试值得照着写，右列是它们各自钉住的行为：

| 测试 | 验证什么 |
|---|---|
| `TestParameterBinding` | 绑定优先级 `injected > path > body > query/form`，JSON 请求不再忽略 URL |
| `TestPathWinsOverQueryWhateverTheSpelling` | 路由变量压过任何拼写的 query 参数 |
| `TestInvalidParameterIsRejected` | 参数存在但不可用时返回 400 `turbo: cannot bind ...`，且不回显原值 |
| `TestInterceptorChainRunsGlobalAndRouteLevel` | 公共拦截器先跑，路由自己的跟在后面，After 逆序 |
| `TestCommonInterceptorsSurviveAConfigReload` | 重载后公共拦截器不会丢 |
| `TestRouteAuditRefusesUnprotectedRoutes` | 声明 `auth.interceptors` 后，未鉴权路由启动被拒、重载保留旧配置 |
| `TestTruncatedConfigNeverBecomesTheRoutingTable` | 配置被截断成空文件时不断服务，日志出现 `ignoring configuration change` |
| `TestFailedConfigReloadKeepsServing` | 组件没注册、YAML 坏掉都不会打挂正在服务的进程 |
| `TestStoppedServerIgnoresConfigChanges` | `Stop()` 之后不再响应配置变更 |
| `TestRouteTableAndNotFoundAreObservable` | `route: ...` 路由表、`404 no route for ...`，且 query 不进日志 |
| `TestConfiguredLogLevelIsUsed` / `TestInvalidLogLevelIsRefused` | `config.log_level` 生效；非法值让启动 panic |
| `TestRequestsDuringReloadAreRaceFree` | 重载与请求并发无数据竞争，需要 `-race` |
| `TestErrorStatusCodes` | 带状态的错误用对应 HTTP 状态，平凡错误仍是 500 |

## make test 会做什么

`Makefile` 的 `test` 目标先跑 `go test -p=1 -cover -coverpkg github.com/vaporz/turbo github.com/vaporz/turbo github.com/vaporz/turbo/test`，再在 `test/testcreateservice` 和 `test/testservice` 两个目录里各跑一次 `go build ./...`。

- 只测这两个包，覆盖率统计的是 turbo 本身。
- `-p=1` 让包串行：测试里有全局 logger、全局 `switcherFunc`、固定端口和生成目录，串行才稳定。
- 编译那两个目录，是确认集成测试生成的代码仍然可构建。
- 它不带 `-race`；重载并发的测试注释明确要求 `-race`，本地排查时自己加 `go test -race -p=1 ./test/...`。
- `make test` 之外还有 `make tidy`（先删生成目录再 `go mod tidy`）和 `make vuln`（govulncheck，需要联网），都不属于测试目标。

## 相关阅读

- [11-binding.md](11-binding.md)：来源优先级与注入的完整语义
- [15-hot-reload.md](15-hot-reload.md)：热重载的行为与限制
- [14-logging.md](14-logging.md)：日志文本与级别
- [19-troubleshooting.md](19-troubleshooting.md)：测试失败时按症状排查
- [18-deployment.md](18-deployment.md)：CI 与构建机上的 Go 版本要求
