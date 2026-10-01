# REJECT 动作

`REJECT` 在重写流程中拒绝匹配到的请求或响应。

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "blocked.example.com"
    action: REJECT
    rewrite-direction: REQUEST
```

`rewrite-direction` 必须是 `REQUEST` 或 `RESPONSE`。

当匹配后需要明确停止该流量时使用 `REJECT`。

HTTP 服务模式下的普通 HTTP 请求命中 `REJECT` 时，返回 `503 Service Unavailable`。REQUEST 方向不会向上游发送请求；RESPONSE 方向用本地 503 响应替代上游响应，不返回上游内容。
