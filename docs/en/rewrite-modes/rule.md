# RULE Rewrite Mode

`RULE` mode evaluates configured rule lists and executes the action attached to the first matching rule unless `continue: true` is set.

## Rule lists

```yaml
rewrite-mode: RULE

header-rewrite: []
body-rewrite: []
url-redirect: []
```

## Evaluation order

For requests, UA3F evaluates Body rules, Header rules, then URL redirect rules. For responses, it evaluates Body rules and then Header rules.

Within each list, rules are evaluated top to bottom. After a rule matches, evaluation stops unless that rule has `continue: true`.

## Direction

`rewrite-direction` controls whether an action applies to `REQUEST` or `RESPONSE`. If omitted for Header or Body actions, UA3F defaults to `REQUEST`.

## Notes

`RULE` is the most flexible mode and is the recommended choice for production configurations that need exceptions, domain scoping, or response rewriting.
