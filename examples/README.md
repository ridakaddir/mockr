# Examples

Each example is a **directory** of config files. Run any of them with:

```sh
mockr --config examples/<name>
```

mockr loads all `.toml`, `.yaml`, `.yml`, and `.json` files in the directory and merges their routes. Files are loaded in alphabetical order — routes defined earlier take priority.

---

## Structure

```
examples/
├── basic/                    # Static stubs, named cases, hot-reload
│   ├── products.toml         # GET /api/products
│   ├── product-detail.toml   # GET /api/products/:id
│   └── stubs/
│
├── conditions/               # Condition-based routing
│   ├── checkout.toml         # POST /api/cart/checkout — body/query/header conditions
│   ├── orders.toml           # POST /api/orders — nested dot-notation body fields
│   └── stubs/
│
├── persist/                  # Stateful CRUD
│   ├── todos.toml            # Full CRUD backed by stubs/todos.json
│   └── stubs/
│
├── dynamic-files/            # {source.field} placeholders in file paths
│   ├── mockr.toml            # GET /api/orders + POST /api/profile
│   └── stubs/
│
├── full-crud/                # All features combined (blog posts API)
│   ├── posts-read.toml       # GET /api/posts — filtering, simulation
│   ├── posts-write.toml      # POST/PUT/PATCH/DELETE — persist + conditions
│   └── stubs/
│
└── record-mode/              # Proxy + auto-record workflow
    └── mockr.toml
```

---

## basic

Serves product data from stub files. Shows hot-reload — change `fallback` in any file and the next request reflects it immediately, no restart.

```sh
mockr --config examples/basic
```

```sh
http :4000/api/products           # list — via stubs/products.json
http :4000/api/products/1         # detail — via stubs/product-detail.json
```

Switch to empty list: edit `products.toml` and change `fallback = "empty"`.  
Switch to error: change `fallback = "error"`.

---

## conditions

The same endpoint returns different responses depending on who is calling.

```sh
mockr --config examples/conditions
```

**Checkout** (`checkout.toml`) — body, query, and header conditions:

```sh
http POST :4000/api/cart/checkout user_type=vip       # 200 discounted
http POST :4000/api/cart/checkout user_type=banned    # 403 forbidden
http POST ':4000/api/cart/checkout?version=v2'        # 200 v2 shape
http POST :4000/api/cart/checkout X-User-Role:admin   # 200 full details
http POST :4000/api/cart/checkout user_type=standard  # 200 fallback
```

**Orders** (`orders.toml`) — nested body fields with dot-notation:

```sh
http POST :4000/api/orders payment:='{"method":"crypto"}' items:='[{"id":"1"}]'  # 202
http POST :4000/api/orders payment:='{"method":"card"}' items:='[{"id":"1"}]'    # 201
http POST :4000/api/orders payment:='{"method":"card"}'                           # 422 missing items
```

---

## persist

A stateful todo list. All writes mutate `stubs/todos.json` on disk — the file is the source of truth between requests.

```sh
mockr --config examples/persist
```

```sh
http :4000/api/todos                                       # list
http POST :4000/api/todos id=4 title="Review PR" done:=false  # create
http PUT :4000/api/todos/2 id=2 title="Write tests" done:=true # replace
http PATCH :4000/api/todos/1 done:=true                    # partial update
http DELETE :4000/api/todos/3                              # delete
http DELETE :4000/api/todos/99                             # → 404
```

Reset: `git checkout examples/persist/stubs/todos.json`

---

## dynamic-files

File paths contain `{source.field}` placeholders resolved from the request at runtime.

```sh
mockr --config examples/dynamic-files
```

```sh
# Resolves to stubs/orders-user-alice-orders.json
http ':4000/api/orders?username=alice'

# Resolves to stubs/orders-user-bob-orders.json
http ':4000/api/orders?username=bob'

# File not found → falls through to fallback (empty orders)
http ':4000/api/orders?username=unknown'

# Resolves to stubs/profile-id-1-profile.json (from body.id)
http POST :4000/api/profile id=1

# Add a new user without changing the config:
cp examples/dynamic-files/stubs/orders-user-alice-orders.json \
   examples/dynamic-files/stubs/orders-user-charlie-orders.json
http ':4000/api/orders?username=charlie'
```

---

## full-crud

A complete blog posts API combining all mockr features. Two config files loaded together from the same directory.

```sh
mockr --config examples/full-crud
```

**Read** (`posts-read.toml`):

```sh
http :4000/api/posts                        # all posts
http ':4000/api/posts?status=published'     # filtered
http ':4000/api/posts?simulate=slow'        # 3s delay (test loading state)
http ':4000/api/posts?simulate=error'       # 500 (test error UI)
```

**Write** (`posts-write.toml`):

```sh
# Create (persisted to stubs/posts.json)
http POST :4000/api/posts id=3 title="New post" body="Hello" author=charlie published:=false

# Admin create (inline JSON with template tokens, not persisted)
http POST :4000/api/posts X-User-Role:admin title="Admin post" body="Elevated"

# Validation error — missing title
http POST :4000/api/posts body="no title"

# Update
http PUT :4000/api/posts/2 id=2 title="Updated" body="..." author=bob published:=true

# Delete
http DELETE :4000/api/posts/3

# Not found → 404
http DELETE :4000/api/posts/99
```

Reset: `git checkout examples/full-crud/stubs/posts.json`

---

## record-mode

Proxy a real API and automatically record responses as stub files. Each new path is recorded once — subsequent requests to the same path are served from the stub (`via=stub`).

```sh
mockr --config examples/record-mode \
      --target https://jsonplaceholder.typicode.com \
      --api-prefix /api \
      --record
```

```sh
http :4000/api/posts
http :4000/api/posts/1
http :4000/api/users
http :4000/api/users/1
```

After recording, serve fully offline (no `--target`, no `--record`):

```sh
mockr --config examples/record-mode --api-prefix /api
```

---

## Directory mode tips

**Split by domain** — each file owns one resource:

```
mocks/
├── auth.toml       # /auth/*
├── users.toml      # /users/*
├── products.toml   # /products/*
└── orders.toml     # /orders/*
```

**Mix formats** — TOML, YAML, and JSON can coexist in the same directory:

```
mocks/
├── auth.toml
├── users.yaml
└── legacy.json
```

**Override order** — files are loaded alphabetically. To force a specific priority, prefix with a number:

```
mocks/
├── 01-base.toml
└── 02-overrides.toml
```
