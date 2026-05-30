# QUIC 阻断

QUIC 阻断会丢弃出站 UDP/443 数据包，用于阻止 QUIC 协议的流量。

```yaml
l3-rewrite:
  block-quic: true
```

在 netfilter 路径中，UA3F 通过 iptables 或 nftables 规则丢弃 UDP/443 流量。启用 eBPF 加速时，UA3F 选择的 TC 程序是 `block_quic`。
