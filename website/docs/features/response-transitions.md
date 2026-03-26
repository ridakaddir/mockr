# Response Transitions

[Home](/) > [Features](index) > Response Transitions

---

Automatically advance resources through a sequence of states over time. Useful for simulating order fulfillment, deployment pipelines, provisioning workflows, or any state machine.

mockr supports two transition modes:

| Mode | Defined on | Timer starts | File mutated? | Use case |
|---|---|---|---|---|
| [Request-time](#request-time-transitions) | GET routes | First GET request | No | Read-only state progression |
| [Background](#background-transitions-on-post) | POST routes | Resource creation | Yes | CRUD with lifecycle states |

---

## Request-time transitions

Transitions on a **GET route** serve different cases based on elapsed time since the first request. The response changes automatically — no file mutation, no POST required.

### Example

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

### Timeline

The timer starts on the **first request** to the route:

```
t = 0s   first request  → shipped          (duration: 30s)
t = 30s  next request   → out_for_delivery  (duration: 60s)
t = 90s  next request   → delivered         (terminal — stays here)
```

### Behaviour

- **Conditions take priority** — if the route also has [conditions](conditions.md), they are evaluated first. Transitions only activate when no condition matches
- **Shared timeline per route pattern** — all requests to `GET /orders/*` share one clock regardless of the specific ID (`/orders/123` and `/orders/456` advance together)
- **Hot reload resets** — editing the config file restarts the sequence from the beginning
- **No looping** — transitions are one-way; the last entry without `duration` is the terminal state

### YAML equivalent

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

Transitions on a **POST route** schedule background file mutations after a resource is created. The file on disk is updated in the background, so all subsequent reads (GET by ID, GET list) reflect the new state automatically.

This is ideal for simulating deployment pipelines, provisioning workflows, or any resource that transitions through states after creation.

### Example: deployment lifecycle

```toml
# POST — creates deployment, starts background transition
[[routes]]
method   = "POST"
match    = "/deployments/{endpointId}"
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
match    = "/deployments/{endpointId}/{deploymentId}"
fallback = "success"

  [routes.cases.success]
  status = 200
  file   = "deployments/{path.endpointId}/{path.deploymentId}.json"

# GET list — directory aggregation, also sees the updated files
[[routes]]
method   = "GET"
match    = "/deployments/{endpointId}"
fallback = "success"

  [routes.cases.success]
  status = 200
  file   = "deployments/{path.endpointId}/"
```

With `defaults/deployment.json`:
```json
{"status": "Deploying", "region": "us-east-1", "createdAt": "{<!-- -->{now}<!-- -->"}
```

And `defaults/deployment-ready.json`:
```json
{"status": "Ready"}
```

### Timeline

The timer starts on **resource creation** (the POST request):

```
t = 0s   POST creates file  → {"status": "Deploying"}
t = 15s  background mutation → merges {"status": "Ready"} into the file
```

### Flow

1. **POST** creates the resource → responds 201 with `"status": "Deploying"`
2. **Background goroutine** sleeps for 15 seconds
3. **After 15s**, mockr merges `` `{"status": "Ready"}` `` into the file on disk
4. **Any GET** (by ID or list) now returns `"status": "Ready"`

### How it works

- The `fallback` case (`created`) handles the actual POST response
- Transition case names (`deploying`, `ready`) are for the scheduler, not for request-time case selection — they don't need to match the `fallback`
- Transition cases with `persist = true` and `defaults` are scheduled as background file mutations
- Each POST creates **independent timers** — different resources transition independently
- **Hot reload** cancels all pending background mutations
- **Server shutdown** waits for pending mutations to finish gracefully
- If the file is deleted before a transition fires, the mutation is skipped (logged as a warning)

### Multiple transition stages

Background transitions support multiple stages with cumulative durations:

```toml
[[routes.transitions]]
case     = "provisioning"
duration = 10               # first 10 seconds

[[routes.transitions]]
case     = "configuring"
duration = 20               # next 20 seconds (fires at t=10s)

[[routes.transitions]]
case     = "ready"          # fires at t=30s

[routes.cases.configuring]
persist  = true
merge    = "update"
defaults = "defaults/configuring.json"

[routes.cases.ready]
persist  = true
merge    = "update"
defaults = "defaults/ready.json"
```

---

## `transitions` fields

| Field | Type | Required | Description |
|---|---|---|---|
| `case` | string | yes | Case key for this stage (request-time: served as the response; background: used for scheduling) |
| `duration` | int | no | How long this state lasts in seconds. Omit on the last entry for a terminal state |

---

## Examples

See [`examples/transitions/`](../../examples/transitions/) for complete working examples:

- **`orders.toml`** — request-time transitions (GET returns different responses over time)
- **`deployments.toml`** — background transitions (POST creates resource, file mutates on disk after delay)

---

**See also:** [Cases](../configuration/cases.md) | [Conditions](conditions.md) | [gRPC Transitions](../grpc/config.md#transitions)
