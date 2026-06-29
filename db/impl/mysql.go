package impl

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

type MySQLDriver struct {
	db *sql.DB
}

type MySQLConfig struct {
	User     string
	Password string
	Host     string
	Port     int
	Database string
	Charset  string
}

func NewMySQLDriver(cfg MySQLConfig) (*MySQLDriver, error) {
	charset := cfg.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	dsn := mysql.NewConfig()
	dsn.User = cfg.User
	dsn.Passwd = cfg.Password
	dsn.Net = "tcp"
	dsn.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dsn.DBName = cfg.Database
	dsn.ParseTime = true
	dsn.Params = map[string]string{
		"charset": charset,
	}

	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		return nil, err
	}
	configurePool(db)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &MySQLDriver{db: db}, nil
}

func (m *MySQLDriver) ExecuteSelect(ctx context.Context, query string, maxRows int) ([]string, []map[string]any, bool, error) {
	return executeSelect(ctx, m.db, query, maxRows)
}

func (m *MySQLDriver) DescribeObject(ctx context.Context, schemaName, objectName string) (info any, err error) {
	query := `
		SELECT table_schema, column_name, data_type, character_maximum_length,
		       numeric_precision, numeric_scale, is_nullable, column_default,
		       column_key, extra
		FROM information_schema.columns
		WHERE table_schema = ?
		  AND table_name = ?
		ORDER BY ordinal_position`

	rows, err := m.db.QueryContext(ctx, query, schemaName, objectName)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, &err)

	var cols []map[string]any
	for rows.Next() {
		var tableSchema, colName, dataType, nullable, columnKey, extra string
		var charLen, numericPrecision, numericScale sql.NullInt64
		var columnDefault sql.NullString
		if err := rows.Scan(&tableSchema, &colName, &dataType, &charLen, &numericPrecision, &numericScale, &nullable, &columnDefault, &columnKey, &extra); err != nil {
			return nil, err
		}

		cols = append(cols, map[string]any{
			"TABLE_SCHEMA":             tableSchema,
			"COLUMN_NAME":              colName,
			"DATA_TYPE":                dataType,
			"CHARACTER_MAXIMUM_LENGTH": nullableInt(charLen),
			"NUMERIC_PRECISION":        nullableInt(numericPrecision),
			"NUMERIC_SCALE":            nullableInt(numericScale),
			"IS_NULLABLE":              nullable,
			"COLUMN_DEFAULT":           nullableString(columnDefault),
			"COLUMN_KEY":               columnKey,
			"EXTRA":                    extra,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cols, nil
}

func (m *MySQLDriver) GetExecutionPlan(ctx context.Context, query string) (plan string, err error) {
	query, err = validateReadOnlyQuery(query, "explain plan")
	if err != nil {
		return "", err
	}

	rows, err := m.db.QueryContext(ctx, "EXPLAIN "+query)
	if err != nil {
		return "", err
	}
	defer closeRows(rows, &err)

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(strings.Join(columns, "\t"))
	builder.WriteString("\n")

	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}

		for i, value := range values {
			if i > 0 {
				builder.WriteString("\t")
			}
			builder.WriteString(formatExplainValue(value))
		}
		builder.WriteString("\n")
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	return builder.String(), nil
}

func (m *MySQLDriver) Close() error {
	return m.db.Close()
}

func nullableInt(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

func nullableString(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}

func formatExplainValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return fmt.Sprint(v)
	}
}
