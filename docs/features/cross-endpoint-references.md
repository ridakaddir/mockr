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
"$spread": "{{ref:path}}"              # Spread object properties into containing object
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

## Security Considerations

Cross-endpoint references are restricted for security:

- **Path Traversal Prevention**: References cannot use `../` or absolute paths
- **Config Directory Boundary**: All referenced files must be within the config directory
- **Template Path Validation**: Template paths follow the same restrictions
- **No External File Access**: References cannot read files outside the project directory

Examples of blocked references:
```json
{"data": "{{ref:../secret.json}}"}           // ❌ Directory traversal
{"data": "{{ref:/etc/passwd}}"}              // ❌ Absolute path  
{"data": "{{ref:data/?template=../tpl.json}}"} // ❌ Template traversal
```

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

## Usage in Defaults Files

Cross-endpoint references support **dynamic placeholders** in defaults files, allowing you to reference different stub files based on request data. This enables environment-specific, tenant-specific, or user-specific data loading in both regular operations and background transitions.

### Dynamic Placeholder Syntax

Use these placeholders inside `{{ref:...}}` tokens:

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{.field}` | Request body field | `{{ref:stubs/{.endpointId}/models/}}` |
| `{.nested.field}` | Nested body field | `{{ref:stubs/{.config.env}/models/}}` |
| `{path.param}` | URL path parameter | `{{ref:stubs/{path.tenantId}/users/}}` |
| `{query.param}` | Query parameter | `{{ref:stubs/{query.version}/models/}}` |
| `{header.Name}` | Request header | `{{ref:stubs/{header.X-Tenant-Id}/models/}}` |

### Example: Environment-Specific Defaults

**Config:**
```toml
[[routes]]
method = "POST"
match = "/api/endpoints"

  [routes.cases.created]
  status = 201
  persist = true
  merge = "append"
  key = "endpointId"
  defaults = "defaults/endpoint.json"
```

**`defaults/endpoint.json`:**
```json
{
  "status": "Deploying",
  "environment": "{.environment}",
  "models": "{{ref:models/{.environment}/}}",
  "config": "{{ref:configs/{path.tenantId}/{.environment}.json}}"
}
```

**Request:**
```bash
POST /api/endpoints
{
  "endpointId": "ep-123",
  "environment": "staging"
}
```

This resolves to defaults that load:
- `models/staging/` directory (environment-specific models)
- `configs/tenant-a/staging.json` file (tenant + environment config)

### Live Directory References

When a defaults file contains a `{{ref:...}}` token that points to a **directory** (path ending with `/`), the reference is preserved as a live token in the created file rather than being resolved at creation time. This means directory references resolve dynamically on every read, so they always reflect the current state of the referenced directory.

For example, if `defaults/endpoint.json` contains:
```json
{
  "endpointId": "{{uuid}}",
  "deployedModels": "{{ref:stubs/deployments/{.endpointId}/?template=stubs/templates/deployed-model.json}}"
}
```

When an endpoint is created, the `deployedModels` field is stored as `"{{ref:stubs/deployments/ep-123/?template=...}}"` (with `{.endpointId}` resolved to the concrete value). Each subsequent GET request resolves this reference against the current contents of the deployment directory — so newly created deployments appear immediately.

File-based refs (not ending with `/`) are still resolved at creation time, since they point to static data.

### Background Transitions

Dynamic refs work in **background transitions** by storing the original request context when the transition is scheduled:

**Config with transitions:**
```toml
[[routes]]
method = "POST"
match = "/api/deployments"
fallback = "created"

  [routes.transitions]
  # Initial state (from request)
  [[routes.transitions]]
  case = "deploying"
  duration = 30
  
  # After 30s, transition to ready
  [[routes.transitions]]  
  case = "ready"

  [routes.cases.created]
  status = 201
  persist = true
  merge = "append"  
  defaults = "defaults/deployment.json"

  [routes.cases.ready]
  persist = true
  merge = "update"
  defaults = "defaults/deployment-ready.json"  # Uses dynamic refs!
```

**`defaults/deployment-ready.json`:**
```json
{
  "status": "Ready",
  "endpoint": "{{ref:endpoints/{.endpointId}/status.json}}",
  "models": "{{ref:models/{.environment}/ready/}}"
}
```

When the background transition fires after 30 seconds, the dynamic placeholders `{.endpointId}` and `{.environment}` are resolved using the **original request data** that was stored when the transition was scheduled.

### Error Handling

Dynamic refs use **strict error handling** for data integrity:

- **Missing field**: Error if placeholder field not found in request
- **Empty value**: Error if placeholder resolves to empty string  
- **Missing file**: Error for non-existent file references
- **Missing directory**: Returns empty array `[]` (standard behavior)

```json
// ❌ These cause errors:
{"data": "{{ref:stubs/{.missingField}/}}"}      // Field not in request
{"data": "{{ref:stubs/{.emptyField}/}}"}        // Field exists but empty
{"data": "{{ref:stubs/{.field}/missing.json}}"} // File doesn't exist

// ✅ This succeeds (returns empty array):
{"data": "{{ref:stubs/{.field}/missing-dir/}}"} // Directory doesn't exist
```

### Advanced Example: Multi-Tenant API

**Directory Structure:**
```
stubs/
├── tenants/
│   ├── acme/
│   │   ├── prod/
│   │   │   └── models/
│   │   │       └── gpt-4.json
│   │   └── staging/
│   └── corp/
└── defaults/
    ├── tenant-deployment.json
    └── tenant-endpoint.json
```

**`defaults/tenant-deployment.json`:**
```json
{
  "tenantId": "{header.X-Tenant-Id}",
  "environment": "{.environment}",
  "models": "{{ref:tenants/{header.X-Tenant-Id}/{.environment}/models/}}",
  "config": "{{ref:tenants/{header.X-Tenant-Id}/config.json}}",
  "metadata": {
    "createdAt": "{{now}}",
    "deploymentId": "{{uuid}}"
  }
}
```

**Request:**
```bash
POST /api/deployments
X-Tenant-Id: acme

{
  "endpointId": "ep-456", 
  "environment": "prod"
}
```

**Resolved defaults:**
```json
{
  "tenantId": "acme",
  "environment": "prod", 
  "models": [{"id": "gpt-4", "name": "GPT-4", "status": "ready"}],
  "config": {"tier": "premium", "limits": {"requests": 10000}},
  "metadata": {
    "createdAt": "2024-03-25T15:30:00Z",
    "deploymentId": "550e8400-e29b-41d4-a716-446655440000"
  }
}
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

## Object Spreading

Object spreading allows you to merge the properties of a referenced object directly into the containing object using the `$spread` field with a `{{ref:...}}` reference. This is particularly useful for combining data from multiple sources into a flat response structure.

### Syntax

```
"$spread": "{{ref:path}}"                    # Spread all properties from referenced object
"$spread": "{{ref:path?filter=field:value}}" # Spread filtered properties
"$spread": "{{ref:path?template=template.json}}" # Spread transformed properties
```

### Basic Spreading

Spread all properties from a referenced file:

```json
{
  "id": "123",
  "$spread": "{{ref:stubs/endpoints/{path.endpointId}.json}}",
  "deployedModels": "{{ref:stubs/deployments/{path.endpointId}/?template=stubs/templates/deployed-model.json}}"
}
```

**Referenced file** (`stubs/endpoints/ep-123.json`):
```json
{
  "endpointId": "ep-123",
  "name": "Production API",
  "region": "us-east-1",
  "status": "active"
}
```

**Result** (flat structure):
```json
{
  "id": "123",
  "endpointId": "ep-123",
  "name": "Production API", 
  "region": "us-east-1",
  "status": "active",
  "deployedModels": [...]
}
```

### Property Override

Explicit properties override spread properties when there are conflicts:

```json
{
  "$spread": "{{ref:stubs/defaults/user.json}}",
  "status": "override"  // This overrides any "status" from the spread object
}
```

### Spreading with Dynamic Placeholders

Combine spreading with path parameters and other dynamic placeholders:

```json
{
  "$spread": "{{ref:stubs/endpoints/{path.endpointId}.json}}",
  "environment": "{path.env}",
  "deployedModels": "{{ref:stubs/deployments/{path.endpointId}/?filter=status:active}}"
}
```

### Use Cases

**1. Endpoint Enhancement**: Add computed fields to existing endpoint data
```json
{
  "$spread": "{{ref:stubs/endpoints/{path.endpointId}.json}}",
  "deployedModels": "{{ref:stubs/deployments/{path.endpointId}/}}",
  "lastUpdated": "{{now}}"
}
```

**2. Configuration Merging**: Combine base configuration with environment-specific overrides
```json
{
  "$spread": "{{ref:stubs/configs/base.json}}",
  "environment": "{path.env}",
  "version": "1.0.0"
}
```

**3. Response Composition**: Build complex responses from multiple data sources
```json
{
  "$spread": "{{ref:stubs/users/{path.userId}.json}}",
  "permissions": "{{ref:stubs/roles/{.role}/permissions.json}}",
  "preferences": "{{ref:stubs/users/{path.userId}/preferences.json}}"
}
```

**4. Nested Object Spreading**: Spread properties into nested structures
```json
{
  "id": "123",
  "profile": {
    "$spread": "{{ref:stubs/profiles/{path.userId}.json}}",
    "active": true
  },
  "status": "online"
}
```

### Limitations

1. **Objects only**: Can only spread objects (`map[string]interface{}`), not arrays or primitives
2. **Per-object scope**: Each `$spread` only merges into its immediate containing object, but you can use `$spread` inside nested objects for nested spreading
3. **Key conflicts**: Later properties override earlier ones (explicit > spread)
4. **Processing order**: Spread resolution happens before regular reference resolution

### Error Handling

**Invalid Syntax:**
```json
{
  "$spread": "invalid-value"  // ERROR: Must be {{ref:...}} token
}
```

**Non-Object Reference:**
```json
{
  "$spread": "{{ref:stubs/arrays/list.json}}"  // ERROR: Cannot spread array
}
```

The above will result in an error: `$spread ref must resolve to an object, got []interface {}`

**Invalid Type:**
```json
{
  "$spread": 123  // ERROR: Must be string
}
```

Result: `$spread field must be a string, got int`

---

**See also:** [Template Tokens](template-tokens.md) | [Directory-Based Stubs](directory-stubs.md) | [Dynamic Files](dynamic-files.md)