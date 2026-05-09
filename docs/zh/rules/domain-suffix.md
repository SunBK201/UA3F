# DOMAIN-SUFFIX 规则

`DOMAIN-SUFFIX` 匹配以 `match-value` 结尾的 Host。

```yaml
header-rewrite:
  - type: DOMAIN-SUFFIX
    match-value: ".example.com"
    action: DIRECT
```

它适合对同一域名族应用策略，例如 `api.example.com` 与 `static.example.com`。

实现使用后缀匹配，因此需要谨慎选择后缀。`example.com` 既可能匹配根域名，也会匹配以该字符串结尾的主机。
