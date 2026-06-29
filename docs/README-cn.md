# mcp-database

`mcp-database` 是一个使用 Go 编写的 MCP Server，用于让支持 MCP 的 AI 客户端通过标准输入输出方式访问已配置的数据库实例。当前实现支持只读查询、表结构查看和 SQL 执行计划分析。

## 功能

- 从 YAML 配置文件加载一个或多个数据库实例。
- 通过 `execute_select` 工具执行 `SELECT` 或 `WITH` 查询。
- 通过 `describe_object` 工具查看表或视图的字段结构。
- 通过 `get_execution_plan` 工具获取 SQL 执行计划。
- 支持 Oracle、MySQL 和 PostgreSQL 数据库。
- 查询结果默认最多返回 50 行，避免上下文过大。

## 环境要求

- 已安装 Go。
- MCP Server 所在机器能够访问目标数据库。
- 数据库账号具备所需的查询、表结构查看和执行计划权限。

## 配置文件

在项目根目录下复制示例配置：

```powershell
Copy-Item examples\example-config.yaml config.yaml
```

然后编辑 `config.yaml`，填入真实数据库连接信息：

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

数据库类型支持：

- Oracle：`oracle`
- MySQL：`mysql`
- PostgreSQL：`postgresql`、`postgres`、`pg`

`examples\example-config.yaml` 只应保存占位示例，可以提交到 Git。`config.yaml` 会保存真实数据库账号、密码、主机地址和服务名，已经被 `.gitignore` 忽略，绝对不要提交到 Git。

## 构建

```powershell
go mod tidy
go -o mcp-database.exe .
```

## 运行

默认读取当前工作目录下的 `config.yaml`：

```powershell
.\mcp-database.exe
```

也可以通过命令行参数指定配置文件：

```powershell
.\mcp-database.exe -config D:\Projects\mcp-database\config.yaml
```

本服务使用 stdio 作为 MCP 传输方式，因此应由 Claude Desktop、Cursor 等 MCP 客户端直接启动可执行文件，而不是通过 HTTP 地址访问。

## MCP 工具

### execute_select

执行只读 SQL 查询。当前仅允许以 `SELECT` 或 `WITH` 开头的 SQL。

参数：

- `instance`：`config.yaml` 中配置的数据库实例别名
- `sql`：需要执行的 SQL 文本

### describe_object

返回指定 Schema 下表或视图的字段结构信息。

参数：

- `instance`：`config.yaml` 中配置的数据库实例别名
- `schema_name`：Schema 名称
- `object_name`：表名或视图名

### get_execution_plan

对 SQL 执行数据库对应的执行计划命令，并返回格式化后的执行计划文本。

参数：

- `instance`：`config.yaml` 中配置的数据库实例别名
- `sql`：需要分析执行计划的 SQL 文本

## 安全建议

- 不要提交 `config.yaml`。
- `examples\example-config.yaml` 中只能保留脱敏后的占位信息。
- 给 MCP Server 使用的数据库账号应尽量采用只读或最小权限。
- 虽然服务会拒绝非 `SELECT` / `WITH` SQL，但数据库权限仍应从源头限制写入、DDL 和管理操作。
- 谨慎暴露生产库实例别名、连接信息和查询结果给 AI 客户端。
