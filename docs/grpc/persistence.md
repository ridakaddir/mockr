# gRPC Persistence

[Home](../README.md) > [gRPC](README.md) > Persistence

---

gRPC routes support the same directory-based persistence as HTTP routes. Each resource is stored as a separate JSON file.

## Full CRUD example

```toml
# Create item — append to directory
[[grpc_routes]]
match    = "/items.ItemService/CreateItem"
enabled  = true
fallback = "created"

  [grpc_routes.cases.created]
  status   = 0
  file     = "stubs/items/"               # Directory path
  persist  = true
  merge    = "append"
  key      = "itemId"                      # Field used as filename; auto-generated if missing
  defaults = "stubs/defaults/item.json"    # Server-generated fields ({{uuid}}, {{now}})

# List items — directory aggregation
[[grpc_routes]]
match    = "/items.ItemService/ListItems"
enabled  = true
fallback = "list"

  [grpc_routes.cases.list]
  file = "stubs/items/"                    # Returns array of all .json files

# Get item — single file read
[[grpc_routes]]
match    = "/items.ItemService/GetItem"
enabled  = true
fallback = "item"

  [grpc_routes.cases.item]
  file = "stubs/items/{body.itemId}.json"  # Dynamic filename from request

# Update item — shallow merge into existing file
[[grpc_routes]]
match    = "/items.ItemService/UpdateItem"
enabled  = true
fallback = "updated"

  [grpc_routes.cases.updated]
  status  = 0
  file    = "stubs/items/{body.itemId}.json"
  persist = true
  merge   = "update"

# Delete item — remove file
[[grpc_routes]]
match    = "/items.ItemService/DeleteItem"
enabled  = true
fallback = "deleted"

  [grpc_routes.cases.deleted]
  status  = 0
  file    = "stubs/items/{body.itemId}.json"
  persist = true
  merge   = "delete"
```

---

## Field name mapping

Both snake_case (`item_id`) and camelCase (`itemId`) field names in protobuf requests are matched against `key` automatically.

---

## Error codes

| Situation | gRPC Code |
|---|---|
| File/record not found | `5` NOT_FOUND |
| Directory required for append | `3` INVALID_ARGUMENT |
| File read/write error | `13` INTERNAL |

---

## Response body

All persist operations return an empty proto response (`{}`). The gRPC status code signals success or failure.

---

## Example

See [`examples/grpc-directory-persist/`](../../examples/grpc-directory-persist/) for a complete working example.

---

**See also:** [Directory-Based Stubs (HTTP)](../features/directory-stubs.md) | [Configuration](config.md) | [Stubs & Conditions](stubs.md)
