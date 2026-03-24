# Response Transitions

[Home](../README.md) > [Features](README.md) > Response Transitions

---

Automatically advance through a sequence of response cases over time. This is useful for simulating state changes like order fulfillment, deployment pipelines, or approval workflows.

## Example

```toml
[[routes]]
method   = "GET"
match    = "/orders/*"
enabled  = true
fallback = "shipped"

  [[routes.transitions]]
  case  = "shipped"
  after = 30          # seconds from first request

  [[routes.transitions]]
  case  = "out_for_delivery"
  after = 90          # seconds from first request

  [[routes.transitions]]
  case  = "delivered"
  # no after — terminal state

  [routes.cases.shipped]
  status = 200
  json   = '{"number": "o123", "status": "shipped"}'

  [routes.cases.out_for_delivery]
  status = 200
  json   = '{"number": "o123", "status": "out_for_delivery"}'

  [routes.cases.delivered]
  status = 200
  json   = '{"number": "o123", "status": "delivered"}'
```

---

## Timeline

The timer starts on the **first request** to the route:

```
t = 0s   first request  → shipped
t = 30s  next request   → out_for_delivery
t = 90s  next request   → delivered  (terminal — stays here)
```

The `after` values are **cumulative** from the first request, not from the previous transition.

---

## `transitions` fields

| Field | Type | Required | Description |
|---|---|---|---|
| `case` | string | yes | Case key to serve during this stage |
| `after` | int | no | Seconds from first request before advancing. Omit on the last entry for a terminal state |

---

## Behaviour

- **Conditions take priority** — if the route also has [conditions](conditions.md), they are evaluated first. Transitions only activate when no condition matches
- **Shared timeline per route pattern** — all requests to `GET /orders/*` share one clock regardless of the specific ID (`/orders/123` and `/orders/456` advance together)
- **Hot reload resets** — editing the config file restarts the sequence from the beginning
- **No looping** — transitions are one-way; the last entry without `after` is the terminal state

---

## YAML equivalent

```yaml
routes:
  - method: GET
    match: /orders/*
    enabled: true
    fallback: shipped
    transitions:
      - case: shipped
        after: 30
      - case: out_for_delivery
        after: 90
      - case: delivered
    cases:
      shipped:
        status: 200
        json: '{"number": "o123", "status": "shipped"}'
      out_for_delivery:
        status: 200
        json: '{"number": "o123", "status": "out_for_delivery"}'
      delivered:
        status: 200
        json: '{"number": "o123", "status": "delivered"}'
```

---

## Example

See [`examples/transitions/`](../../examples/transitions/) for a complete working example.

---

**See also:** [Cases](../configuration/cases.md) | [Conditions](conditions.md) | [gRPC Transitions](../grpc/config.md#transitions)
