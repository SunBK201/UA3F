# TPROXY 模式

`TPROXY` 模式使用 Linux netfilter TPROXY 规则透明接管 TCP 流量，同时保留原始目标地址。

## 适用场景

在 Linux 或 OpenWrt 网关上，如果客户端不方便配置代理，并且 UA3F 需要看到原始目标地址，可以使用 `TPROXY`。

## 配置

```yaml
server-mode: TPROXY
bind-address: 0.0.0.0
port: 1080
include-lan-routes: false
```

## 行为

- UA3F 通过 iptables 或 nftables 安装防火墙规则。
- TCP 流量无需客户端配置代理即可进入 UA3F。
- 原始目标地址会进入连接元数据，供规则匹配使用。

## 注意事项

该模式需要 Linux netfilter 支持和足够权限，主要面向路由器或网关部署。
