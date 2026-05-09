# TTL

TTL 重写将 IPv4 Time To Live 设置为固定值。

```yaml
l3-rewrite:
  ttl: true
```

在 netfilter 路径中，UA3F 通过 iptables 或 nftables 规则将 TTL 设置为 `64`。启用 eBPF 加速时，选择的 TC 程序为 `set_ip_ttl`。

TTL 重写适合在网关侧规范化出站包 TTL。
