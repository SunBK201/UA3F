# FINAL 规则

`FINAL` 总是匹配。

```yaml
header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

通常将 `FINAL` 放在规则列表末尾作为兜底规则。除非设置 `continue: true`，否则 `FINAL` 后面的规则通常不会被执行。
