# URL-REGEX Rule

`URL-REGEX` matches the full request URL.

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/old"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old(.*)"
    rewrite-value: "https://example.com/new$1"
```

It is most commonly used in `url-redirect` rules, but can also be used with Body rules for URL-scoped response rewriting.

Use anchors such as `^` when a rule should only match a URL prefix.
