# DOMAIN 规则

`DOMAIN` 精确匹配请求 Host。

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "example.com"
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

该规则会将 `match-value` 与 UA3F 解析出的 Host 元数据做等值比较。它适合只针对单个主机名配置行为。

`DOMAIN` 不匹配子域名。需要匹配子域名时使用 `DOMAIN-SUFFIX`。
