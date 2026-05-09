# L3 Rewrite

L3 rewrite modifies selected IP/TCP fields at the network layer. It is separate from HTTP Header and Body rewriting.

## Supported features

| Feature | Effect |
| --- | --- |
| TTL | Sets IPv4 TTL to a fixed value through firewall rules |
| IPID | Sets IPv4 Identification to `0` |
| TCP Timestamp | Removes TCP Timestamp options |
| TCP Initial Window | Sets TCP SYN window to `65535` |

## Configuration

```yaml
l3-rewrite:
  ttl: false
  ipid: false
  tcpts: false
  tcpwin: false
  bpf-offload: false
```

Legacy top-level fields such as `ttl`, `ipid`, `tcp_timestamp`, and `tcp_initial_window` are also merged into `l3-rewrite` for compatibility.

## Runtime path

Without eBPF acceleration, UA3F uses firewall rules and NFQUEUE where packet alteration is needed. With `l3-rewrite.bpf-offload: true`, UA3F attaches TC eBPF programs to eligible egress interfaces.
