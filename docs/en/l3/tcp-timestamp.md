# TCP Timestamp

TCP Timestamp rewriting removes TCP Timestamp options from TCP packets.

```yaml
l3-rewrite:
  tcpts: true
```

UA3F removes `TCPOptionKindTimestamps` from parsed TCP options. In the non-eBPF path, packets are handled through NFQUEUE when this feature is enabled.

With eBPF acceleration enabled, UA3F selects the `clear_tcp_syn_ts` TC program.
