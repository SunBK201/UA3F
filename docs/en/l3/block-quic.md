# QUIC Block

QUIC blocking drops outbound UDP/443 packets to prevent QUIC protocol traffic.

```yaml
l3-rewrite:
  block-quic: true
```

In the netfilter path, UA3F drops UDP/443 traffic with iptables or nftables rules. When eBPF acceleration is enabled, the selected TC program is `block_quic`.
