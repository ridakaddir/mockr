# gRPC Quick Start

[Home](../README.md) > [gRPC](README.md) > Quick Start

---

## Generate and serve

The fastest way to start is to generate config and stubs from a `.proto` file:

```sh
# Generate config and stubs
mockr generate --proto service.proto --out ./mocks

# Start HTTP (port 4000) + gRPC (port 50051)
mockr --config ./mocks --grpc-proto service.proto
```

## With upstream proxy

Stub only the methods you care about — forward everything else to a real gRPC server:

```sh
mockr --config ./mocks \
      --grpc-proto service.proto \
      --grpc-target localhost:9090
```

## Inspect with grpcurl

mockr includes built-in [server reflection](https://grpc.github.io/grpc/core/md_doc_server_reflection_tutorial.html), so `grpcurl`, `grpc-ui`, and other tools work without specifying a proto file:

```sh
# List all services
grpcurl -plaintext localhost:50051 list

# Describe a service
grpcurl -plaintext localhost:50051 describe users.UserService

# Describe a message
grpcurl -plaintext localhost:50051 describe users.GetUserRequest

# Call a method
grpcurl -plaintext -d '{"user_id":"1"}' localhost:50051 users.UserService/GetUser
```

---

## What's next?

- [Configuration](config.md) — `[[grpc_routes]]` format and match patterns
- [Stubs & Conditions](stubs.md) — stub format, condition routing, proxy fallthrough
- [Persistence](persistence.md) — directory-based CRUD for gRPC
- [Generation](generation.md) — `mockr generate --proto` in detail
