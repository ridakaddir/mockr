# Directory-Based Stub Storage

[Home](../README.md) > [Features](README.md) > Directory-Based Stubs

---

When `persist: true`, mutating requests operate on individual JSON files stored in directories. Each resource is a separate file, enabling a "single source of truth" convention.

## How it works

### Directory structure

```
stubs/
└── users/
    ├── 1.json     # {"userId": "1", "name": "Alice", ...}
    ├── 2.json     # {"userId": "2", "name": "Bob", ...}
    └── 3.json     # {"userId": "3", "name": "Charlie", ...}
```

### API operations

| HTTP Method | Operation | Behaviour |
|---|---|---|
| `GET` (list) | Aggregates all `.json` files in the directory into an array |
| `GET` (detail) | Reads a single file by ID |
| `POST` | Creates a new file (auto-generates UUID if ID is missing) |
| `PATCH`/`PUT` | Shallow-merges request body into an existing file |
| `DELETE` | Removes the file from disk |

---

## Configuration

### Create — `append`

```toml
[[routes]]
method   = "POST"
match    = "/api/users"
enabled  = true
fallback = "created"

  [routes.cases.created]
  status  = 201
  file    = "stubs/users/"       # Directory path (trailing /)
  persist = true
  merge   = "append"
  key     = "userId"             # Field used as filename
```

### Read — list (directory aggregation)

```toml
[[routes]]
method   = "GET"
match    = "/api/users"
enabled  = true
fallback = "list"

  [routes.cases.list]
  file = "stubs/users/"          # Returns array of all .json files
```

### Read — single file

```toml
[[routes]]
method   = "GET"
match    = "/api/users/{userId}"
enabled  = true
fallback = "user"

  [routes.cases.user]
  file = "stubs/users/{path.userId}.json"    # Dynamic filename from path
```

### Update — `update`

```toml
[[routes]]
method   = "PATCH"
match    = "/api/users/{userId}"
enabled  = true
fallback = "updated"

  [routes.cases.updated]
  file    = "stubs/users/{path.userId}.json"
  persist = true
  merge   = "update"             # Shallow merge into existing file
```

### Delete — `delete`

```toml
[[routes]]
method   = "DELETE"
match    = "/api/users/{userId}"
enabled  = true
fallback = "deleted"

  [routes.cases.deleted]
  status  = 204
  file    = "stubs/users/{path.userId}.json"
  persist = true
  merge   = "delete"             # Remove file from disk
```

---

## Auto-ID generation

When `merge = "append"` and the request body is missing the `key` field, mockr auto-generates a UUID:

```sh
# POST without userId
curl -X POST localhost:4000/api/users -d '{"name": "New User"}'

# Creates file: stubs/users/123e4567-e89b-12d3-a456-426614174000.json
# Response: {"userId": "123e4567-...", "name": "New User"}
```

---

## Defaults

When a `POST` creates a resource, the client typically sends only a subset of fields. The `defaults` field lets you enrich the response with server-generated values.

**Defaults file** (`stubs/defaults/user.json`):

```json
{
  "userId": "{{uuid}}",
  "role": "user",
  "active": true,
  "createdAt": "{{now}}"
}
```

**Config:**

```toml
[routes.cases.created]
status   = 201
file     = "stubs/users/"
persist  = true
merge    = "append"
key      = "userId"
defaults = "stubs/defaults/user.json"
```

**How it works:**

1. mockr reads the defaults file and resolves [template tokens](template-tokens.md) (`{{uuid}}` becomes a real UUID, `{{now}}` becomes a timestamp)
2. Deep-merges: defaults as the base, request body overlaid on top — **body always wins on conflicts**
3. The merged result is saved to disk and returned as the response

```sh
# POST with just name and email
curl -X POST localhost:4000/api/users -d '{"name": "Alice", "email": "alice@example.com"}'

# Response (and saved file) includes defaults:
# {
#   "userId": "a1b2c3d4-...",
#   "name": "Alice",
#   "email": "alice@example.com",
#   "role": "user",
#   "active": true,
#   "createdAt": "2026-03-24T10:30:00Z"
# }
```

**Works with `update` too:**

```toml
[routes.cases.updated]
file     = "stubs/users/{path.userId}.json"
persist  = true
merge    = "update"
defaults = "stubs/defaults/user-update.json"
```

**Dynamic defaults path:**

```toml
defaults = "stubs/defaults/{path.resourceType}.json"
```

**Error handling:** If the defaults file is missing or contains invalid JSON, mockr logs a warning and proceeds with the original request body.

---

## Nested subdirectories

Support sub-resources with nested directories:

```
stubs/
├── deployments/
│   ├── endpoint-123/
│   │   ├── deploy-1.json
│   │   └── deploy-2.json
│   └── endpoint-456/
│       └── deploy-3.json
```

```toml
# GET /endpoints/{endpointId}/deployments — list
[routes.cases.list_deployments]
file = "stubs/deployments/{path.endpointId}/"

# POST /endpoints/{endpointId}/deployments — create
[routes.cases.create_deployment]
file     = "stubs/deployments/{path.endpointId}/"
persist  = true
merge    = "append"
key      = "deploymentId"
defaults = "stubs/defaults/deployment.json"
```

---

## Benefits

- **Single source of truth** — each resource is one file
- **Version control friendly** — clean diffs per resource
- **No size limits** — unlimited scalability vs. single-file arrays
- **Intuitive structure** — file layout mirrors API structure
- **Atomic operations** — each resource operation is independent

---

## Example

See [`examples/directory-stubs/`](../../examples/directory-stubs/) for a complete working example with user listing, creation, retrieval, updates, and deletion.

---

**See also:** [Cases](../configuration/cases.md) | [Dynamic File Resolution](dynamic-files.md) | [Named Parameters](named-parameters.md) | [Template Tokens](template-tokens.md)
