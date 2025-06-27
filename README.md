# Manticore Search MCP Server

A Model Context Protocol (MCP) server that provides access to Manticore Search functionality through MCP-compatible clients like Claude Code, Cursor, and other AI development tools.

## Features

- Full-text search with advanced query options
- Table/index management operations  
- Document insertion and manipulation
- Cluster status monitoring
- Boolean queries with highlighting and fuzzy search
- Configurable result limits and pagination

## Installation

### Prerequisites

- Go 1.24 or later
- Manticore Search server running and accessible

### Build

```bash
git clone https://github.com/krajcik/manticore-mcp-server.git
cd manticore-mcp-server
go build -o manticore-mcp-server
```

## Configuration

Configure using environment variables:

```bash
export MANTICORE_URL="http://localhost:9308"
export MAX_RESULTS_PER_QUERY="100"
export REQUEST_TIMEOUT="30s"
export DEBUG="false"
```

Or command-line flags:

```bash
./manticore-mcp-server --manticore-url="http://localhost:9308" --max-results=100
```

## MCP Client Integration

Add to your MCP client configuration (e.g., `~/.claude.json` for Claude Code):

```json
{
  "mcpServers": {
    "manticore-search": {
      "command": "/path/to/manticore-mcp-server",
      "args": [],
      "env": {
        "MANTICORE_URL": "http://localhost:9308",
        "MAX_RESULTS_PER_QUERY": "50"
      }
    }
  }
}
```

## Available Tools

The MCP protocol automatically exposes these tools to clients:

### search
Full-text search in Manticore indexes.

**Key Parameters:**
- `table` (required): Table name
- `query`: Search text  
- `limit`: Max results
- `highlight`: Enable result highlighting
- `bool_query`: Complex boolean queries

### show_tables
List available tables/indexes.

### describe_table  
Get table schema information.

**Parameters:**
- `table` (required): Table name

### insert_document
Insert document into index.

**Parameters:**
- `table` (required): Table name
- `document` (required): Document data

### show_cluster_status
Display cluster health status.

## Response Format

All tools return structured JSON:

```json
{
  "success": true,
  "data": { /* results */ },
  "meta": {
    "total": 42,
    "count": 10,
    "operation": "search"
  }
}
```

## API Discovery

MCP clients automatically discover available tools and their schemas through the protocol. No manual configuration required.

## Troubleshooting

**Connection Issues:**
- Verify Manticore is running: `curl http://localhost:9308`
- Check server path in MCP configuration
- Use `--debug` for verbose logging

**Performance:**
- Adjust `MAX_RESULTS_PER_QUERY` for your needs
- Increase `REQUEST_TIMEOUT` for complex queries

## License

MIT License