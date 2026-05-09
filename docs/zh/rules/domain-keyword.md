# DOMAIN-KEYWORD 规则

`DOMAIN-KEYWORD` 在 Host 中包含 `match-value` 时匹配。

```yaml
header-rewrite:
  - type: DOMAIN-KEYWORD
    match-value: "httpbin"
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

它适合精确域名和后缀匹配都过于严格的场景。

关键字匹配是简单子串匹配。生产规则中如需精确控制，优先使用 `DOMAIN` 或 `DOMAIN-SUFFIX`。
