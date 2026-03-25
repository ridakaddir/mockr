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
  case     = "shipped"
  duration = 30          # serve for 30 seconds

  [[routes.transitions]]
  case     = "out_for_delivery"
  duration = 60          # serve for 60 seconds

  [[routes.transitions]]
  case     = "delivered"
  # no duration — terminal state

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
t = 0s   first request  → shipped          (duration: 30s)
t = 30s  next request   → out_for_delivery  (duration: 60s)
t = 90s  next request   → delivered         (terminal — stays here)
```

Each `duration` value specifies **how long that state lasts**, not an absolute timestamp. Durations are accumulated internally to determine transition points.

---

## `transitions` fields

| Field | Type | Required | Description |
|---|---|---|---|
| `case` | string | yes | Case key to serve during this stage |
| `duration` | int | no | How long this state lasts in seconds. Omit on the last entry for a terminal state |

---

## Behaviour

- **Conditions take priority** — if the route also has [conditions](conditions.md), they are evaluated first. Transitions only activate when no condition matches
- **Shared timeline per route pattern** — all requests to `GET /orders/*` share one clock regardless of the specific ID (`/orders/123` and `/orders/456` advance together)
- **Hot reload resets** — editing the config file restarts the sequence from the beginning
- **No looping** — transitions are one-way; the last entry without `duration` is the terminal state

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
        duration: 30
      - case: out_for_delivery
        duration: 60
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
