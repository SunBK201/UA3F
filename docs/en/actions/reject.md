# REJECT Action

`REJECT` rejects the matched request or response in the rewrite pipeline.

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "blocked.example.com"
    action: REJECT
    rewrite-direction: REQUEST
```

`rewrite-direction` must be either `REQUEST` or `RESPONSE`.

Use `REJECT` when the flow should stop explicitly after a match.
