# DELETE Action

`DELETE` removes a Header from the request or response.

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "example.com"
    action: DELETE
    rewrite-direction: REQUEST
    rewrite-header: "X-Debug"
```

`rewrite-header` is required. `rewrite-direction` defaults to `REQUEST` when omitted.

`DELETE` is only available in `header-rewrite`.
