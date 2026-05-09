# SRC-IP Rule

`SRC-IP` matches the client source IP address against a CIDR range.

```yaml
header-rewrite:
  - type: SRC-IP
    match-value: "192.168.1.0/24"
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

If no prefix length is provided, UA3F treats the value as a single IPv4 host with `/32`.

Use it to apply different policies to different LAN clients.
