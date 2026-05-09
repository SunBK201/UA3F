# IP-CIDR 规则

`IP-CIDR` 将远端目标 IP 与 CIDR 网段匹配。

```yaml
header-rewrite:
  - type: IP-CIDR
    match-value: "203.0.113.0/24"
    action: DIRECT
```

如果 `match-value` 没有前缀长度，UA3F 会将其作为单个 IPv4 主机处理，并追加 `/32`。

该匹配依赖连接元数据，适合 UA3F 能识别远端地址的代理模式。
