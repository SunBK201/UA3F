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
