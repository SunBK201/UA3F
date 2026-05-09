# DIRECT Rewrite Mode

`DIRECT` mode forwards traffic without rewriting it.

## Configuration

```yaml
rewrite-mode: DIRECT
```

## Behavior

- HTTP parsing and forwarding still happen where the selected service mode requires them.
- No Header, Body, URL, or User-Agent rewrite action is applied.

## Notes

Use this mode to verify routing and proxy behavior before enabling rewrite rules.
