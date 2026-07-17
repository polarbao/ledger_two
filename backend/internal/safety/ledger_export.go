package safety

import (
	"context"
	"database/sql"
	"fmt"
)

func queryLedgerExportRows(
	ctx context.Context,
	database *sql.DB,
	query string,
	args ...any,
) ([]map[string]any, error) {
	rows, err := database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for index := range values {
			destinations[index] = &values[index]
		}
		if err := rows.Scan(destinations...); err != nil {
			return nil, err
		}

		record := make(map[string]any, len(columns))
		for index, column := range columns {
			value := values[index]
			if bytes, ok := value.([]byte); ok {
				value = string(bytes)
			}
			record[column] = value
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func queryLedgerExportSection(
	ctx context.Context,
	database *sql.DB,
	section string,
	query string,
	args ...any,
) ([]map[string]any, error) {
	rows, err := queryLedgerExportRows(ctx, database, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query export section %s: %w", section, err)
	}
	return rows, nil
}
