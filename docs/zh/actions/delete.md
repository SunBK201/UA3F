# DELETE 动作

`DELETE` 删除请求或响应中的指定 Header。

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "example.com"
    action: DELETE
    rewrite-direction: REQUEST
    rewrite-header: "X-Debug"
```

`rewrite-header` 必填。`rewrite-direction` 省略时默认为 `REQUEST`。

`DELETE` 仅可用于 `header-rewrite`。
