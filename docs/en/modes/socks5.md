# SOCKS5 Mode

`SOCKS5` mode runs UA3F as a SOCKS5 proxy. It accepts TCP connections from SOCKS5 clients and then sniffs whether the stream is HTTP, HTTPS, or generic TCP.

## When to use it

Use `SOCKS5` when UA3F is chained behind Clash, a browser, or another proxy manager.

## Configuration

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
```

## Behavior

- TCP streams are accepted through the SOCKS5 handshake.
- HTTP streams can be rewritten directly.
- HTTPS streams require MitM before Header or Body rewriting is possible.
- Generic TCP streams are forwarded without HTTP rewriting.

## Notes

SOCKS5 mode is usually the easiest mode for coexistence with Clash. UDP is not part of UA3F's HTTP rewriting pipeline.
