# DOMAIN-KEYWORD Rule

`DOMAIN-KEYWORD` matches when the parsed host contains `match-value`.

```yaml
header-rewrite:
  - type: DOMAIN-KEYWORD
    match-value: "httpbin"
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

Use it for broad matching when exact host or suffix matching is too strict.

Keyword matching is simple substring matching. Prefer `DOMAIN` or `DOMAIN-SUFFIX` for precise production rules.
