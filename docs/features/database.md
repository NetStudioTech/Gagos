# Database Tools

GAGOS includes built-in clients for PostgreSQL, MySQL, and Redis databases.

## PostgreSQL

### Connecting

1. Open the PostgreSQL window
2. Enter connection details:
   - Host (e.g., `postgres.default.svc`)
   - Port (default: 5432)
   - Database name
   - Username
   - Password
   - SSL mode (disable/require/verify-full)
3. Click "Connect"

### Features

#### Server Info

After connecting, view server information:
- PostgreSQL version
- Database size
- Connection count
- Uptime

#### Query Execution

1. Enter SQL in the query editor
2. Click "Execute"
3. View results in table format

**Tips:**
- Use `LIMIT` for large result sets
- Multiple statements supported (separated by `;`)

#### Schema Browser

View database structure:
- Tables with columns and types
- Indexes
- Views
- Functions

#### Database Dump

Export database:
1. Click "Dump" button
2. Select options (schema only, data only, or both)
3. Download SQL file or copy to clipboard

### API

```bash
# Connect and get info
curl -X POST http://localhost:8080/api/v1/database/postgres/connect \
  -H "Content-Type: application/json" \
  -d '{
    "host":"localhost",
    "port":5432,
    "database":"mydb",
    "user":"postgres",
    "password":"secret"
  }'

# Execute query
curl -X POST http://localhost:8080/api/v1/database/postgres/query \
  -H "Content-Type: application/json" \
  -d '{
    "host":"localhost",
    "port":5432,
    "database":"mydb",
    "user":"postgres",
    "password":"secret",
    "query":"SELECT * FROM users LIMIT 10"
  }'
```

---

## MySQL

### Connecting

1. Open the MySQL window
2. Enter connection details:
   - Host
   - Port (default: 3306)
   - Database name
   - Username
   - Password
3. Click "Connect"

### Features

#### Server Info

View MySQL server information:
- MySQL version
- Database size
- Connection count
- Character set

#### Query Execution

Same as PostgreSQL:
1. Enter SQL query
2. Click "Execute"
3. View tabular results

#### Schema Browser

View database structure:
- Tables with columns
- Indexes
- Views

#### Database Dump

Export with mysqldump format:
1. Click "Dump"
2. Select options
3. Download or copy SQL

### API

```bash
# Connect and get info
curl -X POST http://localhost:8080/api/v1/database/mysql/connect \
  -H "Content-Type: application/json" \
  -d '{
    "host":"localhost",
    "port":3306,
    "database":"mydb",
    "user":"root",
    "password":"secret"
  }'

# Execute query
curl -X POST http://localhost:8080/api/v1/database/mysql/query \
  -H "Content-Type: application/json" \
  -d '{
    "host":"localhost",
    "port":3306,
    "database":"mydb",
    "user":"root",
    "password":"secret",
    "query":"SELECT * FROM users"
  }'
```

---

## Redis

### Connecting

1. Open the Redis window
2. Enter connection details:
   - Host
   - Port (default: 6379)
   - Password (optional)
   - Database number (default: 0)
3. Click "Connect"

### Features

#### Server Info

View Redis server information:
- Redis version
- Memory usage
- Connected clients
- Keyspace statistics

#### Key Browser

Browse Redis keys:
1. Enter pattern (e.g., `user:*`)
2. Click "Scan"
3. View matching keys

Click a key to view its value:
- String values displayed directly
- Hash, List, Set, ZSet shown with structure

#### Command Execution

Execute Redis commands:
1. Enter command (e.g., `GET mykey`)
2. Click "Execute"
3. View response

**Common commands:**
- `GET key`
- `SET key value`
- `HGETALL hashkey`
- `LRANGE listkey 0 -1`
- `SMEMBERS setkey`
- `INFO`

#### Cluster Info

For Redis Cluster deployments:
1. Click "Cluster" tab
2. View cluster nodes
3. See slot distribution

### API

```bash
# Connect and get info
curl -X POST http://localhost:8080/api/v1/database/redis/connect \
  -H "Content-Type: application/json" \
  -d '{
    "host":"localhost",
    "port":6379,
    "password":"secret",
    "db":0
  }'

# Scan keys
curl -X POST http://localhost:8080/api/v1/database/redis/scan \
  -H "Content-Type: application/json" \
  -d '{
    "host":"localhost",
    "port":6379,
    "pattern":"user:*"
  }'

# Execute command
curl -X POST http://localhost:8080/api/v1/database/redis/exec \
  -H "Content-Type: application/json" \
  -d '{
    "host":"localhost",
    "port":6379,
    "command":"GET mykey"
  }'
```

---

## Security Notes

1. **Credentials are not stored** - Connection details are only used for the current session
2. **Use internal networking** - Connect via Kubernetes service names when possible
3. **Limit permissions** - Use read-only database users when appropriate
4. **Enable SSL** - Use encrypted connections for production databases
