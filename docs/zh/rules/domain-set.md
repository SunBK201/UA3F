# DOMAIN-SET 规则

`DOMAIN-SET` 从本地文件路径或远程 HTTP(S) URL 加载域名列表，并按后缀匹配 Host。

```yaml
header-rewrite:
  - type: DOMAIN-SET
    match-value: "/etc/ua3f/domain-set.txt"
    action: DIRECT
```

域名集文件按行解析，空行和以 `#` 开头的行会被忽略。

域名集会在规则初始化时异步加载。如果源加载失败，规则仍然存在，但没有已加载域名可供匹配。
