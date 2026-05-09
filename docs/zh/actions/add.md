# ADD 动作

`ADD` 添加 Header 值。

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "example.com"
    action: ADD
    rewrite-direction: REQUEST
    rewrite-header: "X-UA3F"
    rewrite-value: "enabled"
```

`rewrite-header` 是 Header 名称，`rewrite-value` 是 Header 值。

`ADD` 仅可用于 `header-rewrite`。
