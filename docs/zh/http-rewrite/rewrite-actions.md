# 重写动作

重写动作定义规则匹配后 UA3F 要执行的行为。

## Header 动作

以下动作可用于 `header-rewrite`。

| 动作 | 行为 |
| --- | --- |
| `DIRECT` | 停止重写并按原样转发 |
| `DELETE` | 删除 `rewrite-header` 指定的 Header |
| `ADD` | 添加 `rewrite-header`，值为 `rewrite-value` |
| `REPLACE` | 将 `rewrite-header` 设置为 `rewrite-value` |
| `REPLACE-REGEX` | 替换 `rewrite-header` 中被 `rewrite-regex` 匹配的部分 |
| `REJECT` | 拒绝匹配到的请求或响应 |
| `DROP` | 丢弃匹配到的请求或响应 |

示例：

```yaml
header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-direction: REQUEST
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

## Body 动作

Body 重写当前支持正则替换。

| 动作 | 行为 |
| --- | --- |
| `DIRECT` | 停止重写并按原样转发 |
| `REPLACE-REGEX` | 替换 Body 中被 `rewrite-regex` 匹配的内容 |
| `REJECT` | 拒绝匹配到的请求或响应 |
| `DROP` | 丢弃匹配到的请求或响应 |

示例：

```yaml
body-rewrite:
  - type: URL-REGEX
    match-value: "^http://ua-check.stagoh.com"
    action: REPLACE-REGEX
    rewrite-direction: RESPONSE
    rewrite-regex: "UA2F"
    rewrite-value: "UA3F"
```

## URL 重定向动作

以下动作可用于 `url-redirect`。

| 动作 | 行为 |
| --- | --- |
| `DIRECT` | 停止重定向处理 |
| `REDIRECT-302` | 向客户端返回 `302 Found` |
| `REDIRECT-307` | 向客户端返回 `307 Temporary Redirect` |
| `REDIRECT-HEADER` | 尽可能通过改写 URL/Host 处理跳转，而不返回重定向响应 |

示例：

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/"
    action: REDIRECT-HEADER
    rewrite-regex: "^http://example.com/(.*)"
    rewrite-value: "http://mirror.example.com/$1"
```

`REDIRECT-HEADER` 在 Host 不变时直接更新请求。Host 改变时，UA3F 会自行发送改写后的请求，并把响应写回客户端。
