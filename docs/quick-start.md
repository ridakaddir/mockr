# Quick Start

[Home](README.md) > Quick Start

---

## HTTP — from an OpenAPI spec

Generate a complete mock config from any OpenAPI 3 spec and start serving immediately:

```sh
mockr generate --spec openapi.yaml --out ./mocks
mockr --config ./mocks
```

Your mock server is now running at `http://localhost:4000` with routes for every operation in the spec.

---

## HTTP — manual scaffold

Create a starter config and point your frontend at mockr:

```sh
mockr --init
mockr --target https://api.example.com
```

Your frontend points at `http://localhost:4000`:
- **Matched routes** return mock responses
- **Everything else** proxies to `--target`

Edit `mockr.toml` to add routes, change responses, or switch between cases — changes apply instantly with no restart.

### Minimal config example

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

Change `fallback` from `"success"` to `"empty"` or `"error"` to switch responses. The next request picks up the change automatically.

---

## gRPC — from a `.proto` file

Generate config and stubs from a proto file, then start both HTTP and gRPC servers:

```sh
# Generate config + stubs
mockr generate --proto service.proto --out ./mocks

# Start servers (HTTP on :4000, gRPC on :50051)
mockr --config ./mocks --grpc-proto service.proto
```

Inspect with `grpcurl`:

```sh
# List all services (via server reflection)
grpcurl -plaintext localhost:50051 list

# Call a method
grpcurl -plaintext -d '{"user_id":"1"}' localhost:50051 users.UserService/GetUser
```

---

## What's next?

- [CLI Reference](cli-reference.md) — all flags and subcommands
- [Configuration](configuration/README.md) — config file format and options
- [Features](features/conditions.md) — conditions, persistence, transitions, and more
- [gRPC](grpc/README.md) — gRPC-specific documentation
- [Examples](examples.md) — runnable examples for every feature
