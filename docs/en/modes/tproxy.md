# TPROXY Mode

`TPROXY` mode uses Linux netfilter TPROXY rules to transparently intercept TCP traffic while preserving the original destination address.

## When to use it

Use `TPROXY` on Linux or OpenWrt gateways when clients should not configure an explicit proxy and the original destination address must remain visible to UA3F.

## Configuration

```yaml
server-mode: TPROXY
bind-address: 0.0.0.0
port: 1080
include-lan-routes: false
```

## Behavior

- UA3F installs firewall rules through iptables or nftables.
- TCP traffic is redirected to UA3F without changing client proxy settings.
- The original destination is used by the connection metadata and rule matchers.

## Notes

This mode requires Linux netfilter support and sufficient privileges. It is intended for router or gateway deployments.
