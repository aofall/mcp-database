package db

import (
	"context"
	"fmt"
	"mcp-database/db/impl"
	"strings"

	"gopkg.in/yaml.v3"
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
	Instances map[string]yaml.Node `yaml:"instances" json:"instances"`
}

// CommonConfig 数据库通用配置项
type CommonConfig struct {
	Type     string `yaml:"type" json:"type"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
}

type OracleConfig struct {
	CommonConfig `yaml:",inline" json:",inline"`
	ServiceName  string `yaml:"serviceName" json:"serviceName"`
}

type MySQLConfig struct {
	CommonConfig `yaml:",inline" json:",inline"`
	Database     string `yaml:"database" json:"database"`
	Charset      string `yaml:"charset" json:"charset"`
}

type PostgreSQLConfig struct {
	CommonConfig `yaml:",inline" json:",inline"`
	Database     string `yaml:"database" json:"database"`
	SSLMode      string `yaml:"sslMode" json:"sslMode"`
}

// NewDriver 工厂方法：根据类型实例化具体的驱动
func NewDriver(node yaml.Node) (DBDriver, string, error) {
	var common CommonConfig
	if err := node.Decode(&common); err != nil {
		return nil, "", fmt.Errorf("decode common config: %w", err)
	}

	dbType := strings.ToLower(strings.TrimSpace(common.Type))
	switch dbType {
	case "oracle":
		var cfg OracleConfig
		if err := node.Decode(&cfg); err != nil {
			return nil, dbType, fmt.Errorf("decode oracle config: %w", err)
		}
		driver, err := impl.NewOracleDriver(impl.OracleConfig{
			User:        cfg.User,
			Password:    cfg.Password,
			Host:        cfg.Host,
			Port:        cfg.Port,
			ServiceName: cfg.ServiceName,
		})
		return driver, dbType, err
	case "mysql":
		var cfg MySQLConfig
		if err := node.Decode(&cfg); err != nil {
			return nil, dbType, fmt.Errorf("decode mysql config: %w", err)
		}
		driver, err := impl.NewMySQLDriver(impl.MySQLConfig{
			User:     cfg.User,
			Password: cfg.Password,
			Host:     cfg.Host,
			Port:     cfg.Port,
			Database: cfg.Database,
			Charset:  cfg.Charset,
		})
		return driver, dbType, err
	case "postgresql", "postgres", "pg":
		var cfg PostgreSQLConfig
		if err := node.Decode(&cfg); err != nil {
			return nil, dbType, fmt.Errorf("decode postgresql config: %w", err)
		}
		driver, err := impl.NewPostgreSQLDriver(impl.PostgreSQLConfig{
			User:     cfg.User,
			Password: cfg.Password,
			Host:     cfg.Host,
			Port:     cfg.Port,
			Database: cfg.Database,
			SSLMode:  cfg.SSLMode,
		})
		return driver, dbType, err
	// case "dameng":
	// 	return NewDamengDriver(cfg) // 暂缓开发，预留扩展点
	default:
		return nil, dbType, fmt.Errorf("unsupported database type: %s", common.Type)
	}
}
