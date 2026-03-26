# Routes

[Home](/) > [Configuration](index) > Routes

---

## Route fields

| Field | Type | Default | Description |
|---|---|---|---|
| `method` | string | — | `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS` |
| `match` | string | — | Path pattern (see below) |
| `enabled` | bool | `true` | Whether the route is active |
| `fallback` | string | — | Case to serve when no condition matches; omit to proxy |
| `conditions` | array | — | Ordered list of conditions (see [Conditions](../features/conditions.md)) |
| `cases` | map | — | Named response definitions (see [Cases](cases.md)) |

---

## Path matching

mockr supports three path matching styles:

### Exact match

```toml
match = "/api/users"
```

Matches only `GET /api/users` exactly.

### Wildcard

`*` matches any segment(s):

```toml
match = "/api/users/*"          # /api/users/123, /api/users/abc
match = "/api/*/orders"         # /api/users/orders, /api/items/orders
```

### Regex

Prefix with `~` to use a regular expression:

```toml
match = "~^/api/users/\\d+$"   # /api/users/123 but not /api/users/abc
```

### Named parameters

Use `` `{name}` `` placeholders to extract values from URL segments:

```toml
match = "/api/users/{userId}"                    # extracts userId
match = "/api/users/{userId}/posts/{postId}"     # extracts both
```

Named parameters match exactly **one** path segment. They can be mixed with wildcards:

```toml
match = "/api/v1/*/environments/{envId}/endpoint/{endpointId}"
```

Extracted values are available for:
- [Dynamic file resolution](../features/dynamic-files.md) — `file = "stubs/user-{path.userId}.json"`
- [Persistence key resolution](../features/directory-stubs.md) — `key = "userId"` uses the path value
- [Conditions](../features/conditions.md) — `source = "path"`, `field = "userId"`

See [Named Parameters](../features/named-parameters.md) for full details.

---

## Route behaviour

### Fallback

When `fallback` is set, mockr serves that case when no [condition](../features/conditions.md) matches:

```toml
fallback = "success"    # serves the "success" case
```

When `fallback` is **omitted**, unmatched requests are forwarded to `--target` (proxy mode).

### Enabled

Set `enabled = false` to temporarily disable a route without deleting it. Disabled routes are skipped during matching.

### Match order

Routes are evaluated in the order they appear in the config file (or alphabetical order across files in directory config). The first matching route wins.

---

**See also:** [Cases](cases.md) | [Conditions](../features/conditions.md) | [Named Parameters](../features/named-parameters.md)
