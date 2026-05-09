# REDIRECT Mode

`REDIRECT` mode uses Linux netfilter REDIRECT rules to transparently send TCP traffic to the UA3F listener.

## When to use it

Use `REDIRECT` when a transparent proxy setup is needed but TPROXY routing is not required or not available.

## Configuration

```yaml
server-mode: REDIRECT
bind-address: 0.0.0.0
port: 1080
include-lan-routes: false
```

## Behavior

- UA3F installs firewall rules through iptables or nftables.
- Matching TCP traffic is redirected to the local listener.
- HTTP rewriting works after UA3F reconstructs the target connection metadata.

## Notes

`REDIRECT` is simpler than TPROXY, but it depends on platform support for recovering the original destination.
