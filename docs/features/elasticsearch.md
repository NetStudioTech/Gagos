# Elasticsearch

GAGOS provides a web interface for managing Elasticsearch clusters, similar to Elasticvue.

## Connecting

1. Open the Elasticsearch window
2. Configure connection:
   - URL (e.g., `http://elasticsearch:9200`)
   - Username (optional)
   - Password (optional)
3. Click "Connect"

**Presets available:**
- Localhost (http://localhost:9200)
- Custom configuration

## Features

### Cluster Tab

View cluster health and statistics:

#### Cluster Health
- Status (green/yellow/red)
- Node count
- Active shards
- Relocating shards
- Unassigned shards

#### Cluster Stats
- Document count
- Total storage size
- Index count
- Node details

### Indices Tab

Manage Elasticsearch indices:

#### Index List

View all indices with:
- Name
- Health status
- Document count
- Primary store size
- Replica count

#### Index Operations

**Create Index:**
1. Click "Create Index"
2. Enter index name
3. Optionally configure settings/mappings
4. Click "Create"

**Delete Index:**
1. Click "Delete" on an index
2. Confirm deletion
3. Index is removed

**Refresh Index:**
Click "Refresh" to make recent documents searchable immediately.

**View Mapping:**
Click "Mapping" to see the index field mappings.

**View Settings:**
Click "Settings" to see index configuration.

### Documents Tab

Search and browse documents:

#### Search Documents

1. Select an index
2. Enter search query (Lucene syntax or JSON)
3. Click "Search"
4. View matching documents

**Query examples:**
- Simple: `status:active`
- Phrase: `"error message"`
- Range: `timestamp:[2024-01-01 TO *]`
- Wildcard: `user*`

#### View Document

Click a document to see:
- Full JSON source
- Document ID
- Index name

#### Delete Document

1. Click "Delete" on a document
2. Confirm deletion
3. Document is removed

### Query Console

Execute raw Elasticsearch queries:

1. Select HTTP method (GET, POST, PUT, DELETE)
2. Enter path (e.g., `/_cat/nodes?v`)
3. Enter JSON body (for POST/PUT)
4. Click "Execute"
5. View JSON response

**Example queries:**

```
# Cluster health
GET /_cluster/health

# Cat APIs
GET /_cat/nodes?v
GET /_cat/indices?v
GET /_cat/shards?v

# Search
POST /my-index/_search
{
  "query": {
    "match": {
      "message": "error"
    }
  }
}

# Aggregation
POST /logs/_search
{
  "size": 0,
  "aggs": {
    "status_codes": {
      "terms": { "field": "status" }
    }
  }
}

# Create index with mapping
PUT /new-index
{
  "mappings": {
    "properties": {
      "timestamp": { "type": "date" },
      "message": { "type": "text" }
    }
  }
}
```

## API Reference

```bash
# Connect and test
curl -X POST http://localhost:8080/api/v1/elasticsearch/connect \
  -H "Content-Type: application/json" \
  -d '{
    "url":"http://localhost:9200",
    "username":"elastic",
    "password":"changeme"
  }'

# Get cluster health
curl -X POST http://localhost:8080/api/v1/elasticsearch/health \
  -H "Content-Type: application/json" \
  -d '{"url":"http://localhost:9200"}'

# List indices
curl -X POST http://localhost:8080/api/v1/elasticsearch/indices \
  -H "Content-Type: application/json" \
  -d '{"url":"http://localhost:9200"}'

# Search documents
curl -X POST http://localhost:8080/api/v1/elasticsearch/search \
  -H "Content-Type: application/json" \
  -d '{
    "url":"http://localhost:9200",
    "index":"my-index",
    "query":"status:error",
    "size":100
  }'

# Execute raw query
curl -X POST http://localhost:8080/api/v1/elasticsearch/query \
  -H "Content-Type: application/json" \
  -d '{
    "url":"http://localhost:9200",
    "method":"GET",
    "path":"/_cat/nodes?v"
  }'
```

## Use Cases

### Checking Cluster Health

1. Connect to cluster
2. View Cluster tab
3. Check status color and shard allocation

### Investigating Logs

1. Connect to cluster
2. Go to Documents tab
3. Select logs index
4. Search for error patterns
5. View individual log documents

### Index Management

1. Connect to cluster
2. Go to Indices tab
3. Monitor index sizes
4. Delete old indices to free space
5. Create new indices with proper mappings

### Debugging Queries

1. Connect to cluster
2. Go to Query Console
3. Test queries interactively
4. Refine and optimize before using in code
