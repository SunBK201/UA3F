# RULE 重写模式

`RULE` 模式按配置的规则列表匹配并执行动作。默认匹配到第一条规则后停止，除非规则设置了 `continue: true`。

## 规则列表

```yaml
rewrite-mode: RULE

header-rewrite: []
body-rewrite: []
url-redirect: []
```

## 执行顺序

请求方向依次执行 Body 规则、Header 规则、URL 重定向规则。响应方向依次执行 Body 规则和 Header 规则。

同一个列表内按从上到下执行。匹配到规则后默认停止，设置 `continue: true` 后继续匹配后续规则。

## 方向

`rewrite-direction` 控制动作应用于 `REQUEST` 或 `RESPONSE`。Header 与 Body 动作未配置时默认是 `REQUEST`。

## 注意事项

`RULE` 是最灵活的模式，适合需要例外规则、域名限定或响应重写的生产配置。
