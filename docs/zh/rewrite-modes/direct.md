# DIRECT 重写模式

`DIRECT` 模式只转发流量，不执行重写。

## 配置

```yaml
rewrite-mode: DIRECT
```

## 行为

- 选定服务模式需要解析或转发 HTTP 时，相关流程仍会运行。
- 不执行 Header、Body、URL 或 User-Agent 重写。

## 注意事项

`DIRECT` 适合在启用复杂规则前验证路由、防火墙和代理链路是否正常。
