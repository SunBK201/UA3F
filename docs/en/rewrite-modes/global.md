# GLOBAL Rewrite Mode

`GLOBAL` mode rewrites User-Agent headers globally.

## Configuration

```yaml
rewrite-mode: GLOBAL
user-agent: "FFF"
user-agent-regex: ""
user-agent-partial-replace: false
```

## Behavior

- Every matched HTTP request is considered for User-Agent rewriting.
- `user-agent` is the replacement value.
- `user-agent-regex` limits which User-Agent values are rewritten. Empty means all values.
- `user-agent-partial-replace` replaces only the regex-matched part instead of the whole header.

## Notes

Use `GLOBAL` for simple deployments. Use `RULE` when different hosts, ports, or headers need different behavior.
