# IPID

IPID 重写将 IPv4 Identification 字段设置为 `0`。

```yaml
l3-rewrite:
  ipid: true
```

在 NFQUEUE 路径中，UA3F 解析 IPv4 包并清零 `Id` 字段。IPv6 没有 IPv4 ID 字段，因此不会被修改。

启用 eBPF 加速时，UA3F 会选择 `set_ip_id_zero` TC 程序。
