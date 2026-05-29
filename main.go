package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"mcp-database/db"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"
)

// loadConfig 读取并解析 YAML 配置文件
func loadConfig(filename string) (*db.AppConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config db.AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	return &config, nil
}

func main() {
	configPath := flag.String("config", "config.yaml", "YAML configuration file path")
	flag.Parse()

	// 1. 初始化多实例管理器
	manager := db.NewInstanceManager()

	// 2. 动态加载配置
	appConfig, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("配置初始化致命错误: %v", err)
	}

	// 3. 遍历并注册所有数据库实例
	registeredCount := 0
	for alias, cfg := range appConfig.Instances {
		err := manager.Register(alias, cfg)
		if err != nil {
			// 如果某个实例连不上，打印警告但继续加载其他实例
			log.Printf("警告: 无法注册数据库实例 '%s': %v", alias, err)
			continue
		}
		registeredCount++
		log.Printf("成功注册数据库实例: %s (%s)", alias, cfg.Type)
	}

	if registeredCount == 0 {
		log.Fatalf("没有成功注册任何数据库实例，服务终止。")
	}
	defer func() {
		if err := manager.Close(); err != nil {
			log.Printf("关闭数据库连接时发生错误: %v", err)
		}
	}()

	// 4. 创建并注册 MCP Server 工具 (这部分代码与之前完全一致)
	s := server.NewMCPServer("Go-Database-MCP", "1.0.0")

	// 注册执行查询工具 (智能截断设定为最多 50 行)
	selectTool := mcp.NewTool("execute_select",
		mcp.WithDescription("执行 SELECT 或 WITH 查询。"),
		mcp.WithString("instance", mcp.Required(), mcp.Description("数据库实例别名, 如: prod_oracle")),
		mcp.WithString("sql", mcp.Required(), mcp.Description("SQL 语句")),
	)
	s.AddTool(selectTool, utilExecuteSelect(manager))

	// 注册查看表结构工具
	descTool := mcp.NewTool("describe_object",
		mcp.WithDescription("获取指定 Schema 下数据库表或视图的结构信息。写复杂SQL前必须调用此工具。"),
		mcp.WithString("instance", mcp.Required(), mcp.Description("数据库实例别名")),
		mcp.WithString("schema_name", mcp.Required(), mcp.Description("Schema 名称")),
		mcp.WithString("object_name", mcp.Required(), mcp.Description("表名或视图名")),
	)
	s.AddTool(descTool, utilDescribe(manager))

	// 注册执行计划工具
	planTool := mcp.NewTool("get_execution_plan",
		mcp.WithDescription("获取查询的执行计划 (Explain Plan)，用于慢SQL调优和索引检查。"),
		mcp.WithString("instance", mcp.Required(), mcp.Description("数据库实例别名")),
		mcp.WithString("sql", mcp.Required(), mcp.Description("需要分析的SQL")),
	)
	s.AddTool(planTool, utilExplain(manager))

	// 启动标准输入输出服务 (供 Claude Desktop / Cursor 调用)
	log.Println("Starting MCP Go Server on stdio...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ----------------- 工具执行逻辑 -----------------

func utilExecuteSelect(manager *db.InstanceManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instance, err := request.RequireString("instance")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		sqlQuery, err := request.RequireString("sql")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		driver, err := manager.Get(instance)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// 触发查询，设置阈值为 50 行
		_, rows, truncated, err := driver.ExecuteSelect(ctx, sqlQuery, 50)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("SQL Error: %v", err)), nil
		}

		resp := make(map[string]any)
		resp["data"] = rows
		if truncated {
			resp["meta"] = "Result was truncated to 50 rows to prevent context overflow. Please use pagination or stronger WHERE conditions if you need specific records."
		}

		jsonBytes, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

func utilDescribe(manager *db.InstanceManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instance, err := request.RequireString("instance")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		schemaName, err := request.RequireString("schema_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		objName, err := request.RequireString("object_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		driver, err := manager.Get(instance)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		info, err := driver.DescribeObject(ctx, schemaName, objName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("DB Error: %v", err)), nil
		}

		jsonBytes, _ := json.MarshalIndent(info, "", "  ")
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

func utilExplain(manager *db.InstanceManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instance, err := request.RequireString("instance")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		sqlQuery, err := request.RequireString("sql")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		driver, err := manager.Get(instance)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		planText, err := driver.GetExecutionPlan(ctx, sqlQuery)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Explain Error: %v", err)), nil
		}

		return mcp.NewToolResultText(planText), nil
	}
}
