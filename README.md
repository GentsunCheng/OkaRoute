# OkaRoute — 基于 TOTP 的端口跳跃转发

OkaRoute 是一个使用共享 TOTP 密钥在指定端口范围内进行“端口跳跃”的转发代理。客户端与服务端按统一的时间步长计算出当前周期的端口，并在服务端同时监听“前一周期、当前周期、下一周期”三个端口，降低时间边界的连接失败率。支持同构转发（TCP→TCP，后续可扩展 UDP）。

## 特性
- 基于 TOTP 的端口选择，端口范围可配置
- 服务端三端口并行监听（prev/curr/next）减少切换抖动
- 握手鉴权：客户端首帧携带 `step`、`nonce` 与 `HMAC(token)`
- 同构转发：当前版本实现 TCP→TCP，后续扩展 UDP→UDP
- 多配置支持：
  - 服务端单进程多实例（多 goroutine）
  - 客户端单进程多本地代理（多 goroutine）
- 自动解析配置格式：JSON / YAML / TOML（按文件后缀识别）
- 端口冲突防护：
  - 服务端多路由 `port_range` 之间禁止交叉
  - 客户端多端点 `bind_ip:bind_port` 禁止重复
- 详细日志：来源地址、当前使用的隧道端口、步长、转发目标

## 安装
- 环境要求：
  - Go 1.22+
- 拉取依赖（已在构建时自动获取）：
  - `gopkg.in/yaml.v3`
  - `github.com/BurntSushi/toml`

构建：

```
go build ./...
```

## 快速开始
1) 启动一个示例目标服务（本地 HTTP）：

```
python3 -m http.server 8080
```

2) 选择并准备配置（JSON/YAML/TOML 均可），示例已在 `configs/` 提供。

3) 启动服务端：

```
go run ./cmd/server -config configs/server.yaml
```

4) 启动客户端：

```
go run ./cmd/client -config configs/client.toml
```

5) 验证访问：

```
curl http://127.0.0.1:10080/
```

终端中可看到客户端与服务端的日志，包含来源地址、使用的服务端端口与步长等信息。

## 配置说明
当前支持两种模式：单配置与多配置。程序会根据顶层结构自动识别。

- 字段摘要（服务端 ServerConfig）：
  - `name`：路由名称（用于日志标签，可选）
  - `listen_ip`：服务端监听 IP
  - `port_range`：`{ min, max }` 端口范围
  - `protocol`：`"tcp"`（当前版本）
  - `totp_secret`：Base32 密钥（服务端与客户端共享）
  - `step_seconds`：时间步长（如 30）
  - `skew_steps`：步长容忍窗口（如 1，允许前后一步）
  - `target_addr` / `target_port`：目标地址与端口
  - `allowed_client_ips`：来源 IP 白名单（预留，当前未强制）
  - `tls`：`{ enabled, cert_file, key_file }`（预留，可扩展）

- 字段摘要（客户端 ClientConfig）：
  - `name`：端点名称（用于日志标签，可选）
  - `server_host`：服务端主机名或 IP
  - `port_range`：与服务端一致的端口范围
  - `protocol`：`"tcp"`（当前版本）
  - `totp_secret`：Base32 密钥（与服务端一致）
  - `step_seconds` / `skew_steps`：与服务端一致的步长配置
  - `bind_ip` / `bind_port`：客户端本地代理监听地址与端口
  - `client_id`：客户端标识（参与 HMAC）
  - `tls`：`{ enabled, insecure_skip_verify }`（预留，可扩展）

### 单配置示例
- 服务端（YAML）：`configs/server.yaml`

```
name: "routeA"
listen_ip: "0.0.0.0"
port_range:
  min: 30000
  max: 30050
protocol: "tcp"
totp_secret: "JBSWY3DPEHPK3PXP"
step_seconds: 30
skew_steps: 1
target_addr: "127.0.0.1"
target_port: 8080
allowed_client_ips: []
tls:
  enabled: false
  cert_file: ""
  key_file: ""
```

- 客户端（TOML）：`configs/client.toml`

```
name = "endpointA"
server_host = "127.0.0.1"
protocol = "tcp"
totp_secret = "JBSWY3DPEHPK3PXP"
step_seconds = 30
skew_steps = 1
bind_ip = "127.0.0.1"
bind_port = 10080
client_id = "client"

[port_range]
min = 30000
max = 30050

[tls]
enabled = false
insecure_skip_verify = false
```

### 多配置示例
- 服务端（YAML，顶层 `routes`）：

```
routes:
  - name: "routeA"
    listen_ip: "0.0.0.0"
    port_range: { min: 30000, max: 30020 }
    protocol: "tcp"
    totp_secret: "JBSWY3DPEHPK3PXP"
    step_seconds: 30
    skew_steps: 1
    target_addr: "127.0.0.1"
    target_port: 8080
  - name: "routeB"
    listen_ip: "0.0.0.0"
    port_range: { min: 30030, max: 30050 }
    protocol: "tcp"
    totp_secret: "JBSWY3DPEHPK3PXP"
    step_seconds: 30
    skew_steps: 1
    target_addr: "127.0.0.1"
    target_port: 9090
```

- 客户端（TOML，顶层 `endpoints`）：

```
[[endpoints]]
name = "endpointA"
server_host = "127.0.0.1"
protocol = "tcp"
totp_secret = "JBSWY3DPEHPK3PXP"
step_seconds = 30
skew_steps = 1
bind_ip = "127.0.0.1"
bind_port = 10080

  [endpoints.port_range]
  min = 30000
  max = 30020

[[endpoints]]
name = "endpointB"
server_host = "127.0.0.1"
protocol = "tcp"
totp_secret = "JBSWY3DPEHPK3PXP"
step_seconds = 30
skew_steps = 1
bind_ip = "127.0.0.1"
bind_port = 10081

  [endpoints.port_range]
  min = 30030
  max = 30050
```

注意：
- 服务端多路由的 `port_range` 不得交叉，否则程序启动报错
- 客户端多端点的 `bind_ip:bind_port` 不得重复，否则程序启动报错

## 使用说明
- 启动方式：
  - 单配置：`go run ./cmd/server -config configs/server.yaml`
  - 多配置：同样使用 `-config` 指向包含 `routes`/`endpoints` 的文件，程序自动并发启动各实例
- 日志：
  - 服务端：`[routeName] 服务端启动/轮换/接受连接`，含来源、使用端口、step 与目标
  - 客户端：`[endpointName] 客户端本地监听/建立转发`，含来源、服务端主机、使用端口与 step

## 设计与限制
- 同构协议：为避免复杂性与脆弱性，转发协议需与目标协议一致（当前为 TCP）。
- 时间同步：建议保持客户端与服务端时间误差在步长内；`skew_steps` 缓解轻微漂移。
- 安全性：TOTP+HMAC 仅做同步与鉴权；需要保密时建议启用 TLS（代码已预留结构）。
- 防火墙与端口占用：务必提前开放端口范围并避免与其他服务冲突。

## 许可证
暂未指定许可证。如需开源许可证，请在本节添加选定的许可证内容。