# DOMAIN Rule

`DOMAIN` matches the exact request host.

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "example.com"
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

The matcher compares `match-value` with UA3F's parsed host metadata. Use it when one exact hostname should receive a specific action.

`DOMAIN` does not match subdomains. Use `DOMAIN-SUFFIX` for that.
