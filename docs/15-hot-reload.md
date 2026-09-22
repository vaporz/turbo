# 热重载

Turbo 会监听配置文件，把它当成"运行中的路由表 + 组件表"来用：改完保存，不重启进程也能生效。
这篇讲清**什么会重载、什么必须重启、失败时会发生什么**，以及怎么安全地改生产配置。

## 1. 触发方式

Turbo 用 viper 的 `WatchConfig`（底层是操作系统的文件变更通知）监听 `service.yaml` 这个**文件**：

- 保存文件（任何写入事件）就会触发一次重载尝试；
- 事件密集时（例如编辑器先写临时文件再改名），可能触发多次，后面几次通常会因为"内容和当前一致"而
  直接完成一次无害的重载；
- 某些挂载（网络盘、WSL 下的 DrvFs 目录）可能**不产生事件**，此时保存不会触发重载——生产环境请把配置
  放在本地盘上。

进程启动时 Turbo 会打印一行说明，明确告诉你哪些键不参与重载：

```
turbo: a configuration change reloads urlmapping, components and filter_proto_json;
http_port, grpc_service_port, thrift_service_port, environment, turbo_log_path and log_level need a restart
```

## 2. 什么会重载、什么必须重启

| 会被重载（保存即生效） | 需要重启进程 |
|---|---|
| `urlmapping`（路由表） | `config.http_port` |
| `interceptor` / `preprocessor` / `postprocessor` / `hijacker` / `convertor` | `config.grpc_service_port`、`config.thrift_service_port` |
| `errorhandler` | `config.environment`（决定日志格式与输出） |
| `config.filter_proto_json` 及其两个子项 | `config.turbo_log_path`、`config.log_level` |
| `json_field_names` | `config.file_root_path`、`config.package_path`（只影响代码生成） |
| `auth.interceptors`、`auth.public_routes` | |

各键的含义见 [03-service-yaml.md](03-service-yaml.md)。

## 3. 一次重载做了什么

保存文件后，Turbo 依次：

1. **等文件稳定**（见第 5 节），然后完整读一遍配置文件并校验；读不动或校验不过就走第 4 节的失败分支。
2. **重建组件**：按新的组件声明阶段表重新装配拦截器、前后处理器、劫持器、转换器与错误处理器
   （`SetCommonInterceptor` 装的全局拦截器会保留，见 [05-components.md](05-components.md)）。
3. **重建路由表**，并**重新跑一次路由审计**（`auth.interceptors` 声明的鉴权规则同样生效，违规会拒绝这次重载）。
4. 两样都成功后，**一次性替换**正在生效的组件与路由表，日志打印 `Configuration reloaded`。

替换是原子的：一个请求要么完整地用旧表，要么完整地用新表，不会出现"路由是新的、组件是旧的"。
请求只在**查找 handler** 时短暂持有读锁，拿到之后就开始处理，因此一次重载最多让新请求多等一次重建的时间
（通常几百微秒到几毫秒）。

## 4. 重载失败会怎样

**进程不会退出，正在生效的配置继续服务**。日志分两种，含义不同：

| 日志 | 含义 | 常见原因 |
|---|---|---|
| `turbo: ignoring configuration change, it cannot be loaded: <err>` | 配置文件本身读不出来或校验不过 | YAML 语法错、`urlmapping` 为空、`json_field_names` / `log_level` 取值非法 |
| `turbo: configuration reload failed, keeping the running configuration: <err>` | 配置读进来了，但**装配不出来** | 组件名拼错/没注册、路由审计拒绝（有一条路由没有声明的鉴权拦截器） |

两种情况下，你在**启动**时做同样的事会得到相反的结果：启动阶段配置不合法会直接 `panic`，进程起不来
（快速失败，避免"起来了但没配置正确"）。

> **版本**：v0.6.2 起，**没有任何路由的配置会被拒绝**。以前一份"0 条路由"的配置是合法的，它会把路由表
> 整体替换成空——进程还活着、端口还在听，但每个请求都是 404。这类配置只可能来自"文件被读到一半"，
> 所以现在直接拒绝。

## 5. 写文件的方式：为什么 Turbo 要等一等

保存一个文件通常是**就地写**：写者先把文件截断成 0 字节，再写入新内容。文件变更通知会在截断时先到一次，
如果那一刻就去读，读到的是**空文件或半个文件**。

v0.6.2 起 Turbo 在读之前先等文件"不再变化"：

- 判据是**连续两次读到的内容逐字节相同**，且距上一次变化已过去 **100ms**（每 20ms 看一次）；
- 最多等 **3 秒**，到点就用当时的内容走正常校验（校验不过就按第 4 节处理）；
- **空内容不算"已稳定"**，会继续等——远端拷贝在打开文件到写出第一块之间有网络往返，把它当稳定就白等了。

> **版本**：这套静默期是 v0.6.2 引入的。在 v0.6.1 及更早版本，就地写配置有概率把正在服务的路由表
> 清空，所以升级前的常规做法是"一次性覆盖写"（写入临时文件再 `mv`，或用不会先截断的工具）。

即使有了这层保护，**改配置的正确姿势仍然是**：

```bash
# 1) 先写临时文件，内容完整后再替换（原子替换，事件只来一次）
cp service.yaml.new service.yaml     # 或 mv，视你的发布流程而定

# 2) 看一眼日志确认结果
tail -n 20 /opt/soft/log/yourservice/turbo.log   # 或 stderr

# 3) 用一条只读路由验证服务仍然正常
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8081/hello
```

第 3 步很关键：**进程活着不等于路由表是对的**。一条 200 的健康检查比"看进程在不在"可靠得多。

## 6. 和 `Stop()` 的关系

调用 `s.Stop()` 之后：

- 重载循环退出，配置文件再变也不会重载（日志里不会再出现 `Reloading configuration...`）；
- 正在处理的请求走完，HTTP / RPC 服务关闭。

> 同一个进程里先后启动过多个 server 时，早期启动的**文件监听器不会被拆除**（它们已经不再重载任何东西，
> 但仍在监听）。写测试时请给每个 server 一份独立的配置文件，见 [17-testing.md](17-testing.md)。

## 7. 什么时候**必须**重启

- 改了端口、`environment`、`turbo_log_path`、`log_level`、`file_root_path`；
- **新增了 RPC 方法**：`gen/grpcswitcher.go` 是编译进二进制的代码，必须 `turbo generate` + 重新编译 + 重启；
- 升级了 turbo 自身（依赖版本变了）。

给**已存在**的方法改路由、改组件、改 `auth` 声明，都不需要重启。

## 相关阅读

- [03-service-yaml.md](03-service-yaml.md) —— 每个键是否参与重载
- [16-auth-and-route-audit.md](16-auth-and-route-audit.md) —— 审计也会在重载时拒绝配置
- [14-logging.md](14-logging.md) —— 上面这些日志行的级别与含义
- [19-troubleshooting.md](19-troubleshooting.md) —— "改了没生效"怎么查
