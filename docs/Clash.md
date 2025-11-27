# Clash 配置

Clash 与 UA3F 的配置部署教程详见：[UA3F 与 Clash 从零开始的部署教程](https://sunbk201public.notion.site/UA3F-Clash-16d60a7b5f0e457a9ee97a3be7cbf557?pvs=4)

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

> [!IMPORTANT]
> HTTP/SOCKS5 模式下，如果 Clash 使用 Fake-IP 模式，确保 OpenClash 本地 DNS 劫持选择「使用防火墙转发」，不要使用「Dnsmasq 转发」。

## Clash 参考配置

<table>
  <tr>
    <th>版本</th>
    <th>配置文件</th>
    <th>UA3F 运行模式</th>
    <th>说明</th>
  </tr>
  <tr>
    <td>国内版</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-socks5-cn.yaml">ua3f-socks5-cn.yaml</a></td>
    <td>SOCKS5</td>
    <td>无需进行任何修改，可直接使用</td>
  </tr>
  <tr>
    <td>代理支持</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-socks5-global.yaml">ua3f-socks5-global.yaml</a></td>
    <td>SOCKS5</td>
    <td>注意需要在 proxy-providers > Global-ISP > url 中（第 23 行）加入你的代理订阅链接</td>
  </tr>
  <tr>
    <td>抗 DPI + 代理支持</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-socks5-global-dpi.yaml">ua3f-socks5-global-dpi.yaml</a></td>
    <td>SOCKS5</td>
    <td>注意需要在 proxy-providers > Global-ISP > url 中（第 23 行）加入你的代理订阅链接</td>
  </tr>
  <tr>
    <td>代理支持</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-tproxy-cn-dpi.yaml">ua3f-tproxy-cn-dpi.yaml</a></td>
    <td>TPROXY/REDIRECT/NFQUEUE</td>
    <td>注意需要在 proxy-providers > Global-ISP > url 中（第 13 行）加入你的代理订阅链接</td>
  </tr>
  <tr>
    <td>抗 DPI + 代理支持</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-tproxy-global-dpi.yaml">ua3f-tproxy-global-dpi.yaml</a></td>
    <td>TPROXY/REDIRECT/NFQUEUE</td>
    <td>注意需要在 proxy-providers > Global-ISP > url 中（第 18 行）加入你的代理订阅链接</td>
  </tr>
</table>
