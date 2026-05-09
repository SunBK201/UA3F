# DEST-PORT Rule

`DEST-PORT` matches the destination port as a string.

```yaml
header-rewrite:
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT
```

Use it to bypass or isolate traffic classes. A common pattern is to direct SSH and other non-HTTP ports before a final rewrite rule.

The field name in YAML is still `match-value`, and the value should be quoted when written as YAML.
