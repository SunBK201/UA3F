# HTTPS MitM

MitM lets UA3F decrypt selected HTTPS connections so Header and Body rewrite rules can operate on HTTPS traffic.

Without MitM, UA3F can still forward HTTPS traffic, but the encrypted request and response contents are not visible to the rewrite pipeline.

## When to enable MitM

Enable MitM only when you need to rewrite HTTPS request or response data for specific hostnames.

Typical cases:

- Rewrite HTTPS request headers such as `User-Agent`.
- Rewrite HTTPS response headers.
- Apply Body rewrite rules to HTTPS responses.
- Test rewrite behavior against a known HTTPS endpoint.

## Configuration

```yaml
mitm:
  enabled: true
  hostname: "*.httpbin.com, example.com:8000"
  insecure-skip-verify: false
  ca-passphrase: ""
  ca-p12-base64: ""
```

| Field | Description |
| --- | --- |
| `enabled` | Enables HTTPS MitM |
| `hostname` | Comma-separated hostname allowlist; supports wildcard `*` and `:port` suffixes |
| `insecure-skip-verify` | Skips upstream server certificate verification |
| `ca-passphrase` | Passphrase for the CA PKCS#12 data |
| `ca-p12-base64` | Base64-encoded CA PKCS#12 data |

## Hostname scope

`hostname` controls which HTTPS destinations are intercepted. Keep it narrow.

Examples:

```yaml
mitm:
  enabled: true
  hostname: "example.com, *.httpbin.com, api.example.com:8443"
```

Wildcard matching is intended for domain families. A `:port` suffix limits interception to that port.

## Client trust

Clients must trust the CA used by UA3F. If the client does not trust the CA, HTTPS connections will fail during TLS verification.

For production use, generate and distribute a dedicated CA for UA3F instead of reusing a broad system or organization CA.

## Rewrite example

```yaml
server-mode: SOCKS5
rewrite-mode: RULE

mitm:
  enabled: true
  hostname: "*.httpbin.org"

header-rewrite:
  - type: DOMAIN-SUFFIX
    match-value: "httpbin.org"
    action: REPLACE
    rewrite-direction: REQUEST
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"
```

## Security notes

MitM expands the trust boundary because UA3F terminates client TLS and creates a separate upstream TLS connection. Enable it only for the hostnames that need rewriting, protect CA material carefully, and avoid `insecure-skip-verify` unless you are debugging a controlled environment.
