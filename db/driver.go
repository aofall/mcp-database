package db

import (
	"context"
	"fmt"
	"mcp-database/db/impl"
)

// DBDriver 定义了数据库操作的策略接口
type DBDriver interface {
	// ExecuteSelect 执行查询，返回列名、数据行以及是否被智能截断
	ExecuteSelect(ctx context.Context, sql string, maxRows int) ([]string, []map[string]any, bool, error)
	// DescribeObject 获取表或视图结构
	DescribeObject(ctx context.Context, schemaName, objectName string) (any, error)
	// GetExecutionPlan 获取执行计划
	GetExecutionPlan(ctx context.Context, sql string) (string, error)
	// Close 关闭连接池
	Close() error
}

// AppConfig 用于映射整个配置文件的外层结构
type AppConfig struct {
	Instances map[string]Config `yaml:"instances" json:"instances"`
}

// Config 数据库配置项
type Config struct {
	Type        string `yaml:"type" json:"type"`
	User        string `yaml:"user" json:"user"`
	Password    string `yaml:"password" json:"password"`
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	ServiceName string `yaml:"serviceName" json:"serviceName"`
}

// NewDriver 工厂方法：根据类型实例化具体的驱动
func NewDriver(cfg Config) (DBDriver, error) {
	switch cfg.Type {
	case "oracle":
		return impl.NewOracleDriver(impl.OracleConfig{
			User:        cfg.User,
			Password:    cfg.Password,
			Host:        cfg.Host,
			Port:        cfg.Port,
			ServiceName: cfg.ServiceName,
		})
	// case "dameng":
	// 	return NewDamengDriver(cfg) // 暂缓开发，预留扩展点
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}
