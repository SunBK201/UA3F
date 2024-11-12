# UA3F

UA3F 是下一代 HTTP User-Agent 修改方法，对外作为一个 SOCK5 服务，可以部署在路由器等设备等设备进行透明 UA 修改。

## 特性

- 支持正则表达式规则匹配修改 User-Agent
- 自定义 User-Agent 内容
- 与其他网络加速代理工具共存
- LRU 高速缓存非 HTTP 域名，加速非 HTTP 流量转发
- 支持 LuCI Web 图形页面
- 多种部署方式
- 支持 UDP 转发

![UA3F](https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f)

## 部署

提供 3 种部署方式：

1. 使用 ipk 安装包进行部署：

[Release](https://github.com/SunBK201/UA3F/releases) 页面已经提供常见架构的编译版本，可以根据自己架构下载并解压到路由器等设备上。

2. 使用安装/升级脚本进行部署：

```sh
opkg update
opkg install curl libcurl luci-compat
export url='https://blog.sunbk201.site/cdn' && sh -c "$(curl -kfsSl $url/install.sh)"
service ua3f reload
```

3. OpenWrt 编译安装

```sh
git clone https://github.com/openwrt/openwrt.git && cd openwrt
git checkout openwrt-22.03
./scripts/feeds update -a && ./scripts/feeds install -a
git clone https://github.com/SunBK201/UA3F.git package/UA3F
make menuconfig # 勾选 Network->Web Servers/Proxies->ua3f
make download -j$(nproc) V=s
make -j$(nproc) || make -j1 || make -j1 V=sc # make package/UA3F/openwrt/compile -j1 V=sc # 编译单个包
```

## 使用

UA3F 支持 LuCI Web 页面，可以打开 Services -> UA3F 进行相关配置。

![UA3F-LuCI](https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-luci)

> [!NOTE]
> 设置说明：
>
> - Port: UA3F 监听端口，默认 `1080`。
> - Bind Address: UA3F 监听地址，默认 `127.0.0.1`。
> - Log Level: 日志等级，默认 `info`, 如果需要调试排查错误可以设置为 `debug`。
> - User-Agent: 自定义 User-Agent，默认 `FFF`。
> - User-Agent Regex Pattern: User-Agent 正则表达式规则。如果流量中的 User-Agent 匹配该正则表达式，则会被修改为 User-Agent 字段的内容，否则不会被修改；如果该字段为空，则所有流量 User-Agent 都会被修改。默认 `(iPhone|iPad|Android|Macintosh|Windows|Linux)`，即只修改携带设备与系统信息的 User-Agent。
> - Partial Replace: 部分替换，如果开启，则只替换 User-Agent Regex Pattern 中匹配到的部分。该选项仅在 User-Agent Regex Pattern 不为空时生效。

### 手动命令行启动

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

- `-b <bind addr>`: 自定义绑定监听地址，默认 127.0.0.1
- `-p <port>`: 端口号，默认 1080
- `-l <log level>`: 日志等级，默认 info，可选：debug，默认日志位置：`/var/log/ua3f.log`
- `-f <UA>`: 自定义 UA，默认 FFF
- `-r <regex>`: 自定义正则匹配 User-Agent, 默认 `(iPhone|iPad|Android|Macintosh|Windows|Linux)`
- `-s`: 部分替换，仅替换正则匹配到的部分

### Clash 配置建议

Clash 与 UA3F 的配置部署教程详见：[UA3F 与 Clash 从零开始的部署教程](https://sunbk201public.notion.site/UA3F-Clash-16d60a7b5f0e457a9ee97a3be7cbf557?pvs=4)

- Clash 需要选用 Meta 内核。
- 请确保 `PROCESS-NAME,ua3f,DIRECT` 置于规则列表顶部，`MATCH,ua3f` 置于规则列表底部。
- 可以在 `PROCESS-NAME,ua3f,DIRECT` 与 `MATCH,ua3f` 之间按需加入自定义加密代理规则。如果上述 2 条规则之间加入 DIRECT 规则，请确保匹配该规则的流量属于非 HTTP 协议流量。

```yaml
proxies:
  - name: "ua3f"
    type: socks5
    server: 127.0.0.1
    port: 1080
    url: http://connectivitycheck.platform.hicloud.com/generate_204
    udp: false

rules:
  - PROCESS-NAME,ua3f,DIRECT
  - NETWORK,udp,DIRECT
  - MATCH,ua3f
```

请不要将从 [Release](https://github.com/SunBK201/UA3F/releases) 下载解压得到的 `ua3f` 二进制文件修改名称，
如需修改，则需要在 `PROCESS-NAME,ua3f,DIRECT` 中修改相应的名称。

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
