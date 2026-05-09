# SRC-IP 规则

`SRC-IP` 将客户端源 IP 与 CIDR 网段匹配。

```yaml
header-rewrite:
  - type: SRC-IP
    match-value: "192.168.1.0/24"
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

如果未提供前缀长度，UA3F 会按单个 IPv4 主机 `/32` 处理。

它适合为不同 LAN 客户端配置不同策略。
