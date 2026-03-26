# Configuration

[Home](/) > Configuration

---

mockr supports `.toml`, `.yaml`/`.yml`, and `.json` config files — auto-detected by file extension.

## Single file

Point `--config` at a file:

```sh
mockr --config mockr.toml
```

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

---

## Auto-detection

If `--config` is not set, mockr:

1. Looks for `mockr.toml` in the current directory
2. Falls back to loading all config files in `.` if none is found

---

## Config structure

A config file contains two top-level arrays:

| Key | Description |
|---|---|
| `[[routes]]` | HTTP route definitions |
| `[[grpc_routes]]` | gRPC route definitions |

Both arrays use the same case/condition/transition structure. They can coexist in the same file.

---

## Further reading

- [Routes](routes.md) — route fields, path matching patterns
- [Cases](cases.md) — case fields, response definitions, persistence
- [Config Formats](formats.md) — TOML, YAML, and JSON side-by-side
