# Rewrite Actions

Rewrite actions define what UA3F does after a rule matches.

## Header actions

These actions are available in `header-rewrite`.

| Action | Behavior |
| --- | --- |
| `DIRECT` | Stop rewriting and forward as-is |
| `DELETE` | Remove `rewrite-header` |
| `ADD` | Add `rewrite-header` with `rewrite-value` |
| `REPLACE` | Set `rewrite-header` to `rewrite-value` |
| `REPLACE-REGEX` | Replace the part of `rewrite-header` matched by `rewrite-regex` |
| `REJECT` | Reject the matched request or response |
| `DROP` | Drop the matched request or response |

Example:

```yaml
header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-direction: REQUEST
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

## Body actions

Body rewriting currently supports regex replacement.

| Action | Behavior |
| --- | --- |
| `DIRECT` | Stop rewriting and forward as-is |
| `REPLACE-REGEX` | Replace body content matched by `rewrite-regex` |
| `REJECT` | Reject the matched request or response |
| `DROP` | Drop the matched request or response |

Example:

```yaml
body-rewrite:
  - type: URL-REGEX
    match-value: "^http://ua-check.stagoh.com"
    action: REPLACE-REGEX
    rewrite-direction: RESPONSE
    rewrite-regex: "UA2F"
    rewrite-value: "UA3F"
```

## URL redirect actions

These actions are available in `url-redirect`.

| Action | Behavior |
| --- | --- |
| `DIRECT` | Stop redirect handling |
| `REDIRECT-302` | Return `302 Found` to the client |
| `REDIRECT-307` | Return `307 Temporary Redirect` to the client |
| `REDIRECT-HEADER` | Rewrite URL/Host handling without returning a redirect when possible |

Example:

```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/"
    action: REDIRECT-HEADER
    rewrite-regex: "^http://example.com/(.*)"
    rewrite-value: "http://mirror.example.com/$1"
```

`REDIRECT-HEADER` updates the request directly when the host is unchanged. If the host changes, UA3F sends the rewritten request itself and writes the response back to the client.
