# TCP Obfuscation Injection

When TCP injection is enabled, UA3F sends a low-TTL packet carrying a 64-byte random payload to the server upon receiving the server's `SYN+ACK`, provided the TTL check passes. Injection occurs during the handshake, without waiting for the three-way handshake to complete.

The injected packet is intended to disturb DPI stream reconstruction state. To avoid affecting the real TCP conversation, it uses a low TTL, `3` by default, so it is expected to expire in transit instead of reaching the origin server.

```yaml
desync:
  inject: true
  inject-ttl: 3
```

UA3F swaps the source and destination IP addresses and ports of the received server `SYN+ACK` to build a raw TCP packet in the client → server direction. The packet sets the `ACK` and `PSH` flags and a window of `65535`. Its sequence number is the acknowledgment number of the `SYN+ACK`, and its acknowledgment number is the sequence number of the `SYN+ACK` plus `1`.

The 64-byte random payload is generated when the injection feature starts and reused for subsequent injected packets.

The packet is intended to expire in transit. UA3F checks the observed TTL/Hop Limit before injecting so the configured `inject-ttl` is not higher than the estimated distance.

If fixed TTL rewriting is also enabled, outbound traffic may contain both normal TTL packets and low-TTL injected packets. For example, regular outbound packets may use TTL `64`, while injected packets use TTL `3`.
