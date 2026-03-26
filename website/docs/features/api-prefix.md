# API Prefix

[Home](/) > [Features](index) > API Prefix

---

Use `--api-prefix` when your frontend calls `/api/*` but the real upstream uses bare paths (`/countries`, `/cities`).

## Usage

```sh
mockr --target https://api.example.com --api-prefix /api
```

mockr accepts requests at `/api/*`, strips `/api`, matches routes and forwards upstream using the stripped path.

## Route definitions

Route definitions always use the **stripped** path:

```toml
[[routes]]
method   = "GET"
match    = "/countries"      # not /api/countries
enabled  = true
fallback = "success"
```

Your frontend calls `http://localhost:4000/api/countries`, mockr strips `/api`, and matches against `/countries`.

---

**See also:** [CLI Reference](../cli-reference.md) | [Routes](../configuration/routes.md)
