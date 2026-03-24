# gRPC Stubs & Conditions

[Home](../README.md) > [gRPC](README.md) > Stubs & Conditions

---

## Stub file format

Stub files and inline `json = "..."` values use [protojson](https://protobuf.dev/programming-guides/proto3/#json) — JSON with field names matching the proto field names (camelCase by default):

```json
{
  "userId": "usr_1a2b3c4d",
  "name": "Alice Smith",
  "email": "alice@example.com",
  "active": true
}
```

[Template tokens](../features/template-tokens.md) work in gRPC stubs exactly as they do in HTTP:

```toml
[grpc_routes.cases.ok]
status = 0
json   = '{"userId": "{{uuid}}", "createdAt": "{{now}}"}'
```

---

## Conditions

Conditions evaluate fields from the decoded request message. Use `source = "body"` and dot-notation field paths. Both the proto field name (`payment_type`) and its camelCase equivalent (`paymentType`) are accepted automatically:

```toml
[[grpc_routes]]
match    = "/orders.OrderService/CreateOrder"
enabled  = true
fallback = "ok"

  [[grpc_routes.conditions]]
  source = "body"
  field  = "payment_type"     # snake_case or camelCase both work
  op     = "eq"
  value  = "crypto"
  case   = "pending_review"

  [[grpc_routes.conditions]]
  source = "body"
  field  = "user.address.country"  # nested field, dot-notation
  op     = "eq"
  value  = "EU"
  case   = "eu_response"

  [grpc_routes.cases.ok]
  status = 0
  file   = "stubs/order_created.json"

  [grpc_routes.cases.pending_review]
  status = 0
  json   = '{"status": "pending_review"}'

  [grpc_routes.cases.eu_response]
  status = 0
  file   = "stubs/order_eu.json"
```

All condition operators work: `eq`, `neq`, `contains`, `regex`, `exists`, `not_exists`.

> **Note:** `source = "query"` and `source = "header"` are not applicable to gRPC and are ignored.

See [Conditions](../features/conditions.md) for the full operator reference.

---

## Proxy fallthrough

When a gRPC route has no matching condition and no `fallback`, mockr forwards the call to `--grpc-target`. This lets you stub only the methods you care about:

```toml
# ListProducts is mocked for electronics only; all other categories are proxied
[[grpc_routes]]
match   = "/products.ProductService/ListProducts"
enabled = true
# No fallback — unmatched conditions go to --grpc-target

  [[grpc_routes.conditions]]
  source = "body"
  field  = "category"
  op     = "eq"
  value  = "electronics"
  case   = "electronics"

  [grpc_routes.cases.electronics]
  status = 0
  file   = "stubs/products_electronics.json"

# UpdateProduct is not defined at all — always proxied to --grpc-target
```

When no `--grpc-target` is set, unmatched methods return gRPC `UNIMPLEMENTED`.

---

**See also:** [Configuration](config.md) | [Persistence](persistence.md) | [Conditions](../features/conditions.md)
