# HEADER-REGEX 规则

`HEADER-REGEX` 使用正则表达式匹配请求 Header。

```yaml
header-rewrite:
  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Windows|Android|iPhone)"
    action: REPLACE-REGEX
    rewrite-header: "User-Agent"
    rewrite-regex: "(Windows|Android|iPhone)"
    rewrite-value: "UA3F"
```

UA3F 会以不区分大小写的方式编译匹配正则。表达式无效时会记录日志，该规则不会匹配。

当关键字匹配过于粗略时，使用 `HEADER-REGEX`。
