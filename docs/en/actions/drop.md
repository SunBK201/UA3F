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

For ordinary HTTP requests in HTTP server mode, a `DROP` decision received by the server returns `503 Service Unavailable`, just like `REJECT`. In the REQUEST direction, the request is not sent upstream. In the RESPONSE direction, a local 503 response replaces the upstream response without forwarding its content.
