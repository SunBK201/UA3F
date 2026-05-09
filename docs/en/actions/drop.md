# DROP Action

`DROP` drops the matched request or response in the rewrite pipeline.

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "drop.example.com"
    action: DROP
    rewrite-direction: REQUEST
```

`rewrite-direction` must be either `REQUEST` or `RESPONSE`.

Use it sparingly because clients may observe timeouts or connection failures depending on the service mode.
