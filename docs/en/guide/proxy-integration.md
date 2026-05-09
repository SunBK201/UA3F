# Working with Other Proxies

UA3F can run as a local processing node before or after other proxy tools. A common setup is to run UA3F in `SOCKS5` or `HTTP` mode and let Clash, gateway firewall rules, or another proxy client forward selected TCP/HTTP traffic to UA3F.

## Common topologies

### Clash forwards to UA3F

This is the simplest setup for desktop Clash deployments. Clash handles rule routing and upstream proxies, while UA3F handles HTTP rewrite, MitM, L3 rewrite, or Desync.

```text
Client -> Clash -> UA3F -> Direct/Upstream
```

UA3F configuration:

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

Clash snippet:

```yaml
proxies:
  - name: "ua3f"
    type: socks5
    server: 127.0.0.1
    port: 1080
    url: http://connectivitycheck.platform.hicloud.com/generate_204
    udp: false

rules:
  - NETWORK,udp,DIRECT
  - MATCH,ua3f
```

### UA3F handles transparent traffic first

When UA3F runs on a gateway, it can use `TPROXY`, `REDIRECT`, or `NFQUEUE` mode to capture traffic before applying rewrite and network-layer processing.

```yaml
server-mode: TPROXY
bind-address: 0.0.0.0
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

This setup must be paired with system firewall rules that send target traffic into UA3F's listen port. Linux gateway deployments are usually the better fit when TCP-layer processing, L3 rewrite, or Desync is needed.

## Clash guidance

| Scenario | Recommended setup | Notes |
| --- | --- | --- |
| HTTP rewrite only | Clash -> UA3F `SOCKS5` | Clash handles routing; UA3F handles Header/Body/URL rewrite |
| HTTPS Header/Body rewrite | Clash -> UA3F `SOCKS5` + MitM | Enable MitM only for hostnames that need rewriting |
| Transparent gateway capture | UA3F `TPROXY` or `REDIRECT` | Requires firewall forwarding rules |
| L3 rewrite or Desync | UA3F gateway mode | Network-layer features fit Linux gateways better |
| Upstream proxy subscriptions | Clash proxy-provider | Clash keeps subscriptions; UA3F acts as the local processing node |

## Reference configurations

These files live in the UA3F repository under `configs/clash` and can be downloaded and adjusted as needed.

| Variant | File | UA3F mode | Notes |
| --- | --- | --- | --- |
| China only | [ua3f-socks5-cn.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-socks5-cn.yaml) | `SOCKS5` | Ready to use |
| Proxy support | [ua3f-socks5-global.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-socks5-global.yaml) | `SOCKS5` | Add your subscription URL under `proxy-providers > Global-ISP > url` |
| DPI evasion with proxy support | [ua3f-socks5-global-dpi.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-socks5-global-dpi.yaml) | `SOCKS5` | Add your subscription URL under `proxy-providers > Global-ISP > url` |
| TProxy with proxy support | [ua3f-tproxy-cn-dpi.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-tproxy-cn-dpi.yaml) | `TPROXY` / `REDIRECT` / `NFQUEUE` | Add your subscription URL under `proxy-providers > Global-ISP > url` |
| TProxy DPI evasion with proxy support | [ua3f-tproxy-global-dpi.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-tproxy-global-dpi.yaml) | `TPROXY` / `REDIRECT` / `NFQUEUE` | Add your subscription URL under `proxy-providers > Global-ISP > url` |

## Troubleshooting

- Ensure the UA3F listen address and port match the `server` and `port` values in Clash.
- If UA3F and Clash do not share the same network namespace, do not use `127.0.0.1`; use a reachable address instead.
- UDP usually does not go through UA3F's HTTP rewrite flow, so `NETWORK,udp,DIRECT` is often appropriate in Clash.
- When MitM is enabled, clients must trust the UA3F CA and `mitm.hostname` must match the target hostname.
- When L3 rewrite or Desync is enabled, verify kernel support, firewall rules, and permissions first.
