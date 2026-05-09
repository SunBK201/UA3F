# ADD Action

`ADD` adds a Header value.

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "example.com"
    action: ADD
    rewrite-direction: REQUEST
    rewrite-header: "X-UA3F"
    rewrite-value: "enabled"
```

Use `rewrite-header` for the Header name and `rewrite-value` for the value.

`ADD` is only available in `header-rewrite`.
