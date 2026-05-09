# 匹配规则

匹配规则决定重写动作何时执行。规则会放在以下列表之一：

```yaml
header-rewrite: []
body-rewrite: []
url-redirect: []
```

常见字段：

| 字段 | 说明 |
| --- | --- |
| `type` | 匹配规则类型 |
| `match-value` | 匹配值 |
| `match-header` | Header 匹配规则使用的 Header 名称 |
| `action` | 匹配后执行的重写动作 |
| `continue` | 匹配后是否继续执行后续规则 |

## DOMAIN

`DOMAIN` 精确匹配解析出的请求 Host。

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "example.com"
    action: DIRECT
```

它不会匹配子域名。

## DOMAIN-SUFFIX

`DOMAIN-SUFFIX` 匹配以 `match-value` 结尾的 Host。

```yaml
header-rewrite:
  - type: DOMAIN-SUFFIX
    match-value: "example.com"
    action: DIRECT
```

它适合匹配同一域名族，例如 `example.com`、`api.example.com`、`static.example.com`。

## DOMAIN-KEYWORD

`DOMAIN-KEYWORD` 匹配包含 `match-value` 的 Host。

```yaml
header-rewrite:
  - type: DOMAIN-KEYWORD
    match-value: "example"
    action: DIRECT
```

如果子串匹配过于宽泛，应优先使用 `DOMAIN` 或 `DOMAIN-SUFFIX`。

## DOMAIN-SET

`DOMAIN-SET` 从本地文件路径或远程 HTTP(S) URL 加载域名列表，并按后缀匹配解析出的请求 Host。

```yaml
header-rewrite:
  - type: DOMAIN-SET
    match-value: "/etc/ua3f/domain-set.txt"
    action: DIRECT
```

域名集文件按行解析，空行和以 `#` 开头的行会被忽略。域名集会在规则初始化时异步加载。

## IP-CIDR

`IP-CIDR` 将远端目标 IP 与 CIDR 网段匹配。

```yaml
header-rewrite:
  - type: IP-CIDR
    match-value: "203.0.113.0/24"
    action: DIRECT
```

如果值中没有前缀长度，UA3F 会将其视为单个 IPv4 主机并追加 `/32`。

## SRC-IP

`SRC-IP` 将客户端源 IP 与 CIDR 网段匹配。

```yaml
header-rewrite:
  - type: SRC-IP
    match-value: "192.168.1.0/24"
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

它适合为不同 LAN 客户端配置不同策略。

## DEST-PORT

`DEST-PORT` 按目标端口字符串匹配。

```yaml
header-rewrite:
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT
```

端口建议在 YAML 中加引号，保持字符串形式。

## HEADER-KEYWORD

`HEADER-KEYWORD` 在请求 Header 包含关键字时匹配。Header 值匹配不区分大小写。

```yaml
header-rewrite:
  - type: HEADER-KEYWORD
    match-header: "User-Agent"
    match-value: "MicroMessenger"
    action: DIRECT
```

## HEADER-REGEX

`HEADER-REGEX` 使用不区分大小写的正则表达式匹配请求 Header。

```yaml
header-rewrite:
  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Windows|Android|iPhone)"
    action: REPLACE-REGEX
    rewrite-header: "User-Agent"
    rewrite-regex: "(Windows|Android|iPhone)"
    rewrite-value: "UA3F"
```

表达式无效时会记录日志，该规则不会匹配。

## URL-REGEX

`URL-REGEX` 使用正则表达式匹配完整请求 URL。

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/old"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old(.*)"
    rewrite-value: "https://example.com/new$1"
```

当规则只应匹配 URL 前缀时，建议使用 `^` 等锚点。

## FINAL

`FINAL` 总是匹配，通常放在规则列表末尾作为兜底规则。

```yaml
header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

具体规则应放在宽泛规则之前。如果 `FINAL` 出现在其他规则之前，除非设置 `continue: true`，否则后续规则通常不会执行。
