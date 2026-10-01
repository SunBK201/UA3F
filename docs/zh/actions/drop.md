# DROP 动作

`DROP` 在重写流程中丢弃匹配到的请求或响应。

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "drop.example.com"
    action: DROP
    rewrite-direction: REQUEST
```

`rewrite-direction` 必须是 `REQUEST` 或 `RESPONSE`。

该动作可能让客户端表现为超时或连接失败，应谨慎使用。

HTTP 服务模式处理普通 HTTP 请求时，服务层收到 `DROP` 决定后，与 `REJECT` 一样返回 `503 Service Unavailable`。REQUEST 方向不会向上游发送请求；RESPONSE 方向用本地 503 响应替代上游响应，不返回上游内容。
