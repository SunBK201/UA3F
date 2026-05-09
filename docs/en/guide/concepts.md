# Concepts

## Server modes

UA3F uses server modes to decide how traffic enters the rewrite pipeline.

| Mode | How it works | Typical use |
| --- | --- | --- |
| `HTTP` | HTTP proxy | Applications explicitly configured with an HTTP proxy |
| `SOCKS5` | SOCKS5 proxy | Clash, browsers, or proxy chains |
| `TPROXY` | netfilter TPROXY | Linux/OpenWrt transparent proxy while preserving original destinations |
| `REDIRECT` | netfilter REDIRECT | Linux/OpenWrt transparent proxy with simpler routing |
| `NFQUEUE` | netfilter NFQUEUE | Network-layer queue processing for UA2F-like scenarios |

## Rewrite modes

| Mode | Behavior |
| --- | --- |
| `GLOBAL` | Rewrite `User-Agent` for all requests |
| `DIRECT` | Forward traffic without rewriting |
| `RULE` | Match `header-rewrite`, `body-rewrite`, and `url-redirect` rules |

Most production configurations should use `RULE`, because it lets you treat specific domains, headers, ports, or URLs differently.

## Rule matching

Rules are evaluated from top to bottom. After a match, evaluation stops by default. Set `continue: true` to continue evaluating later rules.

Common match types:

| Type | Description |
| --- | --- |
| `DOMAIN` / `DOMAIN-SUFFIX` / `DOMAIN-KEYWORD` | Match the destination domain |
| `DOMAIN-SET` | Match a domain set |
| `IP-CIDR` / `SRC-IP` | Match destination or source IP |
| `DEST-PORT` | Match destination port |
| `HEADER-KEYWORD` / `HEADER-REGEX` | Match request headers |
| `URL-REGEX` | Match the full URL with a regular expression |
| `FINAL` | Fallback rule |

## Rule actions

| Action | Description |
| --- | --- |
| `DIRECT` | Pass through without rewriting |
| `REPLACE` | Replace a header value |
| `REPLACE-REGEX` | Replace only the regular-expression match |
| `ADD` | Add a header |
| `DELETE` | Delete a header |
| `REJECT` | Reject the request |
| `DROP` | Drop the request |
| `REDIRECT-302` / `REDIRECT-307` | Return an HTTP redirect |
| `REDIRECT-HEADER` | Rewrite request headers to redirect transparently |

## HTTPS MitM

Plain proxying can only inspect HTTP. To rewrite HTTPS headers or bodies, enable `mitm` for selected hostnames and make clients trust the CA used by UA3F.

Keep the hostname list narrow so the trusted CA is only used where rewriting is required.
