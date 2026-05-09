# IPID

IPID rewriting sets the IPv4 Identification field to `0`.

```yaml
l3-rewrite:
  ipid: true
```

In the NFQUEUE path, UA3F parses IPv4 packets and zeroes the `Id` field. IPv6 packets are not modified because they do not have the IPv4 ID field.

With eBPF acceleration enabled, UA3F selects the `set_ip_id_zero` TC program.
