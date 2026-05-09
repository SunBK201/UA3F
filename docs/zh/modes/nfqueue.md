# NFQUEUE 模式

`NFQUEUE` 模式通过 Linux netfilter queue 在包层面处理流量，适用于网络层重写和 UA2F 类场景。

## 适用场景

当流量需要进入包处理队列，而不是通过代理 socket 处理时，使用 `NFQUEUE`。

## 配置

```yaml
server-mode: NFQUEUE
rewrite-mode: GLOBAL
```

## 行为

- netfilter 将选中的 TCP 包送入 NFQUEUE worker。
- UA3F 检测 HTTP payload 并在包内容中重写 User-Agent。
- 非 HTTP 或不支持的包会直接放行。

## 注意事项

NFQUEUE 仅适用于 Linux，兼容性和性能变量更多。除非确实需要包层处理，否则优先使用 HTTP、SOCKS5、TPROXY 或 REDIRECT。
