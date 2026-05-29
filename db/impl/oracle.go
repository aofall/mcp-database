package impl

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/sijms/go-ora/v2" // 引入纯 Go Oracle 驱动
)

type OracleDriver struct {
	db *sql.DB
}

type OracleConfig struct {
	User        string
	Password    string
	Host        string
	Port        int
	ServiceName string
}

func closeRows(rows *sql.Rows, errp *error) {
	if rows == nil {
		return
	}
	if closeErr := rows.Close(); closeErr != nil && *errp == nil {
		*errp = closeErr
	}
}

func NewOracleDriver(cfg OracleConfig) (*OracleDriver, error) {
	// go-ora 的 DSN 格式
	dsn := fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.ServiceName)

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}

	// 4. 连接池管理配置
	db.SetMaxOpenConns(25)                 // 最大打开连接数
	db.SetMaxIdleConns(5)                  // 最大空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) // 连接最大存活时间

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &OracleDriver{db: db}, nil
}

// ExecuteSelect 1. SELECT且支持CTE查询 + 智能截断与分页
func (o *OracleDriver) ExecuteSelect(ctx context.Context, query string, maxRows int) (columns []string, results []map[string]any, truncated bool, err error) {
	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)
	if !strings.HasPrefix(upperQuery, "SELECT") && !strings.HasPrefix(upperQuery, "WITH") {
		return nil, nil, false, fmt.Errorf("security violation: only SELECT or WITH statements are allowed")
	}

	rows, err := o.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, false, err
	}
	defer closeRows(rows, &err)

	columns, err = rows.Columns()
	if err != nil {
		return nil, nil, false, err
	}

	count := 0

	for rows.Next() {
		if count >= maxRows {
			truncated = true
			break // 智能截断：达到阈值停止读取
		}

		// Go 语言动态读取列数据的标准做法
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, false, err
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			// 处理 Go sql 中的 byte 数组转字符串
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, nil, false, err
	}

	return columns, results, truncated, nil
}

// DescribeObject 2. DESCRIBE (支持普通表和视图)
func (o *OracleDriver) DescribeObject(ctx context.Context, schemaName, objectName string) (info any, err error) {
	upperSchema := strings.ToUpper(schemaName)
	upperName := strings.ToUpper(objectName)
	query := `
		SELECT owner, column_name, data_type, data_length, nullable
		FROM all_tab_columns 
		WHERE owner = :1
		  AND table_name = :2
		ORDER BY column_id`

	rows, err := o.db.QueryContext(ctx, query, upperSchema, upperName)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, &err)

	var cols []map[string]any
	for rows.Next() {
		var owner, colName, dataType, nullable string
		var dataLen int
		if err := rows.Scan(&owner, &colName, &dataType, &dataLen, &nullable); err != nil {
			return nil, err
		}
		cols = append(cols, map[string]any{
			"OWNER":       owner,
			"COLUMN_NAME": colName,
			"DATA_TYPE":   dataType,
			"DATA_LENGTH": dataLen,
			"NULLABLE":    nullable,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cols, nil
}

// GetExecutionPlan 3. EXPLAIN PLAN 获取执行计划
func (o *OracleDriver) GetExecutionPlan(ctx context.Context, query string) (plan string, err error) {
	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)
	if !strings.HasPrefix(upperQuery, "SELECT") && !strings.HasPrefix(upperQuery, "WITH") {
		return "", fmt.Errorf("explain plan only supports SELECT or WITH statements")
	}

	statementId := "mcp_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:20]

	// 执行 Explain
	explainSql := fmt.Sprintf("EXPLAIN PLAN SET STATEMENT_ID = '%s' FOR %s", statementId, query)
	_, err = o.db.ExecContext(ctx, explainSql)
	if err != nil {
		return "", err
	}

	// 确保最终清理执行计划
	defer func() {
		_, _ = o.db.Exec("DELETE FROM PLAN_TABLE WHERE STATEMENT_ID = :1", statementId)
	}()

	// 获取格式化结果
	displaySql := `SELECT PLAN_TABLE_OUTPUT FROM TABLE(DBMS_XPLAN.DISPLAY('PLAN_TABLE', :1, 'TYPICAL'))`
	rows, err := o.db.QueryContext(ctx, displaySql, statementId)
	if err != nil {
		return "", err
	}
	defer closeRows(rows, &err)

	var planBuilder strings.Builder
	for rows.Next() {
		var line sql.NullString
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		if line.Valid {
			planBuilder.WriteString(line.String + "\n")
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	return planBuilder.String(), nil
}

func (o *OracleDriver) Close() error {
	return o.db.Close()
}
