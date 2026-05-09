# NFQUEUE Mode

`NFQUEUE` mode processes packets through Linux netfilter queueing. It is designed for network-layer rewriting scenarios and compatibility with UA2F-style packet processing.

## When to use it

Use `NFQUEUE` when traffic should be inspected and modified at the packet layer instead of through a proxy socket.

## Configuration

```yaml
server-mode: NFQUEUE
rewrite-mode: GLOBAL
```

## Behavior

- Netfilter sends selected TCP packets into an NFQUEUE worker.
- UA3F detects HTTP payloads and rewrites User-Agent data in packet payloads.
- Non-HTTP or unsupported packets are accepted without modification.

## Notes

NFQUEUE mode is Linux-only and has a larger compatibility surface than socket proxy modes. Prefer HTTP, SOCKS5, TPROXY, or REDIRECT unless packet-level handling is required.
