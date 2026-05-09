# DOMAIN-SUFFIX Rule

`DOMAIN-SUFFIX` matches hosts that end with `match-value`.

```yaml
header-rewrite:
  - type: DOMAIN-SUFFIX
    match-value: ".example.com"
    action: DIRECT
```

It is useful for applying one policy to a domain family such as `api.example.com` and `static.example.com`.

Because the implementation uses suffix matching, choose the suffix carefully. `example.com` can match both `example.com` and any host ending with that string.
