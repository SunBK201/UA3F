# TTL

TTL 重写将 IPv4 Time To Live 设置为固定值。

```yaml
l3-rewrite:
  ttl: true
  ttl-value: 128
```

`ttl-value` 可设置为 `1` 到 `255`，默认值为 `64`。也可以通过命令行参数 `--l3-rewrite-ttl-value`，或者环境变量 `UA3F_L3_REWRITE_TTL_VALUE` 指定。netfilter 与 eBPF 加速路径都会使用相同的目标值。

TTL 重写适合在网关侧规范化出站包 TTL。

使用 netfilter 路径并开启软件流量卸载（flow offload）时，`ttl-value: 255` 无法保证出站 TTL 为 `255`。UA3F 在入口将 TTL 设置为目标值加一，以补偿转发时的递减；但 IPv4 TTL 最大为 `255`，无法设置为 `256`。因此，走卸载路径的转发包实际出站 TTL 为 `254`，走普通 POSTROUTING 路径的包仍为 `255`。需要统一出站 TTL 为 `255` 时，请关闭流量卸载。
