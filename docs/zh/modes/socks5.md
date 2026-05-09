# SOCKS5 模式

`SOCKS5` 模式把 UA3F 作为 SOCKS5 代理运行。UA3F 接受 SOCKS5 客户端连接后，会嗅探流量是 HTTP、HTTPS 还是普通 TCP。

## 适用场景

当 UA3F 需要接在 Clash、浏览器或其他代理管理器后面时，优先使用 `SOCKS5`。

## 配置

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
```

## 行为

- TCP 流量通过 SOCKS5 握手进入 UA3F。
- HTTP 流量可以直接重写。
- HTTPS 流量需要启用 MitM 才能重写 Header 或 Body。
- 普通 TCP 流量只转发，不进行 HTTP 重写。

## 注意事项

SOCKS5 是与 Clash 伴生运行最简单的模式。UDP 不属于 UA3F 的 HTTP 重写流程。
