# 与其他代理配合

UA3F 可以作为其他代理工具前后的一个本地处理节点。常见方式是让 UA3F 以 `SOCKS5` 或 `HTTP` 模式监听本地端口，再由 Clash、透明网关规则或其他代理客户端把需要处理的 TCP/HTTP 流量转发到 UA3F。

## 典型拓扑

### Clash 转发到 UA3F

这种方式最简单，适合桌面端 Clash 环境。Clash 负责规则分流和上游代理，UA3F 负责 HTTP 重写、MitM、L3 重写或 Desync。

```text
Client -> Clash -> UA3F -> Direct/Upstream
```

UA3F 配置：

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

Clash 配置片段：

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

### UA3F 透明接管后转发

如果 UA3F 运行在网关上，也可以使用 `TPROXY`、`REDIRECT` 或 `NFQUEUE` 模式先接管流量，再按配置执行重写和网络层处理。

```yaml
server-mode: TPROXY
bind-address: 0.0.0.0
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

这种方式需要配合系统防火墙规则，把目标流量导入 UA3F 的监听端口。需要 TCP 层处理、L3 重写或 Desync 时，优先在 Linux 网关环境中部署。

## 与 Clash 配合建议

| 场景 | 推荐方式 | 说明 |
| --- | --- | --- |
| 只需要 HTTP 重写 | Clash -> UA3F `SOCKS5` | Clash 负责分流，UA3F 负责 Header/Body/URL 重写 |
| 需要 HTTPS Header/Body 重写 | Clash -> UA3F `SOCKS5` + MitM | 只对需要重写的主机名开启 MitM |
| 需要透明网关接管 | UA3F `TPROXY` 或 `REDIRECT` | 需要配合防火墙转发规则 |
| 需要 L3 重写或 Desync | UA3F 网关模式 | Linux 网关环境更适合部署网络层功能 |
| 需要上游代理订阅 | 使用 Clash proxy-provider | Clash 维护订阅，UA3F 作为本地处理节点 |

## 参考配置

以下配置文件位于 UA3F 仓库的 `configs/clash` 目录，可以直接下载后按需调整。

| 版本 | 配置文件 | UA3F 运行模式 | 说明 |
| --- | --- | --- | --- |
| 国内版 | [ua3f-socks5-cn.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-socks5-cn.yaml) | `SOCKS5` | 无需进行任何修改，可直接使用 |
| 代理支持 | [ua3f-socks5-global.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-socks5-global.yaml) | `SOCKS5` | 在 `proxy-providers > Global-ISP > url` 中加入代理订阅链接 |
| 抗 DPI + 代理支持 | [ua3f-socks5-global-dpi.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-socks5-global-dpi.yaml) | `SOCKS5` | 在 `proxy-providers > Global-ISP > url` 中加入代理订阅链接 |
| TProxy 代理支持 | [ua3f-tproxy-cn-dpi.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-tproxy-cn-dpi.yaml) | `TPROXY` / `REDIRECT` / `NFQUEUE` | 在 `proxy-providers > Global-ISP > url` 中加入代理订阅链接 |
| TProxy 抗 DPI + 代理支持 | [ua3f-tproxy-global-dpi.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/configs/clash/ua3f-tproxy-global-dpi.yaml) | `TPROXY` / `REDIRECT` / `NFQUEUE` | 在 `proxy-providers > Global-ISP > url` 中加入代理订阅链接 |

## 排查要点

- 确认 UA3F 监听地址和端口与 Clash 中的 `server`、`port` 一致。
- 如果 UA3F 和 Clash 不在同一网络命名空间中，不要使用 `127.0.0.1`，应改为可互通的地址。
- UDP 通常不经过 UA3F 的 HTTP 重写流程，Clash 中可按需保持 `NETWORK,udp,DIRECT`。
- 启用 MitM 时，客户端必须信任 UA3F CA，并且 `mitm.hostname` 必须匹配目标主机名。
- 启用 L3 重写或 Desync 时，优先确认内核、防火墙和权限是否满足要求。
