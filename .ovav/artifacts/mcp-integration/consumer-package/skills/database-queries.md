# Skill: Database Queries & Management

## Trigger
- Need to query databases (PostgreSQL, SQLite)
- Database schema inspection
- Data analysis or migration

## Prerequisites
- Database MCP server configured
- Connection string set in `DATABASE_URL`

## Supported Databases

| Database | MCP Server | Use Case |
|----------|------------|----------|
| PostgreSQL | `@modelcontextprotocol/server-postgres` | Production, complex queries |
| SQLite | `@modelcontextprotocol/server-sqlite` | Local, analytics, testing |

## Workflow

### 1. PostgreSQL Query
```
postgres_query(
  connection_string="${DATABASE_URL}",
  sql="SELECT * FROM users WHERE status = 'active' LIMIT 10"
)
```

### 2. SQLite Query
```
sqlite_query(
  database_path="./data/analytics.db",
  sql="SELECT date, COUNT(*) as events FROM events GROUP BY date"
)
```

### 3. Schema Inspection
```
postgres_schema(
  connection_string="${DATABASE_URL}",
  table="users"
)
```

### 4. Safe Query Builder
```
postgres_safe_query(
  connection_string="${DATABASE_URL}",
  table="users",
  operation="select",
  columns=["id", "email", "name"],
  filters={ status: "active" },
  order_by="-created_at",
  limit=10
)
```

## Query Examples

### Basic SELECT
```sql
-- Get active users
SELECT id, email, name, created_at
FROM users
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT 10;

-- Get user with posts
SELECT u.id, u.name, p.title, p.created_at
FROM users u
LEFT JOIN posts p ON p.user_id = u.id
WHERE u.id = 123;
```

### Aggregation
```sql
-- Daily active users
SELECT 
  DATE(created_at) as date,
  COUNT(DISTINCT user_id) as dau
FROM events
WHERE event_type = 'login'
  AND created_at > NOW() - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY date;

-- Top users by posts
SELECT 
  u.id,
  u.name,
  COUNT(p.id) as post_count,
  AVG(p.likes) as avg_likes
FROM users u
JOIN posts p ON p.user_id = u.id
GROUP BY u.id, u.name
ORDER BY post_count DESC
LIMIT 10;
```

### Mutation (with safety checks)
```sql
-- Update with confirmation
BEGIN;

-- Preview changes
SELECT * FROM users WHERE id = 123 FOR UPDATE;

-- Apply update
UPDATE users 
SET status = 'inactive', updated_at = NOW()
WHERE id = 123;

-- Verify
SELECT * FROM users WHERE id = 123;

COMMIT;
```

## Safety Rules

### Read-Only by Default
```typescript
// ✅ Safe - Read-only query
postgres_query(
  sql="SELECT * FROM users WHERE id = $1",
  params=[123]
)

// ⚠️ Caution - Write operation (requires confirmation)
postgres_query(
  sql="UPDATE users SET status = 'inactive' WHERE id = $1",
  params=[123],
  confirm=true  // Required for writes
)
```

### Parameterized Queries
```typescript
// ✅ Safe - Parameterized
postgres_query(
  sql="SELECT * FROM users WHERE email = $1",
  params=[userEmail]
)

// ❌ Dangerous - SQL injection risk
postgres_query(
  sql=`SELECT * FROM users WHERE email = '${userEmail}'`
)
```

### Transaction Safety
```sql
-- Always use transactions for writes
BEGIN;
  -- Perform write
  INSERT INTO audit_log (action, user_id) VALUES ('update', 123);
  UPDATE users SET status = 'inactive' WHERE id = 123;
COMMIT;

-- Rollback on error
BEGIN;
  -- Risky operation
  DELETE FROM old_data WHERE created_at < '2020-01-01';
  -- If error, rollback
ROLLBACK;
```

## Performance Optimization

### Indexing
```sql
-- Add index for frequent queries
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_posts_user_id ON posts(user_id);

-- Composite index for complex queries
CREATE INDEX idx_posts_user_status ON posts(user_id, status);
```

### Query Analysis
```sql
-- PostgreSQL: Analyze query plan
EXPLAIN ANALYZE
SELECT * FROM users WHERE status = 'active';

-- SQLite: Analyze query
EXPLAIN QUERY PLAN
SELECT * FROM users WHERE status = 'active';
```

### Connection Pooling
```typescript
// PostgreSQL connection pool
import { Pool } from 'pg';

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
  max: 20,           // Max connections
  idleTimeoutMillis: 30000,
  connectionTimeoutMillis: 2000,
});

// Use pool for queries
const result = await pool.query('SELECT * FROM users');
```

## Common Patterns

### Pagination
```sql
-- Cursor-based pagination (recommended)
SELECT * FROM users
WHERE id > $last_id
ORDER BY id
LIMIT 20;

-- Offset-based pagination (simple but slower)
SELECT * FROM users
ORDER BY id
LIMIT 20 OFFSET $page * 20;
```

### Soft Delete
```sql
-- Instead of DELETE, mark as deleted
UPDATE users SET deleted_at = NOW() WHERE id = 123;

-- Query active records
SELECT * FROM users WHERE deleted_at IS NULL;
```

### Audit Trail
```sql
-- Track all changes
CREATE TABLE audit_log (
  id SERIAL PRIMARY KEY,
  table_name TEXT NOT NULL,
  record_id INTEGER NOT NULL,
  action TEXT NOT NULL, -- INSERT, UPDATE, DELETE
  old_data JSONB,
  new_data JSONB,
  user_id INTEGER,
  created_at TIMESTAMP DEFAULT NOW()
);
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Connection refused" | Check DATABASE_URL and network |
| "Permission denied" | Verify database user permissions |
| "Table doesn't exist" | Run schema inspection first |
| "Query timeout" | Add indexes or optimize query |
| "Connection pool exhausted" | Increase pool size or add connection recycling |
