# Cases

[Home](/) > [Configuration](index) > Cases

---

Cases are named response definitions attached to a route. Each route can have multiple cases — one is selected at request time based on [conditions](../features/conditions.md), [transitions](../features/response-transitions.md), or the route's `fallback`.

## Case fields

| Field | Type | Default | Description |
|---|---|---|---|
| `status` | int | `200` | HTTP status code (or gRPC status code for `[[grpc_routes]]`) |
| `json` | string | — | Inline JSON body (supports [template tokens](../features/template-tokens.md)) |
| `file` | string | — | Stub file path (supports [dynamic resolution](../features/dynamic-files.md)) |
| `delay` | int | `0` | Seconds to wait before responding |
| `persist` | bool | `false` | Mutate the stub file/directory on disk |
| `merge` | string | — | `update`, `append`, or `delete` (requires `persist: true`) |
| `key` | string | — | Field name for filename when using `append` with directories (resolved from body, path params, or query — see [directory stubs](../features/directory-stubs.md#key-resolution-for-filenames)) |
| `defaults` | string | — | JSON file with default values for `append`/`update` operations |

---

## Response body

Provide the response body with **either** `json` (inline) or `file` (from disk) — not both.

### Inline JSON

```toml
[routes.cases.success]
status = 200
json   = '{"users": [{"id": 1, "name": "Alice"}]}'
```

Inline JSON supports [template tokens](../features/template-tokens.md):

```toml
[routes.cases.created]
status = 201
json   = '{"id": "{<!-- -->{uuid}<!-- -->", "created_at": "{<!-- -->{now}<!-- -->"}'
```

### File-based response

```toml
[routes.cases.success]
status = 200
file   = "stubs/users.json"
```

File paths support [dynamic resolution](../features/dynamic-files.md):

```toml
[routes.cases.user_profile]
status = 200
file   = "stubs/user-{path.userId}-profile.json"
```

### Directory aggregation

When `file` points to a directory (trailing `/`), mockr aggregates all `.json` files into an array:

```toml
[routes.cases.list]
file = "stubs/users/"    # returns array of all files in stubs/users/
```

---

## Delay

Simulate network latency with `delay` (in seconds):

```toml
[routes.cases.slow_response]
status = 200
json   = '{"data": "slow"}'
delay  = 3
```

---

## Persistence

When `persist: true`, the request body is written to disk. Used with `merge` to define the operation:

| Merge | Operation | Description |
|---|---|---|
| `append` | Create | Writes a new file to a directory |
| `update` | Update | Shallow-merges request body into an existing file |
| `delete` | Delete | Removes a file from disk |

```toml
# Create
[routes.cases.created]
status  = 201
file    = "stubs/users/"
persist = true
merge   = "append"
key     = "userId"

# Update
[routes.cases.updated]
file    = "stubs/users/{path.userId}.json"
persist = true
merge   = "update"

# Delete
[routes.cases.deleted]
status  = 204
file    = "stubs/users/{path.userId}.json"
persist = true
merge   = "delete"
```

See [Directory-Based Stubs](../features/directory-stubs.md) for the full persistence guide.

### Defaults

Enrich `append`/`update` operations with server-generated fields:

```toml
[routes.cases.created]
status   = 201
file     = "stubs/users/"
persist  = true
merge    = "append"
key      = "userId"
defaults = "stubs/defaults/user.json"
```

The defaults file is deep-merged under the request body — body values always win on conflicts. Template tokens in the defaults file are resolved before merging.

---

**See also:** [Routes](routes.md) | [Directory-Based Stubs](../features/directory-stubs.md) | [Template Tokens](../features/template-tokens.md)
