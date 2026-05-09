# HEADER-KEYWORD 规则

`HEADER-KEYWORD` 在请求 Header 中包含关键字时匹配。

```yaml
header-rewrite:
  - type: HEADER-KEYWORD
    match-header: "User-Agent"
    match-value: "MicroMessenger"
    action: DIRECT
```

Header 值的关键字匹配不区分大小写。当前实现基于请求 Header。

它适合在更宽泛的 `FINAL` 规则前配置白名单或例外。
