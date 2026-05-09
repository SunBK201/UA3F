# REDIRECT-302 动作

`REDIRECT-302` 向客户端返回 HTTP `302 Found` 响应。

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/old"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old(.*)"
    rewrite-value: "https://example.com/new$1"
```

该动作只用于 `url-redirect`，并且始终在请求方向执行。

当允许客户端看到并跟随跳转时使用它。
