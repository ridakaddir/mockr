# mockr

A fast, zero-dependency-on-your-app CLI tool for frontend developers to **mock, stub, and proxy HTTP APIs** — written in Go.

Point your frontend at `mockr` instead of the real API. Mock only the endpoints you're actively building. Forward everything else to the real backend. Switch between response scenarios by editing a config file — changes apply instantly with no restart.

[![CI](https://github.com/ridakaddir/mockr/actions/workflows/ci.yml/badge.svg)](https://github.com/ridakaddir/mockr/actions/workflows/ci.yml)
[![Release](https://github.com/ridakaddir/mockr/actions/workflows/release.yml/badge.svg)](https://github.com/ridakaddir/mockr/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/ridakaddir/mockr)](https://goreportcard.com/report/github.com/ridakaddir/mockr)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Features

- **Route-based mocking** — define routes with named response cases, switch between them by editing `fallback`
- **Condition routing** — activate different cases based on request body fields, query params, or headers
- **Dynamic file resolution** — serve `stubs/user-{query.username}-orders.json` resolved at request time
- **Stateful mocks** — `POST`/`PUT`/`PATCH`/`DELETE` persist changes into stub files (append / replace / delete)
- **Directory config** — point `--config` at a folder and mockr loads and merges all config files in it
- **Reverse proxy fallthrough** — unmatched routes forward to a real upstream API
- **Hot reload** — edit any config file and changes apply on the next request, no restart needed
- **Record mode** — proxy a real API, save responses as stub files, replay with matching latency
- **API prefix stripping** — `--api-prefix /api` strips the prefix before matching routes and forwarding upstream
- **CORS** — all responses include CORS headers automatically
- **Response templating** — `{{uuid}}`, `{{now}}`, `{{timestamp}}` in inline JSON values
- **Multi-format config** — TOML, YAML, or JSON — auto-detected by file extension

---

## Install

**Download a binary** from the [latest release](https://github.com/ridakaddir/mockr/releases):

```sh
# macOS Apple Silicon
curl -L https://github.com/ridakaddir/mockr/releases/latest/download/mockr_darwin_arm64.tar.gz | tar xz
sudo mv mockr /usr/local/bin/

# macOS Intel
curl -L https://github.com/ridakaddir/mockr/releases/latest/download/mockr_darwin_amd64.tar.gz | tar xz
sudo mv mockr /usr/local/bin/

# Linux x86-64
curl -L https://github.com/ridakaddir/mockr/releases/latest/download/mockr_linux_amd64.tar.gz | tar xz
sudo mv mockr /usr/local/bin/
```

**Install with Go:**

```sh
go install github.com/ridakaddir/mockr@latest
```

**Build from source:**

```sh
git clone https://github.com/ridakaddir/mockr.git
cd mockr
task build          # requires Task — see Development section
# or
go build -o mockr .
```

---

## Quick start

```sh
# 1. Scaffold a config file and example stubs
mockr --init

# 2. Start the server
mockr --target https://api.example.com
```

Your frontend points at `http://localhost:4000`. Matched routes return mock responses. Everything else proxies to `--target`.

---

## CLI reference

```
mockr [flags]

Flags:
  -t, --target      <url>    Upstream API to proxy unmatched requests to
  -p, --port        <n>      Port to listen on (default: 4000)
  -c, --config      <path>   Config file or directory
                             (default: mockr.toml if present, else current directory)
  -a, --api-prefix  <path>   Strip this prefix before matching routes and forwarding
                             upstream (e.g. /api)
      --init                 Scaffold a mockr.toml template in the current directory
      --record               Record mode: proxy all requests and save responses as stubs
  -h, --help
  -v, --version
```

---

## Config file

mockr supports `.toml`, `.yaml`/`.yml`, and `.json` — auto-detected by extension.

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

## Directory config

Point `--config` at a folder and mockr loads **all** config files in it, merging their routes in alphabetical order. Hot reload watches the whole directory — adding, editing, or removing any file takes effect immediately.

```sh
mockr --config ./mocks
```

Split routes by domain for clarity:

```
mocks/
├── auth.toml       # /auth/*
├── users.toml      # /users/*
├── products.toml   # /products/*
└── orders.toml     # /orders/*
```

Mix formats freely — TOML, YAML, and JSON can coexist in the same directory.

**Auto-detect:** if `--config` is not set, mockr looks for `mockr.toml` in the current directory and falls back to loading all config files in `.` if none is found.

---

## Route fields

| Field | Type | Default | Description |
|---|---|---|---|
| `method` | string | — | `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS` |
| `match` | string | — | Path pattern (see [Path matching](#path-matching)) |
| `enabled` | bool | `true` | Whether the route is active |
| `fallback` | string | — | Case to serve when no condition matches; omit to proxy |
| `conditions` | array | — | Ordered list of conditions (see [Conditions](#conditions)) |
| `cases` | map | — | Named response definitions (see [Cases](#cases)) |

---

## Cases

| Field | Type | Default | Description |
|---|---|---|---|
| `status` | int | `200` | HTTP status code |
| `json` | string | — | Inline JSON body (supports [template tokens](#template-tokens)) |
| `file` | string | — | Stub file path (supports [dynamic resolution](#dynamic-file-resolution)) |
| `delay` | int | `0` | Seconds to wait before responding |
| `persist` | bool | `false` | Write request body back into the stub file |
| `merge` | string | — | `append`, `replace`, or `delete` (requires `persist: true`) |
| `key` | string | — | Field to locate a record for `replace`/`delete` |
| `array_key` | string | — | Array field inside the stub JSON to operate on |

---

## Path matching

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

Conditions are evaluated top-to-bottom. The first passing condition activates its case. If none match, `fallback` is used.

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

### Condition operators

| Op | Description |
|---|---|
| `eq` | Exact match |
| `neq` | Not equal |
| `contains` | String contains |
| `regex` | Regular expression match |
| `exists` | Field is present |
| `not_exists` | Field is absent |

---

## Dynamic file resolution

Use `{source.field}` placeholders in `file` paths — resolved from the request at runtime.

```toml
[routes.cases.user_orders]
status = 200
file   = "stubs/user-{query.username}-orders.json"
```

| Placeholder | Resolves from |
|---|---|
| `{query.username}` | `?username=john` → `stubs/user-john-orders.json` |
| `{body.user.id}` | JSON body field `user.id` |
| `{header.X-User-Id}` | Request header `X-User-Id` |

If the resolved file does not exist, mockr falls through to the next condition or `fallback` — no 500 error.

---

## Stateful mocks (persist)

When `persist: true`, mutating requests update the stub file on disk. Subsequent reads reflect the change.

### `append` — add a record (POST)

```toml
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

Record not found returns `404`:

```json
{ "error": "record not found", "key": "id", "value": "99" }
```

---

## Template tokens

| Token | Output |
|---|---|
| `{{uuid}}` | Random UUID v4 |
| `{{now}}` | RFC3339 timestamp (`2026-03-21T00:00:00Z`) |
| `{{timestamp}}` | Unix epoch in milliseconds |

```toml
[routes.cases.created]
status = 201
json   = '{"id": "{{uuid}}", "created_at": "{{now}}", "ts": {{timestamp}}}'
```

---

## API prefix

Use `--api-prefix` when your frontend calls `/api/*` but the real upstream uses bare paths (`/users`, `/posts`).

```sh
mockr --target https://api.example.com --api-prefix /api
```

mockr accepts requests at `/api/*`, strips `/api`, matches routes and forwards upstream using the stripped path. Route definitions always use the **stripped** path:

```toml
[[routes]]
method   = "GET"
match    = "/users"      # not /api/users
enabled  = true
fallback = "success"
```

---

## Record mode

Record mode proxies all requests to the real API, saves each response as a stub file, and immediately starts serving the stub on subsequent requests. The recorded latency is saved as the `delay` field so stubs replay at realistic speed.

```sh
mockr --config ./mocks \
      --target https://api.example.com \
      --api-prefix /api \
      --record
```

Each new path is recorded once:

```
Request 1  →  via=proxy   (real network, e.g. 73ms)
              → stubs/get_users_1.json saved
              → route appended to mocks/recorded.toml
Request 2  →  via=stub    (local file, <1ms)
```

Serve offline after recording (no `--target`, no `--record`):

```sh
mockr --config ./mocks --api-prefix /api
```

---

## Hot reload

mockr watches the config file or directory for changes. Edit a case, change a `fallback`, or drop a new `.toml` file into the config directory — the next request picks up the changes with no restart.

```
via=stub   — response served from a local mock file
via=proxy  — response forwarded to the upstream API
```

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

## Examples

The `examples/` directory contains runnable examples for each feature. Each example is a directory — run with:

```sh
mockr --config examples/<name>
```

| Example | What it demonstrates |
|---|---|
| `examples/basic` | Static stubs, named cases, hot reload |
| `examples/conditions` | Body / query / header condition routing |
| `examples/persist` | Stateful CRUD backed by stub files |
| `examples/dynamic-files` | `{source.field}` file path placeholders |
| `examples/full-crud` | All features combined — blog posts API |
| `examples/record-mode` | Proxy + auto-record workflow |

See [`examples/README.md`](examples/README.md) for curl commands for each example.

---

## Project structure

```
mockr/
├── main.go
├── cmd/
│   └── root.go              # CLI entry point (cobra)
├── internal/
│   ├── config/
│   │   ├── types.go          # Config, Route, Condition, Case structs
│   │   └── loader.go         # File + directory loader, fsnotify hot reload
│   ├── logger/
│   │   └── logger.go         # Pretty terminal request logger (via=stub/proxy)
│   └── proxy/
│       ├── server.go         # HTTP server + CORS middleware
│       ├── handler.go        # Per-request dispatch
│       ├── matcher.go        # Path matching (exact / wildcard / regex)
│       ├── conditions.go     # Condition evaluation (body / query / header)
│       ├── dynamic_file.go   # {source.field} placeholder resolution
│       ├── mock.go           # Serve mock responses + template rendering
│       ├── persist.go        # Stateful stub file mutations
│       ├── forward.go        # Reverse proxy to upstream
│       └── record.go         # Record mode + --init scaffold
├── examples/                 # Runnable examples
├── Taskfile.yml              # Dev task runner
├── devbox.json               # Reproducible dev environment
└── .goreleaser.yml           # Cross-platform release builds
```

---

## Development

This project uses [devbox](https://www.jetify.com/devbox) for a reproducible local environment and [Task](https://taskfile.dev) as the task runner.

### Setup

```sh
# Enter the dev environment (installs Go 1.25.1, golangci-lint, goreleaser)
devbox shell

# List available tasks
task --list
```

### Common tasks

```sh
task build          # compile binary
task test           # go test ./... -race
task lint           # golangci-lint
task fmt            # gofmt -w .
task vet            # go vet ./...
task check          # fmt + vet + lint + test in one shot
task run:basic      # run with examples/basic config
task run:full-crud  # run with examples/full-crud config
task snapshot       # local goreleaser build (no publish)
task clean          # remove binary and build artifacts
```

### Without devbox

```sh
git clone https://github.com/ridakaddir/mockr.git
cd mockr
go mod download
go build -o mockr .
./mockr --init
./mockr --target https://jsonplaceholder.typicode.com --api-prefix /api
```

---

## Contributing

Contributions are welcome.

1. Fork the repo and create a branch from `main`:
   ```sh
   git checkout -b feat/your-feature-name
   ```
2. Make your changes with clear, focused commits
3. Run `task check` — fmt, vet, lint, and tests must all pass
4. Open a pull request against `main`

### Reporting a bug

Open an issue with:
- mockr version (`mockr --version`)
- Your config file (redact any secrets)
- Steps to reproduce + expected vs actual behaviour

### Areas open for contribution

- Additional condition operators (`lt`, `gt` for numeric comparisons)
- Response headers in case definitions
- Shell completions (`mockr completion bash|zsh|fish`)
- Homebrew formula

---

## CI / Release

| Workflow | Trigger | What it does |
|---|---|---|
| CI | push to `main`, pull requests | vet → lint → test (race) → build, on Linux + macOS |
| Release | `v*` tag push | goreleaser builds cross-platform binaries and creates a GitHub Release |

To cut a release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

---

## License

MIT
