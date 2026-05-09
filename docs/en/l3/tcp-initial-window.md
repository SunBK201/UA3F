# TCP Initial Window

TCP Initial Window rewriting sets the TCP window size on initial SYN packets.

```yaml
l3-rewrite:
  tcpwin: true
```

UA3F only changes packets where `SYN` is set and `ACK` is not set. The target window value is `65535`.

With eBPF acceleration enabled, UA3F selects the `set_tcp_syn_window` TC program.
