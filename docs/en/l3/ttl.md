# TTL

TTL rewriting sets IPv4 Time To Live to a fixed value.

```yaml
l3-rewrite:
  ttl: true
  ttl-value: 128
```

`ttl-value` accepts values from `1` to `255` and defaults to `64`. It can also be set with the `--ttl-value` command-line flag or the `UA3F_L3_REWRITE_TTL_VALUE` environment variable. The netfilter and eBPF acceleration paths use the same target value.

TTL rewriting is useful when the gateway should normalize outgoing packet TTL values.
