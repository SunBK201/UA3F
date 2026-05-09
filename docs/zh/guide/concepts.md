# 核心概念

## 服务模式

UA3F 通过服务模式决定如何接收流量。

| 模式 | 工作方式 | 典型用途 |
| --- | --- | --- |
| `HTTP` | HTTP 代理 | 应用显式配置 HTTP 代理 |
| `SOCKS5` | SOCKS5 代理 | 与 Clash、浏览器或其他代理链路配合 |
| `TPROXY` | netfilter TPROXY | Linux/OpenWrt 透明代理，保留原始目标地址 |
| `REDIRECT` | netfilter REDIRECT | Linux/OpenWrt 透明代理，配置相对简单 |
| `NFQUEUE` | netfilter NFQUEUE | 网络层队列处理，兼容 UA2F 类场景 |

## 重写策略

| 策略 | 行为 |
| --- | --- |
| `GLOBAL` | 所有请求统一改写 `User-Agent` |
| `DIRECT` | 只转发，不重写 |
| `RULE` | 按 `header-rewrite`、`body-rewrite`、`url-redirect` 规则匹配并执行动作 |

生产配置通常使用 `RULE`，这样可以为特定域名、Header、端口或 URL 配置差异化行为。

## 规则匹配

规则从上到下执行。匹配后默认停止继续匹配；设置 `continue: true` 后会继续检查后续规则。

常用匹配类型：

| 类型 | 说明 |
| --- | --- |
| `DOMAIN` / `DOMAIN-SUFFIX` / `DOMAIN-KEYWORD` | 按目标域名匹配 |
| `DOMAIN-SET` | 按域名集合匹配 |
| `IP-CIDR` / `SRC-IP` | 按目标或来源 IP 匹配 |
| `DEST-PORT` | 按目标端口匹配 |
| `HEADER-KEYWORD` / `HEADER-REGEX` | 按 Header 内容匹配 |
| `URL-REGEX` | 按完整 URL 正则匹配 |
| `FINAL` | 兜底规则 |

## 规则动作

| 动作 | 说明 |
| --- | --- |
| `DIRECT` | 直接放行，不重写 |
| `REPLACE` | 替换指定 Header |
| `REPLACE-REGEX` | 替换匹配正则的部分 |
| `ADD` | 添加 Header |
| `DELETE` | 删除 Header |
| `REJECT` | 拒绝请求 |
| `DROP` | 丢弃请求 |
| `REDIRECT-302` / `REDIRECT-307` | 返回 HTTP 重定向 |
| `REDIRECT-HEADER` | 修改请求 Header 完成无感重定向 |

## HTTPS MitM

普通代理只能直接读取明文 HTTP。要改写 HTTPS 的 Header 或 Body，需要启用 `mitm` 并配置目标主机名和 CA。客户端必须信任对应 CA，否则 TLS 握手会失败。

建议仅对确实需要改写的域名启用 MitM，避免扩大证书信任范围。
