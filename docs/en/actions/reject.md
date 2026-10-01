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

For ordinary HTTP requests in HTTP server mode, `REJECT` returns `503 Service Unavailable`. In the REQUEST direction, the request is not sent upstream. In the RESPONSE direction, a local 503 response replaces the upstream response without forwarding its content.
