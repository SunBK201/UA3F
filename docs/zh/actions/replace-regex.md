# REPLACE-REGEX 动作

`REPLACE-REGEX` 只替换 `rewrite-regex` 匹配到的部分。

Header 示例：

```yaml
header-rewrite:
  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Windows|Android)"
    action: REPLACE-REGEX
    rewrite-header: "User-Agent"
    rewrite-regex: "(Windows|Android)"
    rewrite-value: "UA3F"
```

Body 示例：

```yaml
body-rewrite:
  - type: URL-REGEX
    match-value: "^http://ua-check.stagoh.com"
    action: REPLACE-REGEX
    rewrite-direction: RESPONSE
    rewrite-regex: "UA2F"
    rewrite-value: "UA3F"
```

该动作可用于 `header-rewrite` 和 `body-rewrite`。
