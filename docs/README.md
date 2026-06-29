# mcp-database

`mcp-database` is a Go-based MCP Server that lets MCP-compatible AI clients access configured database instances through stdio. The current implementation focuses on read-only queries, object metadata inspection, and SQL execution plan analysis.

## Features

- Load one or more database instances from a YAML configuration file.
- Execute `SELECT` or `WITH` queries through the `execute_select` tool.
- Inspect table or view columns through the `describe_object` tool.
- Get SQL execution plans through the `get_execution_plan` tool.
- Supports Oracle, MySQL, and PostgreSQL databases.
- Return at most 50 rows by default to reduce context overflow risk.

## Requirements

- Go installed.
- Network access from the MCP Server machine to the target database.
- A database account with permissions for querying, object metadata inspection, and execution plan analysis.

## Configuration

Copy the example configuration from the project root:

```powershell
Copy-Item examples\example-config.yaml config.yaml
```

Then edit `config.yaml` with your real database connection details:

```yaml
instances:
  dev_oracle:
    type: "oracle"
    user: "YOUR_USERNAME"
    password: "YOUR_PASSWORD"
    host: "127.0.0.1"
    port: 1521
    serviceName: "YOUR_SERVICE_NAME"

  dev_mysql:
    type: "mysql"
    user: "YOUR_USERNAME"
    password: "YOUR_PASSWORD"
    host: "127.0.0.1"
    port: 3306
    database: "YOUR_DATABASE"
    charset: "utf8mb4"

  dev_pg:
    type: "pg"
    user: "YOUR_USERNAME"
    password: "YOUR_PASSWORD"
    host: "127.0.0.1"
    port: 5432
    database: "YOUR_DATABASE"
    sslMode: "disable"
```

Supported database types:

- Oracle: `oracle`
- MySQL: `mysql`
- PostgreSQL: `postgresql`, `postgres`, `pg`

`examples\example-config.yaml` should contain placeholders only and is safe to commit. `config.yaml` contains real database usernames, passwords, hosts, and service names. It is ignored by `.gitignore` and must never be committed.

## Build

```powershell
go mod tidy
go build -o mcp-database.exe .
```

## Run

By default, the server reads `config.yaml` from the current working directory:

```powershell
.\mcp-database.exe
```

You can also pass a specific configuration file:

```powershell
.\mcp-database.exe -config D:\Projects\mcp-database\config.yaml
```

This server uses stdio as its MCP transport, so MCP clients such as Claude Desktop or Cursor should launch the executable directly instead of calling an HTTP endpoint.

## MCP Tools

### execute_select

Executes a read-only SQL query. Only SQL beginning with `SELECT` or `WITH` is accepted.

Arguments:

- `instance`: database instance alias configured in `config.yaml`
- `sql`: SQL query text

### describe_object

Returns column metadata for a table or view in a specified schema.

Arguments:

- `instance`: database instance alias configured in `config.yaml`
- `schema_name`: schema name
- `object_name`: table or view name

### get_execution_plan

Runs the database-specific execution plan command for a SQL query and returns the formatted execution plan text.

Arguments:

- `instance`: database instance alias configured in `config.yaml`
- `sql`: SQL query text to analyze

## Security Notes

- Do not commit `config.yaml`.
- Keep only sanitized placeholder values in `examples\example-config.yaml`.
- Use a read-only or least-privilege database account for the MCP Server.
- Although the server rejects SQL that does not begin with `SELECT` or `WITH`, database permissions should still block writes, DDL, and administrative operations at the source.
- Be careful when exposing production instance aliases, connection details, and query results to AI clients.
