# Skill: API Design & Integration

## Trigger
- Need to design REST/GraphQL APIs
- Request to integrate external APIs
- API documentation or testing needed

## Prerequisites
- API Gateway MCP server running
- OpenAPI/Swagger spec available (or will generate)

## Workflow

### 1. Register API from Spec
```
api_register(
  name="my-api",
  specUrl="https://api.example.com/openapi.json",
  auth={
    type="bearer",
    token="${API_TOKEN}"
  }
)
```

### 2. Execute API Call
```
api_call(
  api="my-api",
  endpoint="/users",
  method="GET",
  params={ limit: 10, offset: 0 }
)
```

### 3. Auto-Document API
```
api_document(
  source="src/routes/users.ts",
  framework="express"  // or: fastify, gin, fiber
)
```

### 4. Generate API Client
```
api_generate_client(
  api="my-api",
  language="typescript",  // or: python, go, rust
  output="./client"
)
```

## API Design Patterns

### REST Best Practices
```yaml
# Resource naming
/users              # Collection
/users/:id          # Single resource
/users/:id/posts    # Nested resource

# HTTP methods
GET    /users       # List
POST   /users       # Create
GET    /users/:id   # Read
PUT    /users/:id   # Update (full)
PATCH  /users/:id   # Update (partial)
DELETE /users/:id   # Delete

# Query parameters
GET /users?page=1&limit=20&sort=-created_at
GET /users?filter[status]=active
```

### GraphQL Schema
```graphql
type User {
  id: ID!
  email: String!
  name: String!
  posts: [Post!]!
  createdAt: DateTime!
}

type Query {
  users(filter: UserFilter, pagination: PaginationInput): UserConnection!
  user(id: ID!): User
}

type Mutation {
  createUser(input: CreateUserInput!): User!
  updateUser(id: ID!, input: UpdateUserInput!): User!
  deleteUser(id: ID!): Boolean!
}

input UserFilter {
  status: UserStatus
  search: String
}

enum UserStatus {
  ACTIVE
  INACTIVE
  BANNED
}
```

## Security Checklist

- [ ] **Authentication** — API key, JWT, OAuth2
- [ ] **Authorization** — Role-based access control
- [ ] **Rate Limiting** — Prevent abuse
- [ ] **Input Validation** — Sanitize all inputs
- [ ] **CORS** — Configure allowed origins
- [ ] **HTTPS** — Enforce TLS
- [ ] **Logging** — Audit trail for sensitive ops
- [ ] **Error Handling** — Don't expose internals

## Example: User API

```typescript
// Express.js API with auto-documentation
import express from 'express';
import { z } from 'zod';

const app = express();

// Schema validation
const CreateUserSchema = z.object({
  email: z.string().email(),
  name: z.string().min(2),
  password: z.string().min(8)
});

// Route with auto-documentation
app.post('/users', async (req, res) => {
  // Validate input
  const input = CreateUserSchema.parse(req.body);
  
  // Create user
  const user = await createUser(input);
  
  // Return response
  res.status(201).json({
    id: user.id,
    email: user.email,
    name: user.name,
    createdAt: user.createdAt
  });
});

// Auto-generate OpenAPI spec
// api_document(source="src/routes/users.ts", framework="express")
```

## Performance Tips

| Technique | When to Use | Impact |
|-----------|-------------|--------|
| **Pagination** | List endpoints | Prevents large responses |
| **Field Selection** | GraphQL | Reduce payload size |
| **Caching** | Read-heavy APIs | Reduce DB load |
| **Rate Limiting** | Public APIs | Prevent abuse |
| **Connection Pooling** | Database queries | Reduce latency |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "CORS error" | Configure allowed origins in API Gateway |
| "Rate limited" | Implement backoff or request higher limit |
| "Auth failed" | Check token expiry and permissions |
| "Schema mismatch" | Validate input against OpenAPI spec |
