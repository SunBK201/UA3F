# HEADER-REGEX Rule

`HEADER-REGEX` matches a request header with a regular expression.

```yaml
header-rewrite:
  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Windows|Android|iPhone)"
    action: REPLACE-REGEX
    rewrite-header: "User-Agent"
    rewrite-regex: "(Windows|Android|iPhone)"
    rewrite-value: "UA3F"
```

UA3F compiles the matcher with case-insensitive behavior. Invalid expressions are logged and the rule will not match.

Use `HEADER-REGEX` when keyword matching is too broad.
