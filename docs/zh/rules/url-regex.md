# URL-REGEX 规则

`URL-REGEX` 匹配完整请求 URL。

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/old"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old(.*)"
    rewrite-value: "https://example.com/new$1"
```

它最常用于 `url-redirect`，也可以用于 Body 规则，对特定 URL 的响应内容进行改写。

当规则只应匹配 URL 前缀时，建议使用 `^` 等锚点。
