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

## Background transitions on POST

Transitions can also be placed on **POST routes** to simulate resources that change state over time after creation. When a POST route with transitions creates a resource via `merge = "append"`, mockr spawns background goroutines that apply deferred file mutations at the configured intervals.

This is ideal for simulating deployment pipelines, provisioning workflows, or any resource that transitions through states after creation.

### Example: deployment lifecycle

```toml
# POST — creates deployment, starts background transition
[[routes]]
method   = "POST"
match    = "/api/v1/*/environments/*/endpoint/{endpointId}/deployment"
fallback = "created"

  [[routes.transitions]]
  case     = "deploying"
  duration = 15

  [[routes.transitions]]
  case     = "ready"

  [routes.cases.created]
  status   = 201
  file     = "deployments/{path.endpointId}/"
  persist  = true
  merge    = "append"
  key      = "deploymentId"
  defaults = "defaults/deployment.json"

  [routes.cases.ready]
  persist  = true
  merge    = "update"
  defaults = "defaults/deployment-ready.json"

# GET by ID — pure read, no transitions needed
[[routes]]
method   = "GET"
match    = "/api/v1/*/environments/*/endpoint/{endpointId}/deployment/{deploymentId}"
fallback = "success"

  [routes.cases.success]
  status = 200
  file   = "deployments/{path.endpointId}/{path.deploymentId}.json"

# GET list — directory aggregation, also sees the updated files
[[routes]]
method   = "GET"
match    = "/api/v1/*/environments/*/endpoint/{endpointId}/deployment"
fallback = "success"

  [routes.cases.success]
  status = 200
  file   = "deployments/{path.endpointId}/"
```

With `defaults/deployment.json`:
```json
{"status": "Deploying"}
```

And `defaults/deployment-ready.json`:
```json
{"status": "Ready"}
```

### Flow

1. **POST** creates the resource → responds 201 with `"status": "Deploying"`
2. **Background goroutine** sleeps for 15 seconds
3. **After 15s**, mockr merges `{"status": "Ready"}` into the file on disk
4. **Any GET** (by ID or list) now returns `"status": "Ready"`

### How it works

- The transition entries define the timeline — the `duration` on the first entry determines how long before the next case fires
- Transition cases (`ready`) that have `persist = true` and `defaults` are scheduled as background file mutations
- The `fallback` case (`created`) is used for the actual POST response — transition case names do not need to correspond to request-time cases
- Each POST creates independent timers — different deployments transition independently
- **Hot reload** cancels all pending background mutations
- **Server shutdown** waits for pending mutations to finish gracefully
- If the file is deleted before a transition fires, the mutation is skipped (logged as a warning)

---

## Example

See [`examples/transitions/`](../../examples/transitions/) for a complete working example.

---

**See also:** [Cases](../configuration/cases.md) | [Conditions](conditions.md) | [gRPC Transitions](../grpc/config.md#transitions)
