# GLOBAL 重写模式

`GLOBAL` 模式全局重写 User-Agent Header。

## 配置

```yaml
rewrite-mode: GLOBAL
user-agent: "FFF"
user-agent-regex: ""
user-agent-partial-replace: false
```

## 行为

- 所有匹配到的 HTTP 请求都会进入 User-Agent 重写逻辑。
- `user-agent` 是替换后的值。
- `user-agent-regex` 用于限制需要重写的 User-Agent，空值表示全部匹配。
- `user-agent-partial-replace` 为 `true` 时只替换正则匹配到的部分。

## 注意事项

简单部署可使用 `GLOBAL`。如果需要按域名、端口或 Header 分流，使用 `RULE`。
