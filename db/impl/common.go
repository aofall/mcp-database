package impl

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
}

func closeRows(rows *sql.Rows, errp *error) {
	if rows == nil {
		return
	}
	if closeErr := rows.Close(); closeErr != nil && *errp == nil {
		*errp = closeErr
	}
}

func validateReadOnlyQuery(query string, action string) (string, error) {
	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)
	if !strings.HasPrefix(upperQuery, "SELECT") && !strings.HasPrefix(upperQuery, "WITH") {
		return "", fmt.Errorf("%s only supports SELECT or WITH statements", action)
	}
	return query, nil
}

func executeSelect(ctx context.Context, db *sql.DB, query string, maxRows int) (columns []string, results []map[string]any, truncated bool, err error) {
	query, err = validateReadOnlyQuery(query, "security violation")
	if err != nil {
		return nil, nil, false, err
	}

	rows, err := db.QueryContext(ctx, query)
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
			break
		}

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
			if b, ok := values[i].([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = values[i]
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
