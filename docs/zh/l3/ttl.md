# TTL

TTL 重写将 IPv4 Time To Live 设置为固定值。

```yaml
l3-rewrite:
  ttl: true
  ttl-value: 128
```

`ttl-value` 可设置为 `1` 到 `255`，默认值为 `64`。也可以通过命令行参数 `--ttl-value`，或者环境变量 `UA3F_L3_REWRITE_TTL_VALUE` 指定。netfilter 与 eBPF 加速路径都会使用相同的目标值。

TTL 重写适合在网关侧规范化出站包 TTL。
