# REPLACE 动作

`REPLACE` 将 Header 设置为新值。

```yaml
header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-direction: REQUEST
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

它适合完整替换 Header。需要部分替换时使用 `REPLACE-REGEX`。

`REPLACE` 仅可用于 `header-rewrite`。
