# FINAL Rule

`FINAL` always matches.

```yaml
header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

Place `FINAL` at the end of a rule list as a fallback. Any rules after `FINAL` are normally unreachable unless `continue: true` is set on the `FINAL` rule.
