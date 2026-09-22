# Turbo 使用教程

Turbo 是一个 **HTTP → gRPC / Thrift 网关框架**：你写好 gRPC 或 Thrift 服务，Turbo 按一份配置文件把
HTTP 请求翻译成 RPC 调用，再把结果写回 HTTP 响应。于是同一套 RPC 接口同时有了 HTTP API，浏览器、
小程序、第三方回调都能直接调用，而不必再为每个接口手写一层 HTTP handler。

本目录是面向使用者的完整教程，覆盖从"完全不了解 Turbo"到"知道每个配置键、每个组件、每条行为"的全部内容。

> 本文档对应 **v0.6.2**。涉及版本差异的地方会用
> `> **版本**：v0.6.2 起 …` 这样的提示标出；升级旧版本请看 [20-migration.md](20-migration.md)。
>
> 仓库根目录 `README.md` 里指向 `vaporz.github.io` 的链接是**旧版英文文档**，内容停留在 v0.5.x，
> 与本目录不一致时以本目录为准。

## 三条阅读路线

**A. 只想先跑起来（约 20 分钟）**

1. [02-getting-started.md](02-getting-started.md) —— 安装、`turbo create`、跑通第一个请求、加一个新 API
2. [03-service-yaml.md](03-service-yaml.md) —— 回头看懂 `service.yaml` 里每一行
3. [04-routing.md](04-routing.md) —— 路由与路径语法

**B. 完整通读（建议顺序）**

| # | 文档 | 讲什么 |
|---|---|---|
| 1 | [01-overview.md](01-overview.md) | Turbo 是什么、架构、一次请求的完整生命周期、术语表 |
| 2 | [02-getting-started.md](02-getting-started.md) | 安装、创建项目、跑起来、加 API |
| 3 | [03-service-yaml.md](03-service-yaml.md) | 配置逐键参考 |
| 4 | [04-routing.md](04-routing.md) | `urlmapping` 与路径语法 |
| 5 | [05-components.md](05-components.md) | 组件总览、执行顺序、单例与并发 |
| 6 | [06-interceptor.md](06-interceptor.md) | 拦截器：鉴权、注入身份、访问日志 |
| 7 | [07-preprocessor-postprocessor.md](07-preprocessor-postprocessor.md) | 请求前后处理 |
| 8 | [08-hijacker.md](08-hijacker.md) | 劫持请求，自己写响应 |
| 9 | [09-convertor.md](09-convertor.md) | 自定义结构体构造 |
| 10 | [10-errors.md](10-errors.md) | 错误与 HTTP 状态码 |
| 11 | [11-binding.md](11-binding.md) | 参数从哪来、谁优先、怎么注入 |
| 12 | [12-code-generation.md](12-code-generation.md) | CLI 与代码生成 |
| 13 | [13-grpc-thrift.md](13-grpc-thrift.md) | 两条链路的差异与多服务复用 |
| 14 | [14-logging.md](14-logging.md) | Turbo 自己的日志 |
| 15 | [15-hot-reload.md](15-hot-reload.md) | 热重载：重载什么、失败怎么办 |
| 16 | [16-auth-and-route-audit.md](16-auth-and-route-audit.md) | 鉴权声明与启动路由审计 |
| 17 | [17-testing.md](17-testing.md) | 怎么测自己的服务 |
| 18 | [18-deployment.md](18-deployment.md) | 部署与运维 |
| 19 | [19-troubleshooting.md](19-troubleshooting.md) | 排查手册 |
| 20 | [20-migration.md](20-migration.md) | 版本与行为变更、升级清单 |

**C. 当手册查**

- [appendix-cheatsheet.md](appendix-cheatsheet.md) —— 配置键、组件签名、绑定优先级、状态码、CLI、日志行速查
- [appendix-config-example.md](appendix-config-example.md) —— 一份注释齐全的 `service.yaml` 与配套注册代码
- [appendix-api-index.md](appendix-api-index.md) —— 导出 API 索引
- [19-troubleshooting.md](19-troubleshooting.md) —— 症状 → 原因 → 确认 → 修复

## Turbo 的一句话定位

Turbo **不是** RPC 框架，也**不做**业务逻辑：gRPC / Thrift 的服务端实现由你自己写，Turbo 只负责
「HTTP 请求 ⇄ RPC 调用」这一段，以及围绕它的配置、组件、热重载与日志。它适合这样的场景：

- 已经（或打算）用 gRPC / Thrift 组织服务，同时需要给浏览器、小程序、第三方回调提供 HTTP 接口；
- 接口数量多，希望"加一个 API"只改 proto/thrift 与一行配置，而不是再写一个 HTTP handler；
- 需要一个能在**不重启**的情况下改路由与组件的中间层。

如果你只需要一个纯 HTTP 服务，Turbo 只会增加一层；如果 RPC 服务本身已经暴露了 HTTP 网关（如 grpc-gateway），
也请先比较两者的差别再决定。

## 约定

- 文中所有配置键、API、CLI 参数都以仓库源码为准，不描述尚未实现的行为。
- 代码块标了语言；`yaml` 片段默认指 `service.yaml`（或 `service-local.yaml` 之类同结构文件）。
- 示例里的包路径用 `example.com/hello`、服务名用 `Hello` 占位，替换成你自己的即可。
