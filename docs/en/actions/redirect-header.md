# REDIRECT-HEADER Action

`REDIRECT-HEADER` rewrites request URL and Host handling without returning a redirect response when possible.

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/"
    action: REDIRECT-HEADER
    rewrite-regex: "^http://example.com/(.*)"
    rewrite-value: "http://mirror.example.com/$1"
```

If the rewritten URL keeps the same host, UA3F updates the request URL and continues. If the host changes, UA3F sends the rewritten request itself and writes the response back to the client.

On Linux, the internal request path sets a socket mark to avoid being captured again by UA3F's own firewall rules.
