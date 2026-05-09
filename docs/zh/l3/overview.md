# L3 重写介绍

L3 重写会修改选定的 IP/TCP 字段，独立于 HTTP Header 和 Body 重写。

## 支持功能

| 功能 | 效果 |
| --- | --- |
| TTL | 通过防火墙规则将 IPv4 TTL 设置为固定值 |
| IPID | 将 IPv4 Identification 设置为 `0` |
| TCP 时间戳 | 删除 TCP Timestamp 选项 |
| TCP 初始窗口 | 将 TCP SYN 窗口设置为 `65535` |

## 配置

```yaml
l3-rewrite:
  ttl: false
  ipid: false
  tcpts: false
  tcpwin: false
  bpf-offload: false
```

兼容字段 `ttl`、`ipid`、`tcp_timestamp`、`tcp_initial_window` 也会合并进 `l3-rewrite`。

## 运行路径

未启用 eBPF 加速时，UA3F 使用防火墙规则，并在需要修改包内容时使用 NFQUEUE。启用 `l3-rewrite.bpf-offload: true` 后，UA3F 会在符合条件的出口网卡上挂载 TC eBPF 程序。
