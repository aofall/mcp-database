package impl

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgreSQLDriver struct {
	db *sql.DB
}

type PostgreSQLConfig struct {
	User     string
	Password string
	Host     string
	Port     int
	Database string
	SSLMode  string
}

func NewPostgreSQLDriver(cfg PostgreSQLConfig) (*PostgreSQLDriver, error) {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   "/" + cfg.Database,
	}
	values := dsn.Query()
	values.Set("sslmode", sslMode)
	dsn.RawQuery = values.Encode()

	db, err := sql.Open("pgx", dsn.String())
	if err != nil {
		return nil, err
	}
	configurePool(db)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &PostgreSQLDriver{db: db}, nil
}

func (p *PostgreSQLDriver) ExecuteSelect(ctx context.Context, query string, maxRows int) ([]string, []map[string]any, bool, error) {
	return executeSelect(ctx, p.db, query, maxRows)
}

func (p *PostgreSQLDriver) DescribeObject(ctx context.Context, schemaName, objectName string) (info any, err error) {
	query := `
		SELECT table_schema, column_name, data_type, udt_name,
		       character_maximum_length, numeric_precision, numeric_scale,
		       is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name = $2
		ORDER BY ordinal_position`

	rows, err := p.db.QueryContext(ctx, query, schemaName, objectName)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, &err)

	var cols []map[string]any
	for rows.Next() {
		var tableSchema, colName, dataType, udtName, nullable string
		var charLen, numericPrecision, numericScale sql.NullInt64
		var columnDefault sql.NullString
		if err := rows.Scan(&tableSchema, &colName, &dataType, &udtName, &charLen, &numericPrecision, &numericScale, &nullable, &columnDefault); err != nil {
			return nil, err
		}

		cols = append(cols, map[string]any{
			"TABLE_SCHEMA":             tableSchema,
			"COLUMN_NAME":              colName,
			"DATA_TYPE":                dataType,
			"UDT_NAME":                 udtName,
			"CHARACTER_MAXIMUM_LENGTH": nullableInt(charLen),
			"NUMERIC_PRECISION":        nullableInt(numericPrecision),
			"NUMERIC_SCALE":            nullableInt(numericScale),
			"IS_NULLABLE":              nullable,
			"COLUMN_DEFAULT":           nullableString(columnDefault),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cols, nil
}

func (p *PostgreSQLDriver) GetExecutionPlan(ctx context.Context, query string) (plan string, err error) {
	query, err = validateReadOnlyQuery(query, "explain plan")
	if err != nil {
		return "", err
	}

	rows, err := p.db.QueryContext(ctx, "EXPLAIN "+query)
	if err != nil {
		return "", err
	}
	defer closeRows(rows, &err)

	var builder strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	return builder.String(), nil
}

func (p *PostgreSQLDriver) Close() error {
	return p.db.Close()
}
