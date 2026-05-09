# REDIRECT-HEADER 动作

`REDIRECT-HEADER` 尽可能通过改写请求 URL 和 Host 处理跳转，而不是向客户端返回重定向响应。

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/"
    action: REDIRECT-HEADER
    rewrite-regex: "^http://example.com/(.*)"
    rewrite-value: "http://mirror.example.com/$1"
```

如果改写后的 URL 仍是同一 Host，UA3F 会更新请求 URL 并继续转发。如果 Host 发生变化，UA3F 会自行发送改写后的请求，并把响应写回客户端。

在 Linux 上，内部请求会设置 socket mark，避免再次被 UA3F 自己的防火墙规则捕获。
