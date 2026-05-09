# HTTP 模式

`HTTP` 模式把 UA3F 作为标准 HTTP 代理运行。客户端将 absolute-form HTTP 请求发送给 UA3F，HTTPS 流量通过 `CONNECT` 隧道进入。

## 适用场景

当客户端应用可以显式配置 HTTP 代理时使用 `HTTP` 模式。它不需要 netfilter 规则，也适合本地测试。

## 配置

```yaml
server-mode: HTTP
bind-address: 127.0.0.1
port: 1080
```

## 行为

- 明文 HTTP 请求会进入重写流程。
- `CONNECT` 会建立 TCP 隧道。HTTPS 内容只有在目标主机启用 MitM 后才能被重写。
- 支持 `GLOBAL`、`DIRECT`、`RULE` 三种重写模式。

## 注意事项

`HTTP` 模式不会安装防火墙规则，流量需要由客户端主动配置到 UA3F。
