# DIRECT 动作

`DIRECT` 对匹配规则停止重写，并按原样转发流量。

```yaml
header-rewrite:
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT
```

它常用于在宽泛重写规则前配置例外。`DIRECT` 可用于 Header、Body 和 URL 重定向规则列表。
