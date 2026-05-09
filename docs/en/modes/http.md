# HTTP Mode

`HTTP` mode runs UA3F as a standard HTTP proxy. Clients send absolute-form HTTP requests to UA3F, and HTTPS traffic enters through the `CONNECT` tunnel path.

## When to use it

Use `HTTP` mode when the client application can explicitly configure an HTTP proxy. It is also useful for tests because no netfilter rules are required.

## Configuration

```yaml
server-mode: HTTP
bind-address: 127.0.0.1
port: 1080
```

## Behavior

- Plain HTTP requests are parsed and passed through the rewrite pipeline.
- `CONNECT` creates a TCP tunnel. HTTPS contents are only rewritten when MitM is enabled for the target hostname.
- Works with `GLOBAL`, `DIRECT`, and `RULE` rewrite modes.

## Notes

`HTTP` mode does not install firewall rules. Routing traffic to UA3F is the client's responsibility.
