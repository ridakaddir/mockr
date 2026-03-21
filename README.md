# mockr

A fast, zero-dependency-on-your-app CLI tool for frontend developers to **mock, stub, and proxy HTTP APIs** — written in Go.

Point your frontend at `mockr` instead of the real API. Mock only the endpoints you're actively building. Forward everything else to the real backend. Switch between response scenarios by editing a single config file.

---

## Features

- **Route-based mocking** — define routes with named response cases
- **Condition routing** — activate different cases based on request body fields, query params, or headers
- **Dynamic file resolution** — serve `stubs/user-{query.username}-orders.json` resolved at request time
- **Stateful mocks** — `POST`/`PUT`/`PATCH`/`DELETE` can persist changes back into stub files (append / replace / delete)
- **Reverse proxy fallthrough** — unmatched routes forward to a real upstream API
- **Hot reload** — edit your config file and changes apply instantly, no restart needed
- **CORS** — all responses include CORS headers automatically
- **Response templating** — `{{uuid}}`, `{{now}}`, `{{timestamp}}` in inline JSON values
- **Record mode** — proxy a real API and automatically save responses as stub files
- **Multi-format config** — TOML, YAML, or JSON config files

---

## Install

**From source (requires Go 1.22+):**

```sh
git clone https://github.com/ridakaddir/mockr.git
cd mockr
go build -o mockr .
```

**Or install directly:**

```sh
go install github.com/ridakaddir/mockr@latest
```

---

## Quick start

```sh
# 1. Scaffold a config file + example stub
mockr --init

# 2. Start the server (optionally point at a real API)
mockr --target https://api.example.com
```

Your frontend should use `http://localhost:4000` as its API base URL.

---

## CLI reference

```
mockr [flags]

Flags:
  -t, --target      <url>    Upstream API to proxy unmatched requests to
  -p, --port        <n>      Port to listen on (default: 4000)
  -c, --config      <file>   Config file path (default: ./mockr.toml)
  -a, --api-prefix  <path>   Strip this prefix from request paths before matching
                             routes and forwarding to upstream (e.g. /api)
      --init                 Scaffold a mockr.toml template in the current directory
      --record               Record mode: proxy all requests and save responses as stubs
  -h, --help
  -v, --version
```

---

## Config file

mockr supports `.toml`, `.yaml`/`.yml`, and `.json` config files. Auto-detected by extension.

### Minimal example

```toml
[[routes]]
method   = "GET"
match    = "/api/users"
enabled  = true
fallback = "success"

  [routes.cases.success]
  status = 200
  file   = "stubs/users.json"

  [routes.cases.empty]
  status = 200
  json   = '{"users": []}'

  [routes.cases.error]
  status = 500
  json   = '{"message": "Internal Server Error"}'
  delay  = 2
```

---

## Route fields

| Field | Type | Default | Description |
|---|---|---|---|
| `method` | string | — | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS` |
| `match` | string | — | Path pattern (see [Path matching](#path-matching)) |
| `enabled` | bool | `true` | Whether the route is active |
| `fallback` | string | — | Case to serve when no condition matches. Omit to proxy to upstream |
| `conditions` | array | — | Ordered list of conditions (see [Conditions](#conditions)) |
| `cases` | map | — | Named response definitions (see [Cases](#cases)) |

---

## Cases

Each case defines what to respond with.

| Field | Type | Default | Description |
|---|---|---|---|
| `status` | int | `200` | HTTP status code |
| `json` | string | — | Inline JSON response body (supports [template tokens](#template-tokens)) |
| `file` | string | — | Path to a stub file (supports [dynamic resolution](#dynamic-file-resolution)) |
| `delay` | int | `0` | Seconds to wait before responding |
| `persist` | bool | `false` | Write request body back into the stub file |
| `merge` | string | — | `append`, `replace`, or `delete` (requires `persist: true`) |
| `key` | string | — | Field used to locate a record for `replace`/`delete` |
| `array_key` | string | — | The array field inside the stub JSON to operate on |

Either `json` or `file` must be set (unless `persist: true` with `merge: delete`).

---

## Path matching

Three styles are supported:

```toml
# Exact
match = "/api/users"

# Wildcard — * matches any segment(s)
match = "/api/users/*"
match = "/api/*/orders"

# Regex — prefix with ~
match = "~^/api/users/\\d+$"
```

---

## Conditions

Conditions are evaluated top-to-bottom. The first passing condition activates its case. If none match, the `fallback` case is used.

```toml
[[routes]]
method   = "POST"
match    = "/api/orders"
enabled  = true
fallback = "default"

  [[routes.conditions]]
  source = "body"        # body | query | header
  field  = "user.type"  # dot-notation for nested body fields
  op     = "eq"         # eq | neq | contains | regex | exists | not_exists
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

### Condition fields

| Field | Type | Description |
|---|---|---|
| `source` | string | Where to read the value: `body`, `query`, `header` |
| `field` | string | Dot-notation path for body (`user.address.city`); key name for query/header |
| `op` | string | `eq`, `neq`, `contains`, `regex`, `exists`, `not_exists` |
| `value` | string | Comparison value (not needed for `exists` / `not_exists`) |
| `case` | string | Case key to activate when this condition passes |

---

## Dynamic file resolution

Use `{source.field}` placeholders in `file` paths. They are resolved from the request at runtime.

```toml
[routes.cases.user_orders]
status = 200
file   = "stubs/user-{query.username}-orders.json"
```

| Placeholder | Resolves from |
|---|---|
| `{query.username}` | `?username=john` → `stubs/user-john-orders.json` |
| `{body.user.id}` | JSON body `user.id` |
| `{header.X-User-Id}` | Request header `X-User-Id` |

If the resolved file does not exist, mockr falls through to the next condition or `fallback` (no 500 error).

---

## Stateful mocks (persist)

When `persist: true`, mutating requests update the stub file on disk. Subsequent reads reflect the change.

### `append` — add a record (POST)

```toml
[[routes]]
method   = "POST"
match    = "/api/users"
enabled  = true
fallback = "created"

  [routes.cases.created]
  status    = 201
  file      = "stubs/users.json"
  persist   = true
  merge     = "append"
  array_key = "users"
```

### `replace` — update a record (PUT / PATCH)

```toml
[routes.cases.updated]
status    = 200
file      = "stubs/users.json"
persist   = true
merge     = "replace"
key       = "id"
array_key = "users"
```

### `delete` — remove a record (DELETE)

```toml
[routes.cases.deleted]
status    = 204
file      = "stubs/users.json"
persist   = true
merge     = "delete"
key       = "id"
array_key = "users"
```

**Key resolution order for `replace`/`delete`:** path wildcard → request body → query param.

If the record is not found, mockr returns `404`:

```json
{ "error": "record not found", "key": "id", "value": "99" }
```

---

## Template tokens

Use these in `json` values:

| Token | Output |
|---|---|
| `{{uuid}}` | Random UUID v4 |
| `{{now}}` | RFC3339 timestamp (`2026-03-20T15:04:05Z`) |
| `{{timestamp}}` | Unix epoch in milliseconds |

```toml
[routes.cases.created]
status = 201
json   = '{"id": "{{uuid}}", "created_at": "{{now}}", "ts": {{timestamp}}}'
```

---

## API prefix

Use `--api-prefix` when your frontend calls `/api/*` but the real upstream API uses paths without that prefix (e.g. `/users`, `/posts`).

mockr will:
1. Accept requests at `http://localhost:4000/api/*`
2. Match routes using the stripped path (`/users`, not `/api/users`)
3. Forward to upstream using the stripped path

```sh
# Frontend calls /api/users → upstream receives /users
mockr --target https://api.example.com --api-prefix /api
```

Route definitions in your config always use the **stripped** path:

```toml
[[routes]]
method   = "GET"
match    = "/users"        # NOT /api/users
enabled  = true
fallback = "success"
```

---

## Record mode

Start mockr with `--record` to proxy all requests to the real API and automatically:

1. Save each response body to `stubs/<method>_<path>.json`
2. Append a disabled route stub to your config file

```sh
mockr --target https://api.example.com --record
```

Review and enable the generated stubs when you're ready to mock offline.

---

## Hot reload

mockr watches the config file for changes. Edit `active_scenario`, add a route, or tweak a case — changes take effect on the next request with no restart.

---

## YAML config example

```yaml
routes:
  - method: GET
    match: /api/users
    enabled: true
    fallback: success
    cases:
      success:
        status: 200
        file: stubs/users.json
      empty:
        status: 200
        json: '{"users": []}'

  - method: POST
    match: /api/users
    enabled: true
    fallback: created
    conditions:
      - source: body
        field: role
        op: eq
        value: admin
        case: admin_created
    cases:
      admin_created:
        status: 201
        json: '{"id": "{{uuid}}", "role": "admin"}'
      created:
        status: 201
        file: stubs/users.json
        persist: true
        merge: append
        array_key: users
```

---

## JSON config example

```json
{
  "routes": [
    {
      "method": "GET",
      "match": "/api/users",
      "enabled": true,
      "fallback": "success",
      "cases": {
        "success": { "status": 200, "file": "stubs/users.json" },
        "empty":   { "status": 200, "json": "{\"users\": []}" }
      }
    }
  ]
}
```

---

## Project structure

```
mockr/
├── main.go
├── cmd/
│   └── root.go              # CLI entry point (cobra)
└── internal/
    ├── config/
    │   ├── types.go          # Config, Route, Condition, Case structs
    │   └── loader.go         # Multi-format loader + fsnotify hot reload
    ├── logger/
    │   └── logger.go         # Pretty terminal request logger
    └── proxy/
        ├── server.go         # HTTP server + CORS middleware
        ├── handler.go        # Per-request dispatch
        ├── matcher.go        # Path matching (exact / wildcard / regex)
        ├── conditions.go     # Condition evaluation
        ├── dynamic_file.go   # {source.field} placeholder resolution
        ├── mock.go           # Serve mock responses + template rendering
        ├── persist.go        # Stateful stub file mutations
        ├── forward.go        # Reverse proxy to upstream
        └── record.go         # Record mode + --init scaffold
```

---

## Contributing

Contributions are welcome. Here's how to get started.

### Prerequisites

- Go 1.22 or newer
- Git

### Fork and clone

```sh
git clone https://github.com/ridakaddir/mockr.git
cd mockr
go mod download
```

### Build

```sh
go build -o mockr .
```

### Run locally

```sh
# Scaffold a test config
./mockr --init

# Start with a real upstream
./mockr --target https://jsonplaceholder.typicode.com --port 4000
```

### Run tests

```sh
go test ./...
```

### Code style

- Standard Go formatting — run `gofmt -w .` before committing
- `go vet ./...` must pass with no errors
- Keep packages focused: no business logic in `cmd/`, no HTTP concerns in `config/`

### Submitting a pull request

1. Fork the repo and create a branch from `main`:
   ```sh
   git checkout -b feat/your-feature-name
   ```
2. Make your changes with clear, focused commits
3. Add or update tests if your change affects behaviour
4. Run `go test ./...` and `go vet ./...` — both must pass
5. Open a pull request against `main` with a clear description of what and why

### Reporting a bug

Open an issue with:
- mockr version (`mockr --version`)
- Go version (`go version`)
- Your config file (redact any secrets)
- Steps to reproduce
- Expected vs actual behaviour

### Suggesting a feature

Open an issue describing the use case before writing code. This avoids effort on things that may not fit the project's scope.

### Areas open for contribution

- Additional condition operators (e.g. `lt`, `gt` for numeric comparisons)
- Response headers in case definitions
- Multi-route support in record mode (group captured responses by endpoint)
- Shell completions (`mockr completion bash|zsh|fish`)
- Homebrew / release automation

---

## License

MIT
