# REPLACE Action

`REPLACE` sets a Header to a new value.

```yaml
header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-direction: REQUEST
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

Use it for full Header replacement. For partial replacement, use `REPLACE-REGEX`.

`REPLACE` is only available in `header-rewrite`.
