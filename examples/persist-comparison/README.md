# Persist Modes Comparison

This example demonstrates both persist approaches side-by-side to help you understand when to use each mode.

## Overview

Mockr supports two persist modes to match different API response formats:

### Wrapped Object Mode (Traditional)
- **Config:** `wrapped.toml`
- **Stub:** `stubs/todos-wrapped.json` → `{"todos": [...]}`
- **GET Response:** Returns entire object including wrapper
- **Use Case:** When your API returns wrapped responses

### Bare Array Mode (New)  
- **Config:** `bare.toml`
- **Stub:** `stubs/todos-bare.json` → `[...]`
- **GET Response:** Returns bare array directly
- **Use Case:** When your API returns bare arrays (common REST pattern)

## Key Differences

| Aspect | Wrapped Mode | Bare Array Mode |
|--------|--------------|-----------------|
| **Stub format** | `{"todos": [...]}` | `[...]` |
| **GET response** | `{"todos": [...]}` | `[...]` |
| **API contract** | Matches wrapped APIs | Matches bare array APIs |
| **Configuration** | `array_key = "todos"` | `# array_key omitted` |
| **File validation** | Requires object with array field | Requires top-level array |

## Running Examples

### Wrapped Mode (Traditional)
```bash
# Terminal 1
mockr --config examples/persist-comparison/wrapped.toml

# Terminal 2 - Test requests
curl http://localhost:4000/api/todos
# Returns: {"todos": [{"id":"1","title":"Buy groceries","done":false}, ...]}

curl -X POST http://localhost:4000/api/todos \
  -H "Content-Type: application/json" \
  -d '{"id":"4","title":"New task","done":false}'
```

### Bare Array Mode (New)
```bash
# Terminal 1  
mockr --config examples/persist-comparison/bare.toml

# Terminal 2 - Test requests
curl http://localhost:4000/api/todos
# Returns: [{"id":"1","title":"Buy groceries","done":false}, ...]

curl -X POST http://localhost:4000/api/todos \
  -H "Content-Type: application/json" \
  -d '{"id":"4","title":"New task","done":false}'
```

## Configuration Comparison

### Wrapped Mode Configuration
```toml
[routes.cases.created]
status    = 201
file      = "stubs/todos-wrapped.json"
persist   = true
merge     = "append"
array_key = "todos"    # Required: specifies array field
```

### Bare Array Mode Configuration  
```toml
[routes.cases.created]
status  = 201
file    = "stubs/todos-bare.json"
persist = true
merge   = "append"
# array_key omitted — operates on bare array
```

## File Format Comparison

### Wrapped Object Stub (`todos-wrapped.json`)
```json
{
  "todos": [
    {"id": "1", "title": "Buy groceries", "done": false},
    {"id": "2", "title": "Write unit tests", "done": false},
    {"id": "3", "title": "Ship the feature", "done": true}
  ]
}
```

### Bare Array Stub (`todos-bare.json`)
```json
[
  {"id": "1", "title": "Buy groceries", "done": false},
  {"id": "2", "title": "Write unit tests", "done": false},
  {"id": "3", "title": "Ship the feature", "done": true}
]
```

## Validation and Error Handling

Mockr validates that your configuration matches your stub file format:

- **Wrapped mode** (with `array_key`): Stub file must be a JSON object
- **Bare array mode** (without `array_key`): Stub file must be a JSON array

### Example Error Messages

**Mismatched configuration:**
```
Error: stub file "stubs/todos.json" contains a JSON object but array_key is not specified. 
Either provide array_key to specify which field contains the array, or convert the file to a bare JSON array
```

**Wrong field name:**
```
Error: array_key "items" not found in stub file. Available keys: ["todos", "meta"]
```

## When to Use Each Mode

### Use Wrapped Object Mode When:
- Your real API returns responses like: `{"data": [...], "meta": {...}}`
- You need multiple fields in your response
- You're migrating from an existing wrapped setup
- Your API follows envelope/wrapper patterns

### Use Bare Array Mode When:
- Your real API returns bare arrays: `[{...}, {...}]`
- You want API contract compatibility  
- You're building REST APIs following standard patterns
- You want simpler stub file management

## Migration Between Modes

### From Wrapped to Bare Array:
1. Extract the array from your stub file:
   ```bash
   # Before: {"todos": [...]}
   # After: [...]
   jq '.todos' stubs/todos-wrapped.json > stubs/todos-bare.json
   ```
2. Remove `array_key` from your configuration
3. Update file references to point to the new stub

### From Bare Array to Wrapped:
1. Wrap your array in an object:
   ```bash
   # Before: [...]  
   # After: {"todos": [...]}
   jq '{todos: .}' stubs/todos-bare.json > stubs/todos-wrapped.json
   ```
2. Add `array_key = "todos"` to your configuration
3. Update file references to point to the new stub

## Resetting Data

To reset both examples back to original state:
```bash
git checkout examples/persist-comparison/stubs/
```