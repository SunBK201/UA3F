# Rewrite Modes

UA3F supports three HTTP rewrite modes. The mode controls how HTTP requests and responses enter the rewrite pipeline.

```yaml
rewrite-mode: RULE
```

## GLOBAL

`GLOBAL` rewrites User-Agent headers globally.

```yaml
rewrite-mode: GLOBAL
user-agent: "FFF"
user-agent-regex: ""
user-agent-partial-replace: false
```

- `user-agent` is the replacement value.
- `user-agent-regex` limits which User-Agent values are rewritten. Empty means all values.
- `user-agent-partial-replace` replaces only the regex-matched part instead of the whole header.

Use `GLOBAL` for simple deployments where one User-Agent policy applies to all HTTP requests.

## DIRECT

`DIRECT` forwards traffic without HTTP rewriting.

```yaml
rewrite-mode: DIRECT
```

Use it to verify routing, firewall rules, and proxy behavior before enabling rewrite logic.

## RULE

`RULE` evaluates configured rule lists and executes the action attached to matching rules.

```yaml
rewrite-mode: RULE

header-rewrite: []
body-rewrite: []
url-redirect: []
```

For requests, UA3F evaluates Body rules, Header rules, then URL redirect rules. For responses, it evaluates Body rules and then Header rules.

Rules are evaluated top to bottom within each list. Evaluation stops after a match unless the rule has `continue: true`.

`rewrite-direction` controls whether an action applies to `REQUEST` or `RESPONSE`. Header and Body actions default to `REQUEST` when omitted.
