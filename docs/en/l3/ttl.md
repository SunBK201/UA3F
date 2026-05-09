# TTL

TTL rewriting sets IPv4 Time To Live to a fixed value.

```yaml
l3-rewrite:
  ttl: true
```

In the netfilter path, UA3F installs iptables or nftables rules to set TTL to `64`. When eBPF acceleration is enabled, the selected TC program is `set_ip_ttl`.

TTL rewriting is useful when the gateway should normalize outgoing packet TTL values.
