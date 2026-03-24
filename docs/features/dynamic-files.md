# Dynamic File Resolution

[Home](../README.md) > [Features](README.md) > Dynamic File Resolution

---

Use `{source.field}` placeholders in `file` paths to resolve stub filenames from request data at runtime.

## Syntax

```toml
[routes.cases.user_orders]
status = 200
file   = "stubs/user-{query.username}-orders.json"
```

A request to `GET /api/orders?username=john` resolves to `stubs/user-john-orders.json`.

---

## Placeholder sources

| Placeholder | Resolves from | Example |
|---|---|---|
| `{query.fieldName}` | URL query parameter | `?username=john` |
| `{body.fieldName}` | JSON request body field | `{"user": {"id": "42"}}` |
| `{header.HeaderName}` | Request header | `X-User-Id: abc` |
| `{path.paramName}` | Named path parameter | `/users/{userId}` |

### Examples

```toml
# From query parameter
file = "stubs/user-{query.username}-orders.json"
# ?username=alice → stubs/user-alice-orders.json

# From body field (dot-notation for nested)
file = "stubs/org-{body.user.orgId}.json"
# {"user": {"orgId": "acme"}} → stubs/org-acme.json

# From header
file = "stubs/tenant-{header.X-Tenant-Id}.json"
# X-Tenant-Id: t1 → stubs/tenant-t1.json

# From named path parameter
file = "stubs/user-{path.userId}-profile.json"
# /api/users/john123/profile → stubs/user-john123-profile.json
```

---

## Fallback behaviour

If the resolved file does not exist, mockr falls through to the next condition or `fallback` — no 500 error. This lets you serve specific stubs for known values and a generic response for everything else.

---

## Security

Dynamic file resolution includes built-in protections against malicious input:

- **Path traversal prevention** — `.` and `..` patterns are neutralised
- **Hidden file protection** — leading dots in filenames are replaced
- **Character sanitisation** — unsafe characters are removed from file paths

---

**See also:** [Named Parameters](named-parameters.md) | [Cases](../configuration/cases.md) | [Directory-Based Stubs](directory-stubs.md)
