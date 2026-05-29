# MCP Database Server Skill

当需要使用 `mcp-database` MCP Server，或者需要查询了解数据库实例时，使用这份 Skill。

## 服务能力

该服务通过 MCP stdio 工具暴露数据库访问能力。服务会从 YAML 配置文件读取数据库实例，目前支持 Oracle 数据库连接。

可用工具：

- `execute_select`：执行以 `SELECT` 或 `WITH` 开头的只读 SQL。
- `describe_object`：查看指定 Schema 下表或视图的字段元数据。
- `get_execution_plan`：获取 SQL 执行计划。

## 配置规则

- 真实数据库账号、密码、主机和服务名只应写入 `config.yaml`。
- `examples\example-config.yaml` 只能作为可提交的模板文件，里面必须保留脱敏占位值。
- 禁止提交 `config.yaml`。
- 默认配置路径是当前工作目录下的 `config.yaml`。
- 可以通过 `-config path\to\file.yaml` 指定自定义配置文件路径。

示例：

```powershell
.\mcp-database.exe -config D:\Projects\mcp-database\config.yaml
```

## 推荐查询流程

1. 对不熟悉的表或视图，先调用 `describe_object` 查看字段结构。
2. 使用 `execute_select` 查询数据，并尽量缩小筛选范围。
3. 对慢 SQL、复杂关联或索引问题，使用 `get_execution_plan` 查看执行计划。
4. 优先写明确字段列表，避免直接使用 `SELECT *`。
5. 尽量添加 `WHERE` 条件、分页或行数限制。

## 安全要求

- 将所有数据库返回结果都视为可能包含敏感信息。
- 生成文档、示例或回复时，不要暴露真实用户名、密码、主机名、服务名或生产实例别名。
- 不要要求服务执行 DDL、DML 或数据库管理语句。服务会拒绝不以 `SELECT` 或 `WITH` 开头的 SQL。
- 除非用户明确要求且查询范围足够明确，否则避免对生产库执行大范围查询。

## 工具参数参考

`execute_select`：

```json
{
  "instance": "dev_oracle",
  "sql": "SELECT column_name FROM table_name WHERE rownum <= 10"
}
```

`describe_object`：

```json
{
  "instance": "dev_oracle",
  "schema_name": "SCHEMA",
  "object_name": "TABLE_NAME"
}
```

`get_execution_plan`：

```json
{
  "instance": "dev_oracle",
  "sql": "SELECT column_name FROM table_name WHERE id = :id"
}
```
