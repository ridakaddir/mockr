# Named Path Parameters

[Home](../README.md) > [Features](README.md) > Named Parameters

---

mockr supports `{name}` placeholders in route patterns to extract values from URL paths. Extracted values can be used for key resolution, conditions, and dynamic file paths.

## Syntax

Use curly braces to define named parameters that match exactly one path segment:

```toml
[[routes]]
method   = "GET"
match    = "/api/users/{userId}"
enabled  = true
fallback = "success"
```

A request to `GET /api/users/john123` extracts `userId = "john123"`.

---

## Multiple parameters

Extract several values from a single route:

```toml
[[routes]]
method   = "PUT"
match    = "/api/users/{userId}/posts/{postId}"
enabled  = true
fallback = "update_post"
```

`PUT /api/users/42/posts/99` extracts `userId = "42"` and `postId = "99"`.

---

## Mixed patterns

Named parameters can coexist with wildcard `*` patterns:

```toml
[[routes]]
method   = "GET"
match    = "/api/v1/*/environments/{envId}/endpoint/{endpointId}"
enabled  = true
fallback = "success"
```

---

## Dynamic file resolution

Use `{path.paramName}` in file paths to serve resource-specific stub files:

```toml
[[routes]]
method   = "GET"
match    = "/api/users/{userId}/profile"
enabled  = true
fallback = "user_profile"

  [routes.cases.user_profile]
  status = 200
  file   = "stubs/user-{path.userId}-profile.json"
```

**Request:** `GET /api/users/john123/profile`
**Resolves to:** `stubs/user-john123-profile.json`

Nested directory structures work too:

```toml
file = "stubs/env-{path.envId}/endpoint-{path.endpointId}.json"
```

---

## Persistence with named parameters

Named path parameters work as a fallback for key resolution in `merge = "append"` operations. When the request body doesn't contain the `key` field, mockr resolves it from the URL path:

```toml
[[routes]]
method   = "POST"
match    = "/api/v1/*/environments/*/endpoint/{endpointId}/publication"
enabled  = true
fallback = "published"

  [routes.cases.published]
  status   = 201
  file     = "stubs/publications/"
  persist  = true
  merge    = "append"
  key      = "endpointId"    # resolved from {endpointId} in the URL path
```

A `POST` to `/api/v1/org123/environments/env456/endpoint/6053b5e4-cdbf-40c0-a6c8-cb41f29fe6bf/publication` with body `{"subDomain": "gemma"}` creates `stubs/publications/6053b5e4-cdbf-40c0-a6c8-cb41f29fe6bf.json` — the `endpointId` value is extracted from the path and used as both the filename and injected into the saved record.

### Key resolution priority

When using `merge = "append"`, the filename is determined by the `key` field value, resolved in this order:

1. **Request body** — if the body contains the `key` field, that value is used
2. **Defaults** — if a `defaults` file provides the `key` field (e.g. via `{{uuid}}`), that value is used
3. **Named path parameters** — `{endpointId}`, `{postId}` from the URL path
4. **Path wildcards (single `*`)** — for patterns with a single `*`, the first wildcard match may be used as the key; patterns with multiple `*` are not guaranteed to produce a stable filename key
5. **Query parameters** — URL query parameters
6. **Auto-generated UUID** — if none of the above provide a value

> **Note:** Defaults are deep-merged into the request body before key resolution runs. If your `defaults` file sets the `key` field (e.g. `"userId": "{{uuid}}"`), that value will be used as the filename even when a named path parameter could provide one. To use path parameter keys with defaults, ensure the defaults file does **not** include the `key` field.

---

## Conditions with named parameters

Use `source = "path"` to match on extracted values:

```toml
[[routes]]
method   = "GET"
match    = "/api/users/{userId}/orders"
enabled  = true
fallback = "default"

  [[routes.conditions]]
  source = "path"
  field  = "userId"
  op     = "eq"
  value  = "vip-user"
  case   = "vip_orders"

  [routes.cases.vip_orders]
  status = 200
  file   = "stubs/vip-orders.json"

  [routes.cases.default]
  status = 200
  file   = "stubs/regular-orders.json"
```

---

## Security

Named parameter file resolution includes built-in protections:

- **Path traversal prevention** — `.` and `..` patterns are neutralised
- **Hidden file protection** — leading dots in filenames are replaced
- **Character sanitisation** — unsafe characters are removed from file paths

---

## Backward compatibility

Named path parameters are **100% backward compatible**:

- All existing route patterns continue to work unchanged
- Existing wildcard `*` behaviour is preserved
- No breaking changes to configuration format
- Routes without named parameters use existing fast paths

---

**See also:** [Dynamic File Resolution](dynamic-files.md) | [Directory-Based Stubs](directory-stubs.md) | [Conditions](conditions.md)
