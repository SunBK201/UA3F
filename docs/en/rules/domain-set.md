# DOMAIN-SET Rule

`DOMAIN-SET` loads a domain list from a local file path or remote HTTP(S) URL and matches hosts by suffix.

```yaml
header-rewrite:
  - type: DOMAIN-SET
    match-value: "/etc/ua3f/domain-set.txt"
    action: DIRECT
```

Domain set files are parsed line by line. Empty lines and lines beginning with `#` are ignored.

The set is loaded asynchronously during rule initialization. If the source cannot be loaded, the rule remains present but has no loaded domains to match.
