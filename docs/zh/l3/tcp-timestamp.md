# TCP 时间戳

TCP 时间戳重写会删除 TCP Timestamp 选项。

```yaml
l3-rewrite:
  tcpts: true
```

UA3F 会从解析出的 TCP options 中移除 `TCPOptionKindTimestamps`。在非 eBPF 路径中，启用该功能时相关包会通过 NFQUEUE 处理。

启用 eBPF 加速时，UA3F 会选择 `clear_tcp_syn_ts` TC 程序。
