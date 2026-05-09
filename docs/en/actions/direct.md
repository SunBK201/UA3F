# DIRECT Action

`DIRECT` stops rewriting for the matched rule and forwards traffic as-is.

```yaml
header-rewrite:
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT
```

Use it for exceptions before broader rewrite rules. `DIRECT` can be used in Header, Body, and URL redirect rule lists.
