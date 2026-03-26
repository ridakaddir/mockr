# gRPC Configuration

[Home](../README.md) > [gRPC](README.md) > Configuration

---

gRPC routes live in the same config files as HTTP routes. All existing features work: conditions, transitions, fallback, delay, and template tokens.

## Basic config

```toml
[[grpc_routes]]
match    = "/geo.CountryService/GetCountry"
enabled  = true
fallback = "ok"

  [grpc_routes.cases.ok]
  status = 0   # gRPC OK
  file   = "stubs/get_country.json"

  [grpc_routes.cases.not_found]
  status = 5   # gRPC NOT_FOUND
  json   = '{"message": "country not found"}'

  [grpc_routes.cases.error]
  status = 13  # gRPC INTERNAL
  json   = '{"message": "internal server error"}'
  delay  = 1
```

---

## `match` format

The `match` field is the full gRPC method path: `"/package.Service/Method"`. All three matching styles work:

```toml
match = "/geo.CountryService/GetCountry"   # exact
match = "/geo.CountryService/*"            # wildcard — all methods in the service
match = "~/geo\\..*Service/.*"             # regex (prefix with ~)
```

---

## gRPC status codes

`Case.status` is a [gRPC status code](https://grpc.github.io/grpc/core/md_doc_statuscodes.html) integer:

| Code | Name | Meaning |
|---|---|---|
| `0` | OK | Success (default when status is omitted) |
| `1` | CANCELLED | Request cancelled |
| `2` | UNKNOWN | Unknown error |
| `3` | INVALID_ARGUMENT | Bad input |
| `4` | DEADLINE_EXCEEDED | Timeout |
| `5` | NOT_FOUND | Resource not found |
| `6` | ALREADY_EXISTS | Resource already exists |
| `7` | PERMISSION_DENIED | Authorisation failure |
| `9` | FAILED_PRECONDITION | Operation rejected |
| `13` | INTERNAL | Server error |
| `14` | UNAVAILABLE | Service temporarily unavailable |
| `16` | UNAUTHENTICATED | Missing or invalid credentials |

---

## Transitions

Time-based transitions work identically to HTTP. The gRPC route key is the `match` pattern:

```toml
[[grpc_routes]]
match    = "/geo.CountryService/GetVisaStatus"
enabled  = true
fallback = "submitted"

  [[grpc_routes.transitions]]
  case     = "submitted"
  duration = 10

  [[grpc_routes.transitions]]
  case     = "under_review"
  duration = 50

  [[grpc_routes.transitions]]
  case     = "approved"

  [grpc_routes.cases.submitted]
  status = 0
  json   = '{"status": "submitted"}'

  [grpc_routes.cases.under_review]
  status = 0
  json   = '{"status": "under_review"}'

  [grpc_routes.cases.approved]
  status = 0
  json   = '{"status": "approved"}'
```

See [Response Transitions](../features/response-transitions.md) for full details on timeline behaviour.

---

**See also:** [Stubs & Conditions](stubs.md) | [Persistence](persistence.md) | [Generation](generation.md)
