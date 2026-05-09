# REDIRECT-307 Action

`REDIRECT-307` returns an HTTP `307 Temporary Redirect` response to the client.

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/upload"
    action: REDIRECT-307
    rewrite-regex: "^http://example.com/upload(.*)"
    rewrite-value: "https://example.com/upload$1"
```

Use `307` when the client should preserve the original method and body semantics.

This action only applies to `url-redirect` rules.
