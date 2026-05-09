# REDIRECT-302 Action

`REDIRECT-302` returns an HTTP `302 Found` response to the client.

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/old"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old(.*)"
    rewrite-value: "https://example.com/new$1"
```

This action only applies to `url-redirect` rules and always runs in the request direction.

Use it when clients are allowed to see and follow the redirect.
