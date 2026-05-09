# API Reference

UA3F provides a RESTful API for runtime status, configuration, rules, logs, and restart control.

Enable it with:

```yaml
api-server: "127.0.0.1:9000"
api-server-secret: "change-me"
```

## Authentication

When `api-server-secret` is not empty, send the secret with either an Authorization header:

```http
Authorization: Bearer change-me
```

or with the `secret` query parameter:

```text
/version?secret=change-me
```

Unauthorized requests return `401` with:

```json
{"error":"unauthorized"}
```

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/version` | Get the UA3F version |
| `GET` | `/config` | Get the active runtime configuration |
| `GET` | `/rules` | Get all header, body, and redirect rules |
| `GET` | `/rules/header` | Get header rewrite rules |
| `GET` | `/rules/body` | Get body rewrite rules |
| `GET` | `/rules/redirect` | Get URL redirect rules |
| `GET` | `/logs` | Stream or fetch runtime logs |
| `GET` | `/restart` | Reload configuration and restart runtime components |

## Examples

Get the version:

```sh
curl http://127.0.0.1:9000/version
```

Response:

```json
{
  "version": "0.7.0"
}
```

Get all rules:

```sh
curl -H "Authorization: Bearer change-me" http://127.0.0.1:9000/rules
```

Response shape:

```json
{
  "header": [],
  "body": [],
  "redirect": []
}
```

## Rule object

```json
{
  "enabled": true,
  "type": "HEADER-KEYWORD",
  "match_header": "User-Agent",
  "match_value": "MicroMessenger",
  "action": "REPLACE",
  "rewrite_header": "User-Agent",
  "rewrite_value": "FFF",
  "rewrite_direction": "REQUEST",
  "rewrite_regex": "",
  "continue": false
}
```

| Field | Type | Description |
| --- | --- | --- |
| `enabled` | boolean | Whether the rule is enabled |
| `type` | string | Match type |
| `match_header` | string | Header name for header-based matches |
| `match_value` | string | Match value |
| `action` | string | Action to execute |
| `rewrite_header` | string | Header to rewrite |
| `rewrite_value` | string | Replacement value |
| `rewrite_direction` | string | `REQUEST` or `RESPONSE` |
| `rewrite_regex` | string | Regex used by `REPLACE-REGEX` |
| `continue` | boolean | Continue evaluating later rules after this match |
