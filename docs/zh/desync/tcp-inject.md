# TCP 混淆注入

TCP 混淆注入会在 TCP 握手后发送一个 64 字节随机 payload 的低 TTL 包。

该混淆包用于干扰 DPI 设备的内容重组状态机。为了避免混淆包影响真实 TCP 通信，它的 TTL 会设置得较低，默认值为 `3`，使其预期在传输途中被丢弃，而不会到达目标服务器。

```yaml
desync:
  inject: true
  inject-ttl: 3
```

UA3F 会构造一个源/目标地址互换的原始 TCP 包，设置 `ACK`、`PSH` 标志，窗口为 `65535`，payload 为随机数据。

该包预期在传输途中过期。UA3F 会根据观测到的 TTL/Hop Limit 估算距离，避免配置的 `inject-ttl` 高于估算距离。

如果同时开启固定 TTL 重写，出站流量中可能同时存在正常 TTL 包和低 TTL 注入包。例如常规出站包 TTL 为 `64`，注入包 TTL 为 `3`。
