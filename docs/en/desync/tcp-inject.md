# TCP Obfuscation Injection

TCP injection sends a 64-byte random payload packet after the TCP handshake with a low TTL.

The injected packet is intended to disturb DPI stream reconstruction state. To avoid affecting the real TCP conversation, it uses a low TTL, `3` by default, so it is expected to expire in transit instead of reaching the origin server.

```yaml
desync:
  inject: true
  inject-ttl: 3
```

UA3F builds a raw TCP packet with swapped source and destination addresses, `ACK` and `PSH` flags, a `65535` window, and random payload.

The packet is intended to expire in transit. UA3F checks the observed TTL/Hop Limit before injecting so the configured `inject-ttl` is not higher than the estimated distance.

If fixed TTL rewriting is also enabled, outbound traffic may contain both normal TTL packets and low-TTL injected packets. For example, regular outbound packets may use TTL `64`, while injected packets use TTL `3`.
