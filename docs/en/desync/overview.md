# Desync

Desync is a Linux-only DPI evasion feature that manipulates early TCP packets outside the HTTP rewrite pipeline.

UA3F Desync is a serverless DPI evasion mechanism. It mainly uses TCP segment reordering and TCP obfuscation injection to disturb stream reassembly on some DPI devices. It does not require UA3F or any cooperating component on the remote server.

```yaml
desync:
  reorder: false
  reorder-bytes: 8
  reorder-packets: 1500
  inject: false
  inject-ttl: 3
  desync-ports: ""
```

When either `reorder` or `inject` is enabled, UA3F creates dedicated netfilter rules and NFQUEUE workers for Desync traffic.

`desync-ports` can restrict Desync to a comma-separated list of destination ports.

Desync effectiveness depends on the network path, DPI implementation, and middlebox behavior. It can cause short-lived instability at the beginning of a TCP connection, so test each option on the target route before broad deployment.
