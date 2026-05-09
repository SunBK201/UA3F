# HEADER-KEYWORD Rule

`HEADER-KEYWORD` matches when a request header contains a keyword.

```yaml
header-rewrite:
  - type: HEADER-KEYWORD
    match-header: "User-Agent"
    match-value: "MicroMessenger"
    action: DIRECT
```

Header keyword matching is case-insensitive for the header value. It is currently based on request headers.

Use it for allowlist or exception rules before a broader `FINAL` rule.
