# TTL

TTL rewriting sets IPv4 Time To Live to a fixed value.

```yaml
l3-rewrite:
  ttl: true
  ttl-value: 128
```

`ttl-value` accepts values from `1` to `255` and defaults to `64`. It can also be set with the `--l3-rewrite-ttl-value` command-line flag or the `UA3F_L3_REWRITE_TTL_VALUE` environment variable. The netfilter and eBPF acceleration paths use the same target value.

TTL rewriting is useful when the gateway should normalize outgoing packet TTL values.

When using the netfilter path with software flow offload enabled, `ttl-value: 255` cannot guarantee an outgoing TTL of `255`. UA3F sets the ingress TTL to the target value plus one to compensate for the forwarding decrement, but IPv4 TTL is limited to `255` and cannot be set to `256`. As a result, forwarded packets on the offload path leave with TTL `254`, while packets on the normal POSTROUTING path still leave with TTL `255`. Disable flow offload if all outgoing packets must have TTL `255`.
