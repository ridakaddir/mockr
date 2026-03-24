# Directory-Based Stub Storage Example

This example demonstrates the new directory-based approach for stateful mocking where each resource is stored as a separate JSON file.

## Features Demonstrated

- **Directory aggregation** - GET list operations aggregate all .json files in a directory
- **Individual file operations** - POST/PATCH/DELETE work on individual files
- **Auto-UUID generation** - Automatically generates UUIDs for new resources when ID is missing
- **Dynamic file paths** - Uses path parameters to resolve specific files

## How It Works

### File Structure

```
stubs/
└── users/
    ├── 1.json     # {"userId": "1", "name": "Alice Johnson", ...}
    ├── 2.json     # {"userId": "2", "name": "Bob Smith", ...}
    └── 3.json     # {"userId": "3", "name": "Charlie Davis", ...}
```

### API Operations

1. **GET /users** - Aggregates all files in `stubs/users/` into a JSON array
2. **POST /users** - Creates a new file in `stubs/users/` using the `userId` field as filename
3. **GET /users/{userId}** - Reads the specific file `stubs/users/{userId}.json`
4. **PATCH /users/{userId}** - Shallow-merges request body into the existing file
5. **DELETE /users/{userId}** - Removes the file `stubs/users/{userId}.json`

## Running the Example

```bash
# Start the mock server
mockr --config examples/directory-stubs

# List users (directory aggregation)
curl http://localhost:8080/users

# Get specific user
curl http://localhost:8080/users/1

# Create new user (with explicit ID)
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"userId": "4", "name": "Diana Wilson", "email": "diana@example.com", "role": "user"}'

# Create new user (auto-generate ID)
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Eve Brown", "email": "eve@example.com", "role": "user"}'

# Update user (shallow merge)
curl -X PATCH http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -d '{"email": "alice.johnson@newdomain.com", "active": false}'

# Delete user
curl -X DELETE http://localhost:8080/users/3
```

## Benefits

- **Single source of truth** - Each resource is a separate file
- **Version control friendly** - Easy to track changes to individual resources
- **Scalable** - No size limits imposed by single-file arrays
- **Intuitive** - File structure mirrors API structure
- **Nested resources** - Supports subdirectories for hierarchical data

## Comparison to Array-Based Approach

| Aspect | Directory-Based | Array-Based (Old) |
|--------|----------------|-------------------|
| File structure | One file per resource | All resources in one array |
| GET list | Aggregates directory | Returns array from file |
| POST create | Creates new file | Appends to array |
| PATCH update | Merges into file | Finds & merges array element |
| DELETE | Removes file | Removes array element |
| Scalability | Unlimited | Limited by file size |
| Version control | Clean diffs per resource | Noisy diffs on shared file |