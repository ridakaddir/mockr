# Cross-Endpoint Reference Example

This example demonstrates the new **cross-endpoint reference templating** feature using the `{{ref:...}}` syntax. This allows stub files to reference and include data from other stub files/directories with optional filtering and transformation.

## Features Demonstrated

### 1. **Directory References**
Reference all items from a directory:
```json
"allModels": "{{ref:stubs/models/}}"
```

### 2. **Single File References**  
Reference a specific file:
```json
"primaryModel": "{{ref:stubs/models/1.json}}"
```

### 3. **Filtering by Field Value**
Filter items by field equality (supports dot notation for nested fields):
```json
"activeModels": "{{ref:stubs/models/?filter=status:active}}"
"anthropicModels": "{{ref:stubs/models/?filter=provider:Anthropic}}"
```

### 4. **Template Transformation**
Transform data shape using Go templates to rename/select fields:
```json
"deployedModels": "{{ref:stubs/models/?template=stubs/templates/deployed-model.json}}"
```

### 5. **Combined Filter + Template**
Apply both filtering and transformation:
```json
"deployedModels": "{{ref:stubs/models/?filter=status:active&template=stubs/templates/deployed-model.json}}"
```

### 6. **Inline JSON Support**
Works in both file-based stubs and inline JSON in config:
```toml
[routes.cases.created]
json = '''
{
  "availableModels": "{{ref:stubs/models/?filter=status:active}}"
}
'''
```

## Directory Structure

```
stubs/
├── models/               # Model data files
│   ├── 1.json           # GPT-4 (active)
│   ├── 2.json           # Claude-3.5 (active)  
│   └── 3.json           # Legacy-Model (deprecated)
├── endpoints/            # Endpoint configurations
│   ├── prod.json        # Uses filter + template
│   ├── dev.json         # Shows directory and filter refs
│   └── staging.json     # Shows single file and provider filter
└── templates/
    └── deployed-model.json  # Template for field transformation
```

## Template File Format

Template files use Go's `text/template` syntax with JSON structure:

**`stubs/templates/deployed-model.json`:**
```json
{
  "modelId": "{{.id}}",
  "name": "{{.modelName}}",
  "modelVersion": "{{.version}}"
}
```

This transforms:
```json
{"id": "model-1", "modelName": "GPT-4", "version": "1.0", "status": "active"}
```

Into:
```json
{"modelId": "model-1", "name": "GPT-4", "modelVersion": "1.0"}
```

## Running the Example

1. **Start the server:**
   ```bash
   go run . serve -c examples/cross-refs/mockr.toml
   ```

2. **Test the endpoints:**

   **Get all models:**
   ```bash
   curl http://localhost:8080/models
   ```

   **Get all endpoints (shows cross-references in action):**
   ```bash
   curl http://localhost:8080/endpoints
   ```

   **Get specific endpoint:**
   ```bash
   curl http://localhost:8080/endpoints/prod
   # Shows filtered + transformed models
   
   curl http://localhost:8080/endpoints/dev  
   # Shows all models vs active-only
   
   curl http://localhost:8080/endpoints/staging
   # Shows single file ref + provider filtering
   ```

   **Create new endpoint (inline JSON with refs):**
   ```bash
   curl -X POST http://localhost:8080/endpoints
   # Returns new endpoint with active models included
   ```

## Reference Syntax Summary

| Syntax | Description |
|--------|-------------|
| `{{ref:path/to/dir/}}` | All items from directory (returns array) |
| `{{ref:path/to/file.json}}` | Single file (returns object) |
| `{{ref:path/?filter=field:value}}` | Filter by field equality |
| `{{ref:path/?filter=nested.field:value}}` | Filter by nested field (dot notation) |
| `{{ref:path/?template=tpl.json}}` | Transform using template |
| `{{ref:path/?filter=status:active&template=tpl.json}}` | Both filter and template |

## Key Benefits

1. **DRY Principle**: Define model data once, reference everywhere
2. **Dynamic Relationships**: Endpoints automatically reflect changes to models
3. **Flexible Filtering**: Show only relevant data (e.g., active models)
4. **Shape Transformation**: Present data in different formats via templates
5. **Nested Support**: References can contain references (with circular detection)
6. **Type Safety**: Empty filters return `[]`, not `null`