# Desync 介绍

Desync 是 Linux-only 的 DPI 对抗功能，作用于早期 TCP 包，不属于 HTTP 重写流程。

UA3F Desync 是一种无服务器侧配合的 DPI 对抗方式，主要通过 TCP 分片乱序发射与 TCP 混淆注入影响部分 DPI 设备的流重组状态。它不改变目标服务器能力，也不要求远端部署 UA3F。

```yaml
desync:
  reorder: false
  reorder-bytes: 8
  reorder-packets: 1500
  inject: false
  inject-ttl: 3
  desync-ports: ""
```

当 `reorder` 或 `inject` 任一功能启用时，UA3F 会为 Desync 创建专用 netfilter 规则和 NFQUEUE worker。

`desync-ports` 可以用逗号分隔的目标端口列表限制 Desync 生效范围。

Desync 的效果依赖网络路径、DPI 实现和中间设备行为。开启后可能造成 TCP 连接建立后早期通信波动，建议先在目标线路上逐项测试，再扩大部署范围。
