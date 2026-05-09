# REDIRECT-307 动作

`REDIRECT-307` 向客户端返回 HTTP `307 Temporary Redirect` 响应。

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/upload"
    action: REDIRECT-307
    rewrite-regex: "^http://example.com/upload(.*)"
    rewrite-value: "https://example.com/upload$1"
```

当客户端需要保留原始请求方法和请求体语义时使用 `307`。

该动作只用于 `url-redirect`。
