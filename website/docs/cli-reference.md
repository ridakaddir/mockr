# CLI Reference

[Home](/) > CLI Reference

---

## Usage

```
mockr [flags]
mockr generate [flags]
```

---

## Flags

### HTTP server

| Flag | Short | Default | Description |
|---|---|---|---|
| `--target <url>` | `-t` | — | Upstream HTTP API to proxy unmatched requests to |
| `--port <n>` | `-p` | `4000` | HTTP port to listen on |
| `--config <path>` | `-c` | `mockr.toml` or `.` | Config file or directory (auto-detected) |
| `--api-prefix <path>` | `-a` | — | Strip this prefix before matching routes and forwarding upstream (e.g. `/api`) |
| `--init` | | | Scaffold a `mockr.toml` template in the current directory |
| `--record` | | | Record mode: proxy all requests and save responses as stubs |

### gRPC server

gRPC server starts **only** when `--grpc-proto` is provided.

| Flag | Default | Description |
|---|---|---|
| `--grpc-proto <file>` | — | Path to a `.proto` file; repeat for multiple files |
| `--grpc-port <n>` | `50051` | gRPC server port |
| `--grpc-target <addr>` | — | Upstream gRPC server for proxy mode (e.g. `localhost:9090`) |

### General

| Flag | Short | Description |
|---|---|---|
| `--help` | `-h` | Show help |
| `--version` | `-v` | Show version |

---

## Subcommands

### `generate`

Generate a mockr config from an OpenAPI spec or `.proto` files.

```sh
mockr generate [flags]
```

#### OpenAPI flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--spec <file\|url>` | `-s` | — | OpenAPI spec file path or URL |
| `--out <dir>` | `-o` | `mocks` | Output directory for config and stubs |
| `--format <fmt>` | `-f` | `toml` | Config format: `toml`, `yaml`, `json` |
| `--split` | | `true` | One file per tag; `--split=false` for a single file |

#### Proto flags

| Flag | Default | Description |
|---|---|---|
| `--proto <file>` | — | Path to a `.proto` file; repeat for multiple files |
| `--import-path <dir>` | — | Extra directory to search for proto imports; repeat for multiple |

Either `--spec` or `--proto` is required. `--proto` takes precedence if both are provided.

---

## Common usage patterns

**Scaffold and start:**

```sh
mockr --init
mockr --target https://restcountries.com/v3.1
```

**Generate from OpenAPI and serve:**

```sh
mockr generate --spec openapi.yaml --out ./mocks
mockr --config ./mocks
```

**Selective mock with proxy fallthrough:**

```sh
mockr --config ./mocks --target https://restcountries.com/v3.1
```

**Record real API responses:**

```sh
mockr --config ./mocks --target https://restcountries.com/v3.1 --record
```

**Start HTTP + gRPC servers:**

```sh
mockr --config ./mocks --grpc-proto geo.proto
```

**gRPC with upstream proxy:**

```sh
mockr --config ./mocks \
      --grpc-proto geo.proto \
      --grpc-target localhost:9090
```

**Using Task runner:**

```sh
task generate SPEC=openapi.yaml
task generate SPEC=https://petstore3.swagger.io/api/v3/openapi.json
task generate SPEC=openapi.yaml OUT=./geo-mocks
```

---

**See also:** [Quick Start](quick-start.md) | [Configuration](configuration/)
