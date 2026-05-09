# TCP 初始窗口

TCP 初始窗口重写会修改初始 SYN 包的 TCP Window。

```yaml
l3-rewrite:
  tcpwin: true
```

UA3F 只修改 `SYN` 为真且 `ACK` 为假的包，目标窗口值为 `65535`。

启用 eBPF 加速时，UA3F 会选择 `set_tcp_syn_window` TC 程序。
