# Conditions

[Home](/) > [Features](index) > Conditions

---

Conditions let you route requests to different response cases based on request data. They are evaluated top-to-bottom — the first passing condition activates its case. If none match, the route's `fallback` case is used.

## Example

```toml
[[routes]]
method   = "POST"
match    = "/api/orders"
enabled  = true
fallback = "default"

  [[routes.conditions]]
  source = "body"
  field  = "user.type"
  op     = "eq"
  value  = "vip"
  case   = "vip_response"

  [[routes.conditions]]
  source = "query"
  field  = "version"
  op     = "eq"
  value  = "v2"
  case   = "v2_response"

  [[routes.conditions]]
  source = "header"
  field  = "X-User-Role"
  op     = "eq"
  value  = "admin"
  case   = "admin_response"

  [routes.cases.vip_response]
  status = 200
  json   = '{"discount": 0.3}'

  [routes.cases.v2_response]
  status = 200
  json   = '{"version": "v2"}'

  [routes.cases.admin_response]
  status = 200
  json   = '{"scope": "admin"}'

  [routes.cases.default]
  status = 200
  json   = '{"discount": 0.0}'
```

---

## Condition fields

| Field | Type | Description |
|---|---|---|
| `source` | string | Where to look: `body`, `query`, `header`, or `path` |
| `field` | string | Field name (dot-notation for nested body fields) |
| `op` | string | Comparison operator (see below) |
| `value` | string | Value to compare against (not used with `exists`/`not_exists`) |
| `case` | string | Case key to activate when the condition passes |

---

## Sources

### `body`

Match on JSON request body fields. Supports dot-notation for nested fields:

```toml
[[routes.conditions]]
source = "body"
field  = "user.address.country"
op     = "eq"
value  = "US"
case   = "us_response"
```

### `query`

Match on URL query parameters:

```toml
[[routes.conditions]]
source = "query"
field  = "page"
op     = "eq"
value  = "1"
case   = "first_page"
```

### `header`

Match on request headers:

```toml
[[routes.conditions]]
source = "header"
field  = "Authorization"
op     = "exists"
case   = "authenticated"
```

### `path`

Match on [named path parameters](named-parameters.md):

```toml
# Route: match = "/api/users/{userId}/orders"
[[routes.conditions]]
source = "path"
field  = "userId"
op     = "eq"
value  = "vip-user"
case   = "vip_orders"
```

> **Note:** `source = "query"` and `source = "header"` are not applicable to gRPC routes and are ignored.

---

## Operators

| Operator | Description | Example |
|---|---|---|
| `eq` | Exact match | `value = "admin"` |
| `neq` | Not equal | `value = "guest"` |
| `contains` | String contains | `value = "@example.com"` |
| `regex` | Regular expression match | `value = "^usr_[a-z]+$"` |
| `exists` | Field is present | (no `value` needed) |
| `not_exists` | Field is absent | (no `value` needed) |

---

## Evaluation order

1. Conditions are evaluated **top-to-bottom** in array order
2. The **first** passing condition activates its case
3. If **no condition** matches, the `fallback` case is served
4. If **no fallback** is set, the request is forwarded to `--target` (proxy mode)

---

## Multiple conditions example

```toml
[[routes]]
method   = "POST"
match    = "/api/payments"
enabled  = true
fallback = "default"

  # Check body field first
  [[routes.conditions]]
  source = "body"
  field  = "amount"
  op     = "regex"
  value  = "^[0-9]{5,}$"
  case   = "large_payment"

  # Then check query param
  [[routes.conditions]]
  source = "query"
  field  = "currency"
  op     = "eq"
  value  = "EUR"
  case   = "euro_payment"

  # Then check header
  [[routes.conditions]]
  source = "header"
  field  = "X-Priority"
  op     = "exists"
  case   = "priority_payment"
```

---

**See also:** [Routes](../configuration/routes.md) | [Cases](../configuration/cases.md) | [Named Parameters](named-parameters.md)
