# 快速开始

UA3F 是一个 HTTP(S) 重写代理，可用于透明改写请求/响应 Header、Body，也可按规则执行 URL 重定向、拒绝或丢弃请求。除 HTTP 重写外，UA3F 还支持 L3 重写与 Desync：L3 重写可调整 TTL、IPID、TCP Timestamp、TCP 初始窗口等网络层特征，并可通过 TC eBPF 加速；Desync 可通过 TCP 分片乱序发射和低 TTL 混淆注入干扰部分 DPI 设备的流重组行为。

## 安装

### 使用 Release 包

前往 [GitHub Releases](https://github.com/SunBK201/UA3F/releases) 下载适合目标平台的二进制文件、opkg 包或 apk 包。

### 使用 Docker

以 SOCKS5 代理模式启动：

```sh
docker run -p 1080:1080 sunbk201/ua3f -f FFF
```

### 从源码构建

```sh
git clone https://github.com/SunBK201/UA3F.git
cd UA3F/src
go build -o ua3f main.go
```

## 第一次运行

使用默认配置启动：

```sh
ua3f
```

默认监听 `127.0.0.1:1080`，服务模式为 `SOCKS5`，重写策略为 `GLOBAL`。

指定配置文件启动：

```sh
ua3f -c /path/to/config.yaml
```

生成模板配置文件：

```sh
ua3f -g
```

## 常见部署路径

| 场景 | 推荐方式 |
| --- | --- |
| 只需要本机或上游代理接入 | `SOCKS5` 或 `HTTP` |
| OpenWrt 透明代理 | `TPROXY` 或 `REDIRECT` |
| 需要网络层队列处理 | `NFQUEUE` |
| 与 Clash 共存 | UA3F 使用 `SOCKS5`，Clash 将 HTTP/TCP 流量转发到 UA3F |
| 需要 HTTPS Header/Body 重写 | 启用 `mitm` 并配置目标主机名 |
| 需要网络层特征重写 | 启用 L3 重写，按需配置 TTL、IPID、TCP Timestamp 或 TCP 初始窗口 |
| 需要 DPI 流重组干扰 | 启用 Desync，按需配置 TCP 分片乱序发射或 TCP 混淆注入 |

## 下一步

- 阅读 [核心概念](./concepts.md) 了解模式、策略与规则。
- 阅读 [配置说明](./configuration.md) 编写 YAML 配置。
- 阅读 [HTTP 重写](/zh/http-rewrite/rewrite-modes.md)、[L3 重写](/zh/l3/overview.md) 与 [Desync](/zh/desync/overview.md) 了解完整功能。
- 查看 [API 文档](/zh/api/) 接入状态查询与日志读取。
