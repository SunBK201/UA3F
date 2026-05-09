# DEST-PORT 规则

`DEST-PORT` 按目标端口字符串匹配。

```yaml
header-rewrite:
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT
```

它常用于绕过或隔离某类流量，例如在兜底重写规则前先放行 SSH。

YAML 字段仍然是 `match-value`，端口建议写成字符串。
