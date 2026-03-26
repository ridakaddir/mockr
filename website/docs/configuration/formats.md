# Config Formats

[Home](/) > [Configuration](index) > Formats

---

mockr auto-detects the config format from the file extension. All three formats support the same feature set.

## TOML

```toml
[[routes]]
method   = "GET"
match    = "/api/users"
enabled  = true
fallback = "success"

  [[routes.conditions]]
  source = "query"
  field  = "role"
  op     = "eq"
  value  = "admin"
  case   = "admin_users"

  [routes.cases.success]
  status = 200
  file   = "stubs/users.json"

  [routes.cases.admin_users]
  status = 200
  file   = "stubs/admin-users.json"

[[routes]]
method   = "POST"
match    = "/api/users"
enabled  = true
fallback = "created"

  [routes.cases.created]
  status = 201
  json   = '{"id": "{<!-- -->{uuid}<!-- -->", "role": "user"}'
```

---

## YAML

```yaml
routes:
  - method: GET
    match: /api/users
    enabled: true
    fallback: success
    conditions:
      - source: query
        field: role
        op: eq
        value: admin
        case: admin_users
    cases:
      success:
        status: 200
        file: stubs/users.json
      admin_users:
        status: 200
        file: stubs/admin-users.json

  - method: POST
    match: /api/users
    enabled: true
    fallback: created
    cases:
      created:
        status: 201
        json: '{"id": "{<!-- -->{uuid}<!-- -->", "role": "user"}'
```

---

## JSON

```json
{
  "routes": [
    {
      "method": "GET",
      "match": "/api/users",
      "enabled": true,
      "fallback": "success",
      "conditions": [
        {
          "source": "query",
          "field": "role",
          "op": "eq",
          "value": "admin",
          "case": "admin_users"
        }
      ],
      "cases": {
        "success": { "status": 200, "file": "stubs/users.json" },
        "admin_users": { "status": 200, "file": "stubs/admin-users.json" }
      }
    },
    {
      "method": "POST",
      "match": "/api/users",
      "enabled": true,
      "fallback": "created",
      "cases": {
        "created": {
          "status": 201,
          "json": "{\"id\": \"{<!-- -->{uuid}<!-- -->\", \"role\": \"user\"}"
        }
      }
    }
  ]
}
```

---

## Mixing formats

When using [directory config](index#directory-config), you can mix formats freely:

```
mocks/
├── auth.toml
├── users.yaml
└── products.json
```

All files are loaded and their routes merged in alphabetical order.

---

**See also:** [Configuration Overview](index) | [Routes](routes.md) | [Cases](cases.md)
