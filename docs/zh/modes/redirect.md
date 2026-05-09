# REDIRECT 模式

`REDIRECT` 模式使用 Linux netfilter REDIRECT 规则将 TCP 流量透明转发到 UA3F 监听端口。

## 适用场景

需要透明代理，但不需要或无法使用 TPROXY 路由能力时，可以使用 `REDIRECT`。

## 配置

```yaml
server-mode: REDIRECT
bind-address: 0.0.0.0
port: 1080
include-lan-routes: false
```

## 行为

- UA3F 通过 iptables 或 nftables 安装防火墙规则。
- 匹配的 TCP 流量会被重定向到本地监听端口。
- UA3F 恢复目标连接元数据后执行 HTTP 重写。

## 注意事项

`REDIRECT` 配置比 TPROXY 简单，但依赖平台提供原始目标地址恢复能力。
