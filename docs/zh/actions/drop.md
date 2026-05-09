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
