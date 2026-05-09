# 重写模式

UA3F 支持三种 HTTP 重写模式。重写模式决定 HTTP 请求和响应如何进入重写流程。

```yaml
rewrite-mode: RULE
```

## GLOBAL

`GLOBAL` 全局重写 User-Agent Header。

```yaml
rewrite-mode: GLOBAL
user-agent: "FFF"
user-agent-regex: ""
user-agent-partial-replace: false
```

- `user-agent` 是替换后的值。
- `user-agent-regex` 用于限制需要重写的 User-Agent，空值表示全部匹配。
- `user-agent-partial-replace` 为 `true` 时只替换正则匹配到的部分。

当所有 HTTP 请求都适用同一套 User-Agent 策略时，使用 `GLOBAL`。

## DIRECT

`DIRECT` 只转发流量，不执行 HTTP 重写。

```yaml
rewrite-mode: DIRECT
```

它适合在启用重写逻辑前验证路由、防火墙规则和代理链路。

## RULE

`RULE` 按配置的规则列表匹配，并执行匹配规则附带的动作。

```yaml
rewrite-mode: RULE

header-rewrite: []
body-rewrite: []
url-redirect: []
```

请求方向依次执行 Body 规则、Header 规则、URL 重定向规则。响应方向依次执行 Body 规则和 Header 规则。

同一个列表内按从上到下执行。匹配到规则后默认停止，设置 `continue: true` 后继续匹配后续规则。

`rewrite-direction` 控制动作应用于 `REQUEST` 或 `RESPONSE`。Header 与 Body 动作未配置时默认是 `REQUEST`。
