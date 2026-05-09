# IP-CIDR Rule

`IP-CIDR` matches the remote destination IP address against a CIDR range.

```yaml
header-rewrite:
  - type: IP-CIDR
    match-value: "203.0.113.0/24"
    action: DIRECT
```

If `match-value` does not include a prefix length, UA3F treats it as a single IPv4 host by appending `/32`.

This matcher depends on connection metadata. It is best suited to proxy modes where UA3F can identify the remote endpoint.
