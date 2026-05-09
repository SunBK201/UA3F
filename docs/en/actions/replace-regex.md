# REPLACE-REGEX Action

`REPLACE-REGEX` replaces only the part matched by `rewrite-regex`.

Header example:

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

Body example:

```yaml
body-rewrite:
  - type: URL-REGEX
    match-value: "^http://ua-check.stagoh.com"
    action: REPLACE-REGEX
    rewrite-direction: RESPONSE
    rewrite-regex: "UA2F"
    rewrite-value: "UA3F"
```

This action is available in `header-rewrite` and `body-rewrite`.
