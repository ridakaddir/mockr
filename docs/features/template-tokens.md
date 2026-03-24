# Template Tokens

[Home](../README.md) > [Features](README.md) > Template Tokens

---

mockr supports template tokens in inline JSON values and defaults files. Tokens are replaced with generated values at request time.

## Available tokens

| Token | Output | Example |
|---|---|---|
| `{{uuid}}` | Random UUID v4 | `"a1b2c3d4-e5f6-7890-abcd-ef1234567890"` |
| `{{now}}` | RFC 3339 timestamp | `"2026-03-24T10:30:00Z"` |
| `{{timestamp}}` | Unix epoch in milliseconds | `1711273800000` |

---

## Usage in inline JSON

```toml
[routes.cases.created]
status = 201
json   = '{"id": "{{uuid}}", "created_at": "{{now}}", "ts": {{timestamp}}}'
```

**Response:**

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "created_at": "2026-03-24T10:30:00Z",
  "ts": 1711273800000
}
```

> **Note:** `{{uuid}}` and `{{now}}` are string tokens (include quotes). `{{timestamp}}` is a numeric token (no quotes needed).

---

## Usage in defaults files

Template tokens also work in [defaults files](directory-stubs.md#defaults) for persistence operations:

**`stubs/defaults/user.json`:**

```json
{
  "userId": "{{uuid}}",
  "role": "user",
  "active": true,
  "createdAt": "{{now}}"
}
```

Tokens are resolved before the defaults are merged with the request body.

---

## Usage in gRPC stubs

Template tokens work identically in gRPC stub responses:

```toml
[grpc_routes.cases.ok]
status = 0
json   = '{"userId": "{{uuid}}", "createdAt": "{{now}}"}'
```

---

**See also:** [Cases](../configuration/cases.md) | [Directory-Based Stubs](directory-stubs.md)
