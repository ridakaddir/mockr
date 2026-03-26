# Cross-Endpoint References

[Home](../README.md) > [Features](README.md) > Cross-Endpoint References

---

Cross-endpoint references allow you to include data from other stub files in your responses using the `{{ref:...}}` syntax. This enables building interconnected mock APIs where endpoints reference and share data with optional filtering and transformation.

## Syntax

```
{{ref:path}}                           # Reference all items from a directory or single file
{{ref:path?filter=field:value}}        # Filter by field equality
{{ref:path?template=template.json}}    # Transform data shape using Go templates
{{ref:path?filter=field:value&template=template.json}}  # Both filter and transform
```

---

## Basic Usage

### Directory References

Reference all JSON files in a directory:

```json
{
  "id": "endpoint-1",
  "name": "My Endpoint",
  "allModels": "{{ref:stubs/models/}}"
}
```

### Single File References

Reference a specific JSON file:

```json
{
  "id": "endpoint-1",
  "name": "My Endpoint", 
  "primaryModel": "{{ref:stubs/models/gpt-4.json}}"
}
```

---

## Filtering

Filter referenced data by field values using `?filter=field:value` syntax.

### Simple Field Filtering

```json
{
  "activeModels": "{{ref:stubs/models/?filter=status:active}}",
  "productionEndpoints": "{{ref:stubs/endpoints/?filter=environment:production}}"
}
```

### Nested Field Filtering

Use dot notation to filter by nested object properties:

```json
{
  "adminUsers": "{{ref:stubs/users/?filter=user.role:admin}}",
  "cloudProviders": "{{ref:stubs/providers/?filter=config.type:cloud}}"
}
```

### Multiple Filters

Chain multiple filters with `&` (all must match):

```json
{
  "filtered": "{{ref:stubs/items/?filter=status:active&filter=type:premium}}"
}
```

---

## Template Transformation

Transform the shape of referenced data using Go template files to rename fields, select specific properties, or restructure the output.

### Template File

Create a template file using Go's `text/template` syntax:

**`stubs/templates/model-summary.json`:**
```json
{
  "modelId": "{{.id}}",
  "displayName": "{{.modelName}}",
  "version": "{{.version}}"
}
```

### Usage

Reference the template to transform data:

```json
{
  "deployedModels": "{{ref:stubs/models/?template=stubs/templates/model-summary.json}}"
}
```

This transforms:
```json
{"id": "gpt-4", "modelName": "GPT-4", "version": "1.0", "status": "active", "provider": "OpenAI"}
```

Into:
```json
{"modelId": "gpt-4", "displayName": "GPT-4", "version": "1.0"}
```

---

## Combining Filter and Template

Apply both filtering and transformation:

```json
{
  "activeDeployedModels": "{{ref:stubs/models/?filter=status:active&template=stubs/templates/model-summary.json}}"
}
```

This workflow:
1. Loads all models from `stubs/models/`
2. Filters to only `status:active` models
3. Transforms each model using the template
4. Returns the transformed array

---

## Advanced Features

### Nested References

Referenced files can contain their own `{{ref:...}}` tokens (with circular reference detection):

**`stubs/endpoints/prod.json`:**
```json
{
  "id": "prod-endpoint",
  "models": "{{ref:stubs/models/?filter=status:active}}",
  "config": "{{ref:stubs/configs/prod.json}}"
}
```

**`stubs/configs/prod.json`:**
```json
{
  "environment": "production",
  "secrets": "{{ref:stubs/secrets/?filter=env:prod}}"
}
```

### Empty Results

When filters match no items, an empty array `[]` is returned (never `null`):

```json
{
  "noMatches": "{{ref:stubs/models/?filter=status:nonexistent}}"
}
// Returns: {"noMatches": []}
```

### Error Handling

- **Missing files**: Clear error with file path
- **Circular references**: Automatic detection and prevention  
- **Invalid syntax**: Descriptive parsing errors
- **Template errors**: Go template compilation/execution errors

---

## Example: E-commerce API

**Directory Structure:**
```
stubs/
├── products/
│   ├── 1.json          # {"id": "1", "name": "Laptop", "category": "electronics", "inStock": true}
│   └── 2.json          # {"id": "2", "name": "Book", "category": "books", "inStock": false}
├── users/
│   └── alice.json      # {"id": "alice", "name": "Alice", "role": "admin"}
├── orders/
│   └── order-123.json
└── templates/
    └── product-summary.json
```

**`stubs/orders/order-123.json`:**
```json
{
  "orderId": "order-123",
  "customerId": "alice",
  "customer": "{{ref:stubs/users/alice.json}}",
  "items": "{{ref:stubs/products/?filter=inStock:true&template=stubs/templates/product-summary.json}}",
  "allProducts": "{{ref:stubs/products/}}"
}
```

**`stubs/templates/product-summary.json`:**
```json
{
  "productId": "{{.id}}",
  "productName": "{{.name}}"
}
```

**Result:**
```json
{
  "orderId": "order-123",
  "customerId": "alice", 
  "customer": {"id": "alice", "name": "Alice", "role": "admin"},
  "items": [{"productId": "1", "productName": "Laptop"}],
  "allProducts": [
    {"id": "1", "name": "Laptop", "category": "electronics", "inStock": true},
    {"id": "2", "name": "Book", "category": "books", "inStock": false}
  ]
}
```

---

## Usage in Config

Cross-endpoint references work in both file-based stubs and inline JSON:

### File-Based Stubs

```toml
[[routes]]
method = "GET"
match = "/endpoints"

  [routes.cases.list]
  file = "stubs/endpoints/"  # Files in this directory can contain {{ref:...}}
```

### Inline JSON

```toml
[[routes]]
method = "POST" 
match = "/endpoints"

  [routes.cases.created]
  status = 201
  json = '''
  {
    "id": "{{uuid}}",
    "createdAt": "{{now}}",
    "availableModels": "{{ref:stubs/models/?filter=status:active}}"
  }
  '''
```

---

## Best Practices

1. **Organize by domain**: Group related stubs in directories (`users/`, `products/`, etc.)

2. **Use descriptive template names**: `deployed-model.json`, `user-summary.json`

3. **Filter for relevance**: Only include data that makes sense for the context

4. **Template for clean APIs**: Transform internal data structures to clean API responses

5. **Avoid deep nesting**: Keep reference chains shallow for maintainability

6. **Handle empty cases**: Design UIs to handle empty arrays gracefully

---

**See also:** [Template Tokens](template-tokens.md) | [Directory-Based Stubs](directory-stubs.md) | [Dynamic Files](dynamic-files.md)