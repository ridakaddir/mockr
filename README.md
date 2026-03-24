<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-dark.svg">
    <img src="assets/logo.svg" alt="mockr" width="220">
  </picture>
</p>

<p align="center">
  A fast, zero-dependency-on-your-app CLI tool for developers to<br>
  <strong>mock, stub, and proxy HTTP and gRPC APIs</strong> — written in Go.
</p>

<p align="center">
  <a href="https://github.com/ridakaddir/mockr/actions/workflows/ci.yml"><img src="https://github.com/ridakaddir/mockr/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/ridakaddir/mockr/releases"><img src="https://github.com/ridakaddir/mockr/actions/workflows/release.yml/badge.svg" alt="Release"></a>
  <a href="https://goreportcard.com/report/github.com/ridakaddir/mockr"><img src="https://goreportcard.com/badge/github.com/ridakaddir/mockr" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
</p>

<p align="center">
  Point your frontend or service at <code>mockr</code> instead of the real API.<br>
  Mock only the endpoints you're actively building. Forward everything else to the real backend.<br>
  Switch between response scenarios by editing a config file — changes apply instantly with no restart.
</p>

---

## Features

- **Route-based mocking** — define routes with named response cases, switch between them by editing `fallback`
- **Named path parameters** — extract values from URLs with `{name}` syntax for key resolution and dynamic files
- **Condition routing** — activate different cases based on request body fields, query params, headers, or path parameters
- **Dynamic file resolution** — serve `stubs/user-{query.username}-orders.json` or `stubs/user-{path.userId}-profile.json` resolved at request time
- **Stateful mocks** — `POST`/`PUT`/`PATCH`/`DELETE` persist changes into stub files (append / replace / delete); supports both wrapped objects and bare arrays for API contract compatibility
- **Directory config** — point `--config` at a folder and mockr loads and merges all config files in it
- **Reverse proxy fallthrough** — unmatched routes forward to a real upstream API
- **Hot reload** — edit any config file and changes apply on the next request, no restart needed
- **Record mode** — proxy a real API, save responses as stub files, replay with matching latency
- **API prefix stripping** — `--api-prefix /api` strips the prefix before matching routes and forwarding upstream
- **CORS** — all responses include CORS headers automatically
- **Response templating** — `{{uuid}}`, `{{now}}`, `{{timestamp}}` in inline JSON values
- **Multi-format config** — TOML, YAML, or JSON — auto-detected by file extension
- **OpenAPI generation** — generate a complete mockr config from any OpenAPI 3 spec (file or URL)
- **Response transitions** — automatically advance through a sequence of cases over time (e.g. `shipped` → `out_for_delivery` → `delivered`, with `shipped` → `out_for_delivery` after 30s)
- **gRPC mock** — mock unary gRPC methods from `.proto` files; stub responses with protojson; no `protoc` or codegen required
- **gRPC proxy** — forward unmatched gRPC methods transparently to an upstream server (h2c)
- **gRPC conditions** — route gRPC calls to different cases based on request body fields, using the same condition operators as REST
- **gRPC persist** — stateful CRUD: `append` / `replace` / `delete` mutations on stub JSON files, same as REST persist
- **gRPC generate** — `mockr generate --proto` scaffolds `[[grpc_routes]]` config and stub JSON files from a `.proto` file
- **gRPC reflection** — built-in server reflection so `grpcurl` and `grpc-ui` work out of the box

---

## 📖 Additional Documentation

- **[`NAMED_PARAMETERS.md`](NAMED_PARAMETERS.md)** — Detailed implementation guide for named path parameters
- **[`examples/README.md`](examples/README.md)** — Complete examples with `grpcurl` commands
- **[`LICENSE`](LICENSE)** — MIT license terms

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

**HTTP — from an OpenAPI spec:**

```sh
mockr generate --spec openapi.yaml --out ./mocks
mockr --config ./mocks
```

**HTTP — scaffold a config manually:**

```sh
mockr --init
mockr --target https://api.example.com
```

Your frontend points at `http://localhost:4000`. Matched routes return mock responses. Everything else proxies to `--target`.

**gRPC — from a `.proto` file:**

```sh
# Generate config + stubs from a proto
mockr generate --proto service.proto --out ./mocks

# Start both HTTP (port 4000) and gRPC (port 50051) servers
mockr --config ./mocks --grpc-proto service.proto
```

Inspect with `grpcurl`:

```sh
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext -d '{"user_id":"1"}' localhost:50051 users.UserService/GetUser
```

---

## CLI reference

```
mockr [flags]

Flags:
  -t, --target        <url>    Upstream HTTP API to proxy unmatched requests to
  -p, --port          <n>      HTTP port to listen on (default: 4000)
  -c, --config        <path>   Config file or directory
                               (default: mockr.toml if present, else current directory)
  -a, --api-prefix    <path>   Strip this prefix before matching routes and forwarding
                               upstream (e.g. /api)
      --init                   Scaffold a mockr.toml template in the current directory
      --record                 Record mode: proxy all requests and save responses as stubs

  # gRPC flags (gRPC server starts only when --grpc-proto is provided)
      --grpc-proto    <file>   Path to a .proto file; repeat for multiple files
      --grpc-port     <n>      gRPC server port (default: 50051)
      --grpc-target   <addr>   Upstream gRPC server for proxy mode (e.g. localhost:9090)

  -h, --help
  -v, --version

Subcommands:
  generate    Generate a mockr config from an OpenAPI spec or .proto files
```

---

## Generate

Generate a complete mockr config directory from an OpenAPI 3 spec in one command. Works with local files and remote URLs.

```sh
# From a local file
mockr generate --spec openapi.yaml --out ./mocks

# From a remote URL
mockr generate --spec https://petstore3.swagger.io/api/v3/openapi.json --out ./mocks

# YAML format, single file instead of one per tag
mockr generate --spec openapi.yaml --format yaml --split=false
```

Then serve immediately — no editing required:

```sh
mockr --config ./mocks
```

### What is generated

For each path + operation in the spec:

- One config file per tag (e.g. `users.toml`, `orders.toml`) containing one route per operation
- One stub JSON file per response status code in `stubs/`
- OpenAPI path parameters (`{id}`) are converted to mockr wildcards (`*`)
- The first 2xx response is set as the route `fallback`

**Example output for the Petstore spec:**

```
mocks/
├── pet.toml        # 13 routes
├── store.toml      #  3 routes
├── user.toml       #  3 routes
└── stubs/
    ├── get_pet_petId_200.json
    ├── get_pet_findByStatus_200.json
    ├── post_pet_200.json
    └── ... (42 more)
```

**Generated config file (`pet.toml`):**

```toml
# Generated from openapi.yaml
# Tag: pet

# Returns pets based on status
[[routes]]
method   = "GET"
match    = "/pet/findByStatus"
enabled  = true
fallback = "success"

  [routes.cases.success]
  status = 200
  file   = "stubs/get_pet_findByStatus_200.json"

  [routes.cases.bad_request]
  status = 400
  file   = "stubs/get_pet_findByStatus_400.json"
```

### Stub quality

Stubs are populated in priority order:

| Priority | Source | Description |
|---|---|---|
| 1 | Spec `examples` | Used verbatim when present in the spec |
| 2 | Schema `example` / `default` / `enum` | First value used |
| 3 | Schema synthesis | Objects built from properties, arrays of one item, strings with format hints |

Format hints in synthesised stubs:

| Schema format | Synthesised value |
|---|---|
| `uuid` | `"{{uuid}}"` — rendered as a real UUID at request time |
| `date-time` | `"{{now}}"` — rendered as RFC3339 timestamp at request time |
| `date` | `"2026-01-01"` |
| `email` | `"user@example.com"` |
| `uri` | `"https://example.com"` |

### `generate` flags

```
mockr generate [flags]

OpenAPI flags:
  -s, --spec          <file|url>   OpenAPI spec file path or URL
  -o, --out           <dir>        Output directory for config and stubs  (default: mocks)
  -f, --format        <fmt>        Config format: toml, yaml, json        (default: toml)
      --split                      One file per tag; --split=false for a single file (default: true)

Proto flags:
      --proto         <file>       Path to a .proto file; repeat for multiple files
      --import-path   <dir>        Extra directory to search for proto imports; repeat for multiple
```

Either `--spec` or `--proto` is required. `--proto` takes precedence if both are provided.

### Using Task

```sh
task generate SPEC=openapi.yaml
task generate SPEC=https://petstore3.swagger.io/api/v3/openapi.json
task generate SPEC=openapi.yaml OUT=./petstore
```

---

## gRPC

mockr supports gRPC mock and proxy alongside the HTTP server — both run in the same process, activated by `--grpc-proto`.

### How it works

1. Provide one or more `.proto` files via `--grpc-proto` — no `protoc` or code generation required
2. Define `[[grpc_routes]]` in your config files alongside existing `[[routes]]`
3. mockr starts a gRPC server on `--grpc-port` (default 50051) and an HTTP server on `--port` (default 4000)
4. Incoming gRPC calls are matched by full method path, decoded from protobuf to JSON for condition evaluation, and the stub response is encoded back to protobuf wire format
5. Unmatched calls are forwarded to `--grpc-target` if set, or return `UNIMPLEMENTED`

### Quick start

```sh
# Generate config and stubs from a proto file
mockr generate --proto service.proto --out ./mocks

# Start (HTTP + gRPC)
mockr --config ./mocks --grpc-proto service.proto

# With upstream proxy for unmatched methods
mockr --config ./mocks \
      --grpc-proto service.proto \
      --grpc-target localhost:9090
```

### gRPC config — `[[grpc_routes]]`

gRPC routes live in the same config files as HTTP routes. All existing features work: conditions, transitions, fallback, delay, and template tokens.

```toml
[[grpc_routes]]
match    = "/users.UserService/GetUser"
enabled  = true
fallback = "ok"

  # Condition on a request body field (snake_case or camelCase both work)
  [[grpc_routes.conditions]]
  source = "body"
  field  = "user_id"
  op     = "eq"
  value  = "999"
  case   = "not_found"

  [grpc_routes.cases.ok]
  status = 0   # gRPC OK
  file   = "stubs/get_user.json"

  [grpc_routes.cases.not_found]
  status = 5   # gRPC NOT_FOUND
  json   = '{"message": "user not found"}'

  [grpc_routes.cases.error]
  status = 13  # gRPC INTERNAL
  json   = '{"message": "internal server error"}'
  delay  = 1
```

#### `match` format

The `match` field is the full gRPC method path: `"/package.Service/Method"`. All three matching styles work:

```toml
match = "/users.UserService/GetUser"   # exact
match = "/users.UserService/*"         # wildcard — all methods in the service
match = "~/users\\..*Service/.*"       # regex (prefix with ~)
```

#### gRPC status codes

`Case.status` is a [gRPC status code](https://grpc.github.io/grpc/core/md_doc_statuscodes.html) integer. Common values:

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
| `9` | FAILED_PRECONDITION | Operation rejected (e.g. already shipped) |
| `13` | INTERNAL | Server error |
| `14` | UNAVAILABLE | Service temporarily unavailable |
| `16` | UNAUTHENTICATED | Missing or invalid credentials |

#### Stub file format

Stub files and inline `json = "..."` values use [protojson](https://protobuf.dev/programming-guides/proto3/#json) — JSON with field names matching the proto field names (camelCase by default).

```json
{
  "userId": "usr_1a2b3c4d",
  "name": "Alice Smith",
  "email": "alice@example.com",
  "active": true
}
```

Template tokens (`{{uuid}}`, `{{now}}`, `{{timestamp}}`) work in gRPC stubs exactly as they do in REST:

```toml
[grpc_routes.cases.ok]
status = 0
json   = '{"userId": "{{uuid}}", "createdAt": "{{now}}"}'
```

#### Conditions on gRPC requests

Conditions evaluate fields from the decoded request message. Use `source = "body"` and dot-notation field paths. Both the proto field name (`payment_type`) and its camelCase equivalent (`paymentType`) are accepted automatically:

```toml
[[grpc_routes.conditions]]
source = "body"
field  = "payment_type"   # snake_case or camelCase both work
op     = "eq"
value  = "crypto"
case   = "pending_review"

[[grpc_routes.conditions]]
source = "body"
field  = "user.address.country"  # nested field, dot-notation
op     = "eq"
value  = "EU"
case   = "eu_response"
```

All condition operators work: `eq`, `neq`, `contains`, `regex`, `exists`, `not_exists`.

> Note: `source = "query"` and `source = "header"` are not applicable to gRPC and are ignored.

#### Proxy fallthrough

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

#### Stateful persist

gRPC routes support the same `persist` / `merge` / `key` / `array_key` fields as REST cases. The stub file is mutated on disk and subsequent reads reflect the change.

```toml
# Append incoming request body as a new record
[[grpc_routes]]
match    = "/items.ItemService/CreateItem"
enabled  = true
fallback = "created"

  [grpc_routes.cases.created]
  status    = 0
  file      = "stubs/items.json"
  persist   = true
  merge     = "append"
  array_key = "items"

# Replace the record matching key = "itemId" with the incoming body
[[grpc_routes]]
match    = "/items.ItemService/UpdateItem"
enabled  = true
fallback = "updated"

  [grpc_routes.cases.updated]
  status    = 0
  file      = "stubs/items.json"
  persist   = true
  merge     = "replace"
  key       = "itemId"
  array_key = "items"

# Delete the record matching key = "itemId"
[[grpc_routes]]
match    = "/items.ItemService/DeleteItem"
enabled  = true
fallback = "deleted"

  [grpc_routes.cases.deleted]
  status    = 0
  file      = "stubs/items.json"
  persist   = true
  merge     = "delete"
  key       = "itemId"
  array_key = "items"
```

**Key resolution:** for `replace` and `delete`, the key value is extracted from the incoming request body. Both snake_case (`item_id`) and camelCase (`itemId`) field names in the request are matched against `key` automatically.

**Error codes:**

| Situation | gRPC code |
|---|---|
| Record not found | `5` NOT_FOUND |
| Key field missing from request | `3` INVALID_ARGUMENT |
| Stub file unreadable / parse error | `13` INTERNAL |

**Response body:** all persist operations return an empty proto response (`{}`). The gRPC status code signals success or failure — inspect the stub file directly or follow up with a list/get call to confirm the mutation.

#### Transitions

Time-based transitions work identically to REST. The gRPC route key is the `match` pattern:

```toml
[[grpc_routes]]
match    = "/orders.OrderService/GetOrder"
enabled  = true
fallback = "processing"

  [[grpc_routes.transitions]]
  case  = "processing"
  after = 10

  [[grpc_routes.transitions]]
  case  = "shipped"
  after = 60

  [[grpc_routes.transitions]]
  case  = "delivered"

  [grpc_routes.cases.processing]
  status = 0
  json   = '{"status": "processing"}'

  [grpc_routes.cases.shipped]
  status = 0
  json   = '{"status": "shipped"}'

  [grpc_routes.cases.delivered]
  status = 0
  json   = '{"status": "delivered"}'
```

### `generate --proto`

Scaffold a complete `[[grpc_routes]]` config and synthetic stub files from a `.proto` file in one command:

```sh
mockr generate --proto service.proto --out ./mocks

# Multiple proto files
mockr generate --proto users.proto --proto orders.proto --out ./mocks

# With extra import paths for proto imports
mockr generate --proto service.proto --import-path ./vendor/protos --format yaml
```

**Generated output for a `UserService` with three methods:**

```
mocks/
├── mockr.toml            # [[grpc_routes]] for all methods
└── stubs/
    ├── UserService_GetUser.json
    ├── UserService_ListUsers.json
    └── UserService_CreateUser.json
```

Stubs are synthesised from the output message descriptor — field names, types and common naming patterns are used to produce sensible placeholder values:

| Field name pattern | Synthesised value |
|---|---|
| contains `id` | `"{{uuid}}"` |
| contains `email` | `"user@example.com"` |
| contains `url` / `uri` | `"https://example.com"` |
| contains `time` / `at` / `date` | `"{{now}}"` |
| contains `name` | `"Example Name"` |
| `bool` type | `true` |
| `int32` / `int64` etc. | `1` |
| `float` / `double` | `1.0` |

Then start the server immediately:

```sh
mockr --config ./mocks --grpc-proto service.proto
```

### gRPC reflection

mockr registers [gRPC server reflection](https://grpc.github.io/grpc/core/md_doc_server_reflection_tutorial.html) automatically. This means `grpcurl`, `grpc-ui`, and other tools can discover your services without a separate proto file:

```sh
# List all services
grpcurl -plaintext localhost:50051 list

# Describe a service
grpcurl -plaintext localhost:50051 describe users.UserService

# Describe a message
grpcurl -plaintext localhost:50051 describe users.GetUserRequest

# Call without specifying proto (reflection provides the schema)
grpcurl -plaintext -d '{"user_id":"1"}' localhost:50051 users.UserService/GetUser
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
| `status` | int | `200` | HTTP status code — or gRPC status code for `[[grpc_routes]]` (e.g. `0` = OK, `5` = NOT_FOUND) |
| `json` | string | — | Inline JSON body (supports [template tokens](#template-tokens)) |
| `file` | string | — | Stub file path (supports [dynamic resolution](#dynamic-file-resolution)) |
| `delay` | int | `0` | Seconds to wait before responding |
| `persist` | bool | `false` | Mutate the stub file on disk (append / replace / delete) |
| `merge` | string | — | `append`, `replace`, or `delete` (requires `persist: true`) |
| `key` | string | — | Field in the stored record to match on for `replace`/`delete` |
| `array_key` | string | — | Top-level array field inside the stub JSON to operate on |

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

## Named Path Parameters

mockr supports `{name}` placeholders in route patterns to extract and use named path parameters for key resolution, conditions, and dynamic file paths.

### Syntax

Use curly braces to define named parameters that match exactly one path segment:

```toml
[[routes]]
method   = "GET"
match    = "/api/users/{userId}"
enabled  = true
fallback = "success"
```

### Mixed Patterns

Named parameters can coexist with wildcard `*` patterns in the same route:

```toml
[[routes]]
method   = "GET"
match    = "/api/v1/*/environments/{envId}/endpoint/{endpointId}"
enabled  = true
fallback = "success"
```

### Dynamic File Resolution with Named Parameters

Use `{path.paramName}` placeholders in file paths to create user-specific or resource-specific stub files:

```toml
[[routes]]
method   = "GET"
match    = "/api/users/{userId}/profile"
enabled  = true
fallback = "user_profile"

  [routes.cases.user_profile]
  status = 200
  file   = "stubs/user-{path.userId}-profile.json"
```

**Request:** `GET /api/users/john123/profile`
**Resolves to:** `stubs/user-john123-profile.json`

### Persistence with Named Parameters

Named path parameters have the **highest priority** in key resolution for persistence operations:

```toml
[[routes]]
method   = "PUT"
match    = "/api/users/{userId}/posts/{postId}"
enabled  = true
fallback = "update_post"

  [routes.cases.update_post]
  status = 200
  file   = "stubs/posts.json"
  persist = true
  merge   = "replace"
  key     = "postId"  # Extracted from {postId} in URL path
  array_key = "posts"
```

### Key Resolution Priority

When using persistence operations, key values are resolved in this order:

1. **Named path parameters** — `{userId}`, `{postId}` from the URL path
2. **Path wildcards** — existing `*` behavior (fallback)
3. **Request body fields** — JSON field extraction
4. **Query parameters** — URL query parameters

### Conditions with Named Parameters

Named path parameters can be used in conditions via the `path` source:

```toml
[[routes]]
method   = "GET"
match    = "/api/users/{userId}/orders"
enabled  = true
fallback = "default"

  [[routes.conditions]]
  source = "path"
  field  = "userId"
  op     = "eq"
  value  = "vip-user"
  case   = "vip_orders"

  [routes.cases.vip_orders]
  status = 200
  file   = "stubs/vip-orders.json"

  [routes.cases.default]
  status = 200
  file   = "stubs/regular-orders.json"
```

### Security Features

Named parameter file resolution includes built-in security protections:

- **Path traversal prevention** — `.` and `..` patterns are neutralized
- **Hidden file protection** — leading dots in filenames are replaced
- **Character sanitization** — unsafe characters are removed from file paths

### Backward Compatibility

Named path parameters are **100% backward compatible**:

- All existing route patterns continue to work unchanged
- Existing wildcard `*` behavior is preserved
- No breaking changes to configuration format
- Routes without named parameters use existing fast paths

For detailed implementation information, see [`NAMED_PARAMETERS.md`](NAMED_PARAMETERS.md).

---

## Stateful mocks (persist)

When `persist: true`, mutating requests update the stub file on disk. Subsequent reads reflect the change.

**Mockr supports two persist modes** to match different API response formats. Choose the mode that matches your API's actual response structure for true contract compatibility:

### Wrapped Object Mode (Traditional)

Use when your API returns objects containing arrays like `{"users": [...]}`:

#### `append` — add a record (POST)

```toml
[routes.cases.created]
status    = 201
file      = "stubs/users.json"
persist   = true
merge     = "append"
array_key = "users"    # Required: specifies array field
```

#### `replace` — update a record (PUT / PATCH)

```toml
[routes.cases.updated]
status    = 200
file      = "stubs/users.json"
persist   = true
merge     = "replace"
key       = "id"
array_key = "users"    # Required: specifies array field
```

#### `delete` — remove a record (DELETE)

```toml
[routes.cases.deleted]
status    = 204
file      = "stubs/users.json"
persist   = true
merge     = "delete"
key       = "id"
array_key = "users"    # Required: specifies array field
```

**Stub file format:**
```json
{
  "users": [
    {"id": "1", "name": "Alice"}
  ]
}
```

### Bare Array Mode (New)

Use when your API returns bare arrays like `[{...}, {...}]` for true API contract compatibility:

#### `append` — add a record (POST)

```toml
[routes.cases.created]
status  = 201
file    = "stubs/users.json"
persist = true
merge   = "append"
# array_key omitted — operates on bare array
```

#### `replace` — update a record (PUT / PATCH)

```toml
[routes.cases.updated]
status  = 200
file    = "stubs/users.json"
persist = true
merge   = "replace"
key     = "id"
# array_key omitted — operates on bare array
```

#### `delete` — remove a record (DELETE)

```toml
[routes.cases.deleted]
status  = 204
file    = "stubs/users.json"
persist = true
merge   = "delete"
key     = "id"
# array_key omitted — operates on bare array
```

**Stub file format:**
```json
[
  {"id": "1", "name": "Alice"}
]
```

### Configuration Validation

Mockr automatically validates that your configuration matches your stub file format:

- **Wrapped mode** (with `array_key`): Stub file must be a JSON object containing the specified array field
- **Bare array mode** (without `array_key`): Stub file must be a JSON array at the root level

**Validation happens on every persist operation** to ensure data consistency and provide helpful error guidance.

**Common validation errors:**

```bash
# Mismatch: object file without array_key
Error: stub file "stubs/users.json" contains a JSON object but array_key is not specified. 
Either provide array_key to specify which field contains the array, or convert the file to a bare JSON array

# Mismatch: array file with array_key  
Error: stub file "stubs/users.json" is a bare JSON array but array_key="users" is specified. 
Either remove array_key or wrap the array in an object like {"users": [...]}

# Wrong field name
Error: array_key "items" not found in stub file. Available keys: ["data", "metadata"]

# Wrong field type
Error: field "users" is not a JSON array (found string). 
Ensure the field contains an array like: "users": [{...}, {...}]

# Invalid JSON content
Error: stub file contains null, expected JSON array or JSON object
Error: stub file contains number, expected JSON array or JSON object
```

### When to Use Each Mode

**🎯 Use Bare Array Mode (Recommended for new projects):**
- Your real API returns bare arrays: `[{}, {}]`
- You want **exact API contract compatibility**
- Building standard REST APIs following JSON:API or similar specs
- You prefer simpler stub file management
- Working with frontend frameworks that expect bare arrays

**📦 Use Wrapped Object Mode when:**
- Your real API returns `{"data": [...], "meta": {...}}`
- You need multiple fields in responses (pagination, metadata)
- Working with existing wrapped configurations
- Legacy API compatibility requirements

### Examples

- **Wrapped Object**: See `examples/persist/` 
- **Bare Array**: See `examples/bare-array-persist/`
- **Side-by-side comparison**: See `examples/persist-comparison/`

**Performance Note**: Persist operations read files only once per operation, with automatic validation and type detection for optimal performance.

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

## Response transitions

Automatically advance through a sequence of response cases over time. The timer starts on the **first request** to the route and resets whenever the config is hot-reloaded.

```toml
[[routes]]
method   = "GET"
match    = "/orders/*"
enabled  = true
fallback = "shipped"

  [[routes.transitions]]
  case  = "shipped"
  after = 30          # seconds from first request → advance to next

  [[routes.transitions]]
  case  = "out_for_delivery"
  after = 90          # seconds from first request → advance to next

  [[routes.transitions]]
  case  = "delivered"
  # no after — terminal state, stays here permanently

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

```
t = 0s   first request  → shipped
t = 30s  next request   → out_for_delivery
t = 90s  next request   → delivered  (terminal — stays here)
```

The `after` values are cumulative from the first request, not from the previous transition.

### `transitions` fields

| Field | Type | Required | Description |
|---|---|---|---|
| `case` | string | yes | Case key to serve during this stage |
| `after` | int | no | Seconds from first request before advancing. Omit on the last entry to make it terminal |

### Behaviour notes

- **Conditions take priority** — if the route also has `conditions`, they are evaluated first. Transitions only activate when no condition matches
- **Shared timeline per route pattern** — all requests to `GET /orders/*` share one clock regardless of the specific id (`/orders/123` and `/orders/456` advance together)
- **Hot reload resets** — editing the config file restarts the sequence from the beginning
- **No looping** — transitions are one-way; the last entry without `after` is the terminal state

### YAML equivalent

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
| `examples/persist` | Stateful CRUD backed by stub files (wrapped object mode) |
| `examples/bare-array-persist` | Stateful CRUD with bare array mode for API contract compatibility |
| `examples/persist-comparison` | Side-by-side comparison of wrapped vs bare array persist modes |
| `examples/dynamic-files` | `{source.field}` file path placeholders and named path parameters |
| `examples/full-crud` | All features combined — blog posts API |
| `examples/transitions` | Time-based response transitions and state progression |
| `examples/record-mode` | Proxy + auto-record workflow |
| `examples/openapi-generate` | Generate config from OpenAPI spec (Petstore example) |
| `examples/grpc-mock` | gRPC unary mock — named cases, error codes, template tokens |
| `examples/grpc-conditions` | gRPC condition routing on request body fields |
| `examples/grpc-proxy` | gRPC selective mock + transparent upstream proxy fallthrough |
| `examples/grpc-persist` | gRPC stateful CRUD — append / replace / delete backed by a stub file |

**Additional example files:**
- `named-params-example.toml` — Standalone named path parameters demonstration
- `dynamic-files-example.toml` — Dynamic file resolution patterns

**HTTP examples:** run with `mockr --config examples/<name>`

**gRPC examples:** run with `mockr --config examples/<name> --grpc-proto examples/<name>/<file>.proto`

See [`examples/README.md`](examples/README.md) for `grpcurl` commands for each example.

---

## Project structure

```
mockr/
├── main.go
├── cmd/
│   ├── root.go              # CLI entry point (cobra) — HTTP + gRPC flags
│   └── generate.go          # generate subcommand (OpenAPI + proto modes)
├── internal/
│   ├── config/
│   │   ├── types.go          # Config, Route, GRPCRoute, Condition, Case, Transition structs
│   │   └── loader.go         # File + directory loader, fsnotify hot reload
│   ├── generate/
│   │   ├── generator.go      # OpenAPI orchestrator: load spec → parse → write
│   │   ├── parser.go         # kin-openapi wrapper: load spec (file/URL), parse operations
│   │   ├── synth.go          # OpenAPI schema → synthetic example JSON
│   │   ├── writer.go         # Write TOML/YAML/JSON config + stub files (OpenAPI)
│   │   └── proto_generator.go# Proto → grpc_routes config + stub JSON files
│   ├── grpc/
│   │   ├── codec.go          # Raw-bytes passthrough codec (enables unknown-service handler)
│   │   ├── descriptor.go     # Runtime proto registry (jhump/protoreflect, no protoc)
│   │   ├── forward.go        # Transparent h2c proxy to upstream gRPC server
│   │   ├── handler.go        # gRPC unknown-service handler: match → condition → mock/proxy
│   │   ├── persist.go        # gRPC stateful mutations (append / replace / delete)
│   │   ├── server.go         # grpc.Server lifecycle + reflection
│   │   ├── template.go       # {{uuid}} / {{now}} / {{timestamp}} rendering for gRPC stubs
│   │   └── transitions.go    # Time-based transition state for gRPC routes
│   ├── logger/
│   │   └── logger.go         # Pretty terminal logger (HTTP + gRPC, via=stub/proxy)
│   ├── persist/
│   │   ├── persist.go        # Transport-agnostic stub file mutations (shared by HTTP + gRPC)
│   │   └── persist_test.go   # Comprehensive test coverage for persistence operations
│   └── proxy/
│       ├── server.go         # HTTP server + CORS middleware
│       ├── handler.go        # Per-request dispatch
│       ├── matcher.go        # Path matching (exact / wildcard / regex / named params)
│       ├── matcher_test.go   # Named parameters and pattern matching tests
│       ├── conditions.go     # Condition evaluation (body / query / header / path)
│       ├── dynamic_file.go   # {source.field} and {path.param} placeholder resolution
│       ├── dynamic_file_test.go # Dynamic file resolution and security tests
│       ├── mock.go           # Serve mock responses + template rendering
│       ├── persist.go        # HTTP persist wrapper (uses internal/persist)
│       ├── persist_test.go   # HTTP persistence integration tests
│       ├── transitions.go    # Time-based response transition state
│       ├── forward.go        # Reverse proxy to upstream
│       └── record.go         # Record mode + --init scaffold
├── examples/
│   ├── basic/                # HTTP — static stubs and named cases
│   ├── conditions/           # HTTP — condition routing
│   ├── persist/              # HTTP — stateful CRUD (wrapped object mode)
│   ├── bare-array-persist/   # HTTP — stateful CRUD (bare array mode)
│   ├── persist-comparison/   # HTTP — side-by-side persist mode comparison
│   ├── dynamic-files/        # HTTP — {source.field} and named parameter placeholders
│   ├── full-crud/            # HTTP — all features combined
│   ├── transitions/          # HTTP — time-based response transitions
│   ├── record-mode/          # HTTP — proxy + auto-record
│   ├── openapi-generate/     # HTTP — generate from OpenAPI spec
│   ├── grpc-mock/            # gRPC — basic unary mock
│   ├── grpc-conditions/      # gRPC — condition routing on body fields
│   ├── grpc-proxy/           # gRPC — selective mock + upstream proxy fallthrough
│   ├── grpc-persist/         # gRPC — stateful CRUD backed by a stub file
│   ├── named-params-example.toml     # Standalone named parameters demo
│   ├── dynamic-files-example.toml    # Standalone dynamic file demo
│   └── README.md             # Example usage with grpcurl commands
├── assets/
│   ├── logo.svg              # mockr logo (light theme)
│   └── logo-dark.svg         # mockr logo (dark theme)
├── scripts/
│   └── pre-push              # Git pre-push hook
├── .github/workflows/
│   ├── ci.yml                # Continuous integration pipeline
│   └── release.yml           # Automated release builds
├── NAMED_PARAMETERS.md       # Named path parameters implementation guide
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
task build                                          # compile binary
task test                                           # go test ./... -race
task lint                                           # golangci-lint
task fmt                                            # gofmt -w .
task vet                                            # go vet ./...
task check                                          # fmt + vet + lint + test in one shot
task run:basic                                      # run with examples/basic config
task run:full-crud                                  # run with examples/full-crud config
task run:grpc-mock                                  # run gRPC basic mock example
task run:grpc-conditions                            # run gRPC conditions example
task run:grpc-proxy                                 # run gRPC proxy example (GRPC_TARGET=addr)
task run:grpc-persist                               # run gRPC stateful CRUD example
task generate SPEC=openapi.yaml                     # generate from local OpenAPI spec
task generate SPEC=https://petstore3.swagger.io/api/v3/openapi.json  # generate from URL
task generate:proto PROTO=service.proto             # generate from a .proto file
task generate:proto:example                         # regenerate grpc-mock example (smoke test)
task snapshot                                       # local goreleaser build (no publish)
task clean                                          # remove binary and build artifacts
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

### Testing

mockr has comprehensive test coverage across all major components:

```sh
task test                    # Run all tests with race detection
go test ./... -race -v       # Verbose test output
go test ./internal/proxy/... # Test specific package
```

**Test coverage includes:**
- Unit tests for all internal packages (`*_test.go` files)
- Named path parameter extraction and matching
- Persistence operations (both wrapped and bare array modes)
- Dynamic file resolution and security features
- Condition evaluation and route matching
- gRPC functionality and protocol handling

### CI/CD Pipeline

The project uses GitHub Actions for continuous integration and release automation:

- **CI Workflow** (`.github/workflows/ci.yml`)
  - Runs on push to `main` and all pull requests
  - Tests on Linux and macOS
  - Includes `go vet`, `golangci-lint`, and race condition detection
  - Builds binary to ensure compilation success

- **Release Workflow** (`.github/workflows/release.yml`)  
  - Triggers on `v*` tag pushes
  - Uses GoReleaser for cross-platform binary builds
  - Automatically creates GitHub releases with binaries

### Development Workflow

```sh
# Pre-commit hook (optional but recommended)
ln -s ../../scripts/pre-push .git/hooks/pre-push

# Full development cycle
task fmt     # Format code
task vet     # Static analysis  
task lint    # golangci-lint
task test    # Run tests
task check   # All of the above in sequence
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
