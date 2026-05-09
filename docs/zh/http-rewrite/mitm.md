# HTTPS MitM

MitM 让 UA3F 可以解密指定 HTTPS 连接，使 Header 和 Body 重写规则能够作用于 HTTPS 流量。

未启用 MitM 时，UA3F 仍然可以转发 HTTPS 流量，但加密后的请求和响应内容不会进入可见的重写流程。

## 何时启用 MitM

只有在需要对特定主机名的 HTTPS 请求或响应数据进行重写时，才建议启用 MitM。

典型场景：

- 重写 HTTPS 请求 Header，例如 `User-Agent`。
- 重写 HTTPS 响应 Header。
- 对 HTTPS 响应应用 Body 重写规则。
- 针对已知 HTTPS 端点测试重写行为。

## 配置

```yaml
mitm:
  enabled: true
  hostname: "*.httpbin.com, example.com:8000"
  insecure-skip-verify: false
  ca-passphrase: ""
  ca-p12-base64: ""
```

| 字段 | 说明 |
| --- | --- |
| `enabled` | 启用 HTTPS MitM |
| `hostname` | 逗号分隔的主机名白名单；支持通配符 `*` 和 `:port` 后缀 |
| `insecure-skip-verify` | 跳过上游服务器证书校验 |
| `ca-passphrase` | CA PKCS#12 数据的密码 |
| `ca-p12-base64` | Base64 编码的 CA PKCS#12 数据 |

## 主机名范围

`hostname` 控制哪些 HTTPS 目标会被拦截。建议保持范围尽可能小。

示例：

```yaml
mitm:
  enabled: true
  hostname: "example.com, *.httpbin.com, api.example.com:8443"
```

通配符适合域名族匹配。`:port` 后缀可将拦截限制到指定端口。

## 客户端信任

客户端必须信任 UA3F 使用的 CA。否则 HTTPS 连接会在 TLS 证书校验阶段失败。

生产环境建议为 UA3F 生成并分发专用 CA，不要复用范围过大的系统或组织 CA。

## 重写示例

```yaml
server-mode: SOCKS5
rewrite-mode: RULE

mitm:
  enabled: true
  hostname: "*.httpbin.org"

header-rewrite:
  - type: DOMAIN-SUFFIX
    match-value: "httpbin.org"
    action: REPLACE
    rewrite-direction: REQUEST
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

## 安全注意事项

MitM 会扩大信任边界，因为 UA3F 会终止客户端 TLS，并另外建立上游 TLS 连接。只对确实需要重写的主机名启用 MitM，妥善保护 CA 材料，并避免在非调试场景使用 `insecure-skip-verify`。
