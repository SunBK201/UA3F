# Match Rules

Match rules decide when a rewrite action should run. A rule is placed in one of these lists:

```yaml
header-rewrite: []
body-rewrite: []
url-redirect: []
```

Common fields:

| Field | Description |
| --- | --- |
| `type` | Match rule type |
| `match-value` | Value used by the matcher |
| `match-header` | Header name for Header matchers |
| `action` | Rewrite action to execute |
| `continue` | Continue evaluating later rules after this match |

## DOMAIN

`DOMAIN` matches the parsed request host exactly.

```yaml
header-rewrite:
  - type: DOMAIN
    match-value: "example.com"
    action: DIRECT
```

It does not match subdomains.

## DOMAIN-SUFFIX

`DOMAIN-SUFFIX` matches hosts ending with `match-value`.

```yaml
header-rewrite:
  - type: DOMAIN-SUFFIX
    match-value: "example.com"
    action: DIRECT
```

Use it for a domain family such as `example.com`, `api.example.com`, and `static.example.com`.

## DOMAIN-KEYWORD

`DOMAIN-KEYWORD` matches hosts containing `match-value`.

```yaml
header-rewrite:
  - type: DOMAIN-KEYWORD
    match-value: "example"
    action: DIRECT
```

Use exact or suffix matching when substring matching is too broad.

## DOMAIN-SET

`DOMAIN-SET` loads a domain list from a local file path or remote HTTP(S) URL and suffix-matches the parsed request host.

```yaml
header-rewrite:
  - type: DOMAIN-SET
    match-value: "/etc/ua3f/domain-set.txt"
    action: DIRECT
```

Domain set files are parsed line by line. Empty lines and lines beginning with `#` are ignored. The set is loaded asynchronously during rule initialization.

## IP-CIDR

`IP-CIDR` matches the remote destination IP against a CIDR range.

```yaml
header-rewrite:
  - type: IP-CIDR
    match-value: "203.0.113.0/24"
    action: DIRECT
```

If the value does not include a prefix length, UA3F treats it as a single IPv4 host by appending `/32`.

## SRC-IP

`SRC-IP` matches the client source IP against a CIDR range.

```yaml
header-rewrite:
  - type: SRC-IP
    match-value: "192.168.1.0/24"
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

Use it to apply different policies to different LAN clients.

## DEST-PORT

`DEST-PORT` matches the destination port as a string.

```yaml
header-rewrite:
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT
```

Quote the port in YAML to keep it as a string.

## HEADER-KEYWORD

`HEADER-KEYWORD` matches when a request Header contains a keyword. Header value matching is case-insensitive.

```yaml
header-rewrite:
  - type: HEADER-KEYWORD
    match-header: "User-Agent"
    match-value: "MicroMessenger"
    action: DIRECT
```

## HEADER-REGEX

`HEADER-REGEX` matches a request Header with a case-insensitive regular expression.

```yaml
header-rewrite:
  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Windows|Android|iPhone)"
    action: REPLACE-REGEX
    rewrite-header: "User-Agent"
    rewrite-regex: "(Windows|Android|iPhone)"
    rewrite-value: "UA3F"
```

Invalid expressions are logged and the rule will not match.

## URL-REGEX

`URL-REGEX` matches the full request URL with a regular expression.


```yaml
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/old"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old(.*)"
    rewrite-value: "https://example.com/new$1"
```

Use anchors such as `^` when the rule should only match a URL prefix.

## FINAL

`FINAL` always matches and is usually placed at the end of a rule list.

```yaml
header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

Place specific rules before broad rules. If `FINAL` appears before other rules, later rules are normally unreachable unless `continue: true` is set.
