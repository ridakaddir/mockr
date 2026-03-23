# Bare Array Persist Example

This example demonstrates mockr's bare array persist mode, where stub files are top-level JSON arrays instead of wrapped objects.

## Key Benefits

- **API Contract Compatibility**: GET requests return exactly the same format that real APIs use
- **Simplified Structure**: No wrapper objects needed in stub files
- **Direct Array Operations**: All persist operations work directly on the array

## File Structure

```
examples/bare-array-persist/
├── mockr.toml               # Configuration (array_key omitted)
├── stubs/
│   └── todos-bare.json      # Bare array: [{"id": "1"}, ...]
└── README.md                # This file
```

## Comparison with Traditional Mode

| Aspect | Bare Array Mode | Wrapped Mode |
|--------|-----------------|--------------|
| **Stub Format** | `[{"id": "1"}]` | `{"todos": [{"id": "1"}]}` |
| **GET Response** | `[{"id": "1"}]` | `{"todos": [{"id": "1"}]}` |
| **Configuration** | `# array_key omitted` | `array_key = "todos"` |
| **Use Case** | REST APIs returning arrays | APIs with wrapper responses |

## Running the Example

```bash
# Start the server
mockr --config examples/bare-array-persist

# Test the API
curl http://localhost:4000/api/todos
# Returns: [{"id":"1","title":"Buy groceries","done":false}, ...]

# Add a new todo
curl -X POST http://localhost:4000/api/todos \
  -H "Content-Type: application/json" \
  -d '{"id":"4","title":"Test feature","done":false}'

# Update a todo
curl -X PUT http://localhost:4000/api/todos/2 \
  -H "Content-Type: application/json" \
  -d '{"id":"2","title":"Updated task","done":true}'

# Delete a todo
curl -X DELETE http://localhost:4000/api/todos/3
```

## API Behavior

- **GET /api/todos**: Returns the bare array from `stubs/todos-bare.json`
- **POST /api/todos**: Appends new todo to the array
- **PUT /api/todos/{id}**: Updates todo with matching `id`
- **PATCH /api/todos/{id}**: Partially updates todo with matching `id`
- **DELETE /api/todos/{id}**: Removes todo with matching `id`

All mutations persist to `stubs/todos-bare.json` immediately.

## Configuration Details

The key difference is the absence of `array_key` in the route configuration:

```toml
[routes.cases.created]
status  = 201
file    = "stubs/todos-bare.json"
persist = true
merge   = "append"
# array_key omitted — operates on bare array
```

When `array_key` is omitted, mockr automatically detects that the stub file is a bare array and operates directly on it.

## Resetting Data

To reset the todos back to the original state:

```bash
git checkout examples/bare-array-persist/stubs/todos-bare.json
```