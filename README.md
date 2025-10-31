# UA3F

UA3F 是下一代 HTTP User-Agent 重写工具，作为一个 SOCKS5/TPROXY/REDIRECT 服务部署在路由器等设备进行透明 User-Agent 重写。

## 特性

- 支持多种服务模式：SOCKS5、TPROXY、REDIRECT
- 支持正则表达式规则匹配重写 User-Agent
- 自定义 User-Agent 内容
- 与其他网络加速代理工具共存
- LRU 高速缓存非 HTTP 域名，加速非 HTTP 流量转发
- 支持 LuCI Web 图形页面
- 支持 opkg 安装、编译安装、Docker 部署多种方式
- 兼容 OpenWrt 17.01 及以上版本

<table>
  <tr>
    <td><img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-luci.png" alt="UA3F-LuCI"></td>
    <td><img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-stat.png" alt="UA3F-Statistics"></td>
  </tr>
</table>

![UA3F](https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f.png)

## 部署

提供 3 种部署方式：

- 使用 ipk 安装包进行部署：

  [Release](https://github.com/SunBK201/UA3F/releases) 页面已经提供常见架构的编译版本，可以根据自己设备的架构下载到 OpenWrt 上使用 `opkg install` 进行安装。

- OpenWrt 编译安装：

  ```sh
  git clone https://github.com/openwrt/openwrt.git && cd openwrt
  git checkout openwrt-22.03
  ./scripts/feeds update -a && ./scripts/feeds install -a
  git clone https://github.com/SunBK201/UA3F.git package/UA3F
  make menuconfig # 勾选 Network->Web Servers/Proxies->ua3f
  make download -j$(nproc) V=s
  make -j$(nproc) || make -j1 || make -j1 V=sc # make package/UA3F/openwrt/compile -j1 V=sc # 编译单个包
  ```

- Docker 部署：

  ```sh
  docker run -p 1080:1080 sunbk201/ua3f -f FFF
  ```

## 使用

UA3F 支持 LuCI Web 页面，可以打开 Services -> UA3F 进行相关配置。

`TPROXY` 与 `REDIRECT` 模式不依赖 Clash 等 SOCKS5 客户端，UA3F 可以独立工作（不保证与 Clash 的兼容性）。

> [!NOTE]
> 设置说明：
>
> - Server Mode (服务模式): 支持 `SOCKS5`、`TPROXY`、`REDIRECT` 三种模式，默认 `SOCKS5`
> - Port (监听端口): 默认 `1080`
> - Bind Address (绑定地址): 默认 `127.0.0.1`
> - Log Level (日志等级): 默认 `info`, 如果需要调试排查错误可以设置为 `debug`
> - User-Agent (自定义重写 User-Agent): 默认 `FFF`
> - User-Agent Regex (User-Agent 正则表达式): 只重写匹配成功的 User-Agent。如果为空，全部重写
> - Partial Replace (部分替换): 只替换正则表达式匹配的部分。该选项仅在 User-Agent 正则表达式非空时生效

设备与系统信息正则表达式参考：

```regex
(Apple|iPhone|iPad|Macintosh|Mac OS X|Mac|Darwin|Microsoft|Windows|Linux|Android|OpenHarmony|Mobile|Dalvik)
```

<details>
<summary>手动命令行启动</summary>

```sh
sudo -u nobody /usr/bin/ua3f
```

shellclash/shellcrash 用户建议使用以下命令启动:

```sh
sudo -u shellclash /usr/bin/ua3f
# 如果上面命令报错执行下面该命令
sudo -u shellcrash /usr/bin/ua3f
```

相关命令行启动参数:

- `-m <mode>`: 服务模式，支持 SOCKS5、TPROXY、REDIRECT，默认 SOCKS5
- `-b <bind addr>`: 自定义绑定监听地址，默认 127.0.0.1
- `-p <port>`: 端口号，默认 1080
- `-l <log level>`: 日志等级，默认 info，可选：debug，默认日志位置：`/var/log/ua3f.log`
- `-f <UA>`: 自定义 UA，默认 FFF
- `-r <regex>`: 自定义正则匹配 User-Agent, 默认为空, 表示所有 User-Agent 都会被重写
- `-s`: 部分替换，仅替换正则匹配到的部分
</details>

## Clash 配置

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
> 如果使用 Fake-IP 模式，确保 OpenClash 本地 DNS 劫持选择「使用防火墙转发」，不要使用「Dnsmasq 转发」。

### Clash 参考配置

提供 3 个参考配置：

1. 国内版，无需进行任何修改，可直接使用 [ua3f-cn.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-cn.yaml) (Clash 需要选用 Meta 内核。)
2. 国际版，针对有特定需求的特殊用户进行适配，[ua3f-global.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-global.yaml)，注意需要在 proxy-providers > Global-ISP > url 中（第 23 行）加入你的代理节点订阅链接。(Clash 需要选用 Meta 内核。)
3. 国际版(增强)，针对流量特征检测 (DPI) 进行规则补充，注意该配置会对 QQ、微信等平台的流量进行分流代理，因此需要根据自己的需求谨慎选择该配置，[ua3f-global-enhance.yaml](https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-global-enhance.yaml)，注意需要在 proxy-providers > Global-ISP > url 中（第 23 行）加入你的代理节点订阅链接。(Clash 需要选用 Meta 内核。)

## Extra

> [!TIP]
> 使用 nftables 固定 TTL 为 64：
>
> ```sh
> nft add table inet ttl64
> nft add chain inet ttl64 postrouting { type filter hook postrouting priority -150\; policy accept\; }
> nft add rule inet ttl64 postrouting counter ip ttl set 64
> ```

> [!TIP]
> 使用 iptables 固定 TTL 为 64：
>
> ```sh
> iptables -t mangle -A POSTROUTING -j TTL --ttl-set 64
> ```
