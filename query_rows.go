package golinq

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

func scanStruct(rows *sql.Rows, dest interface{}) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	fieldAddrs := make([]interface{}, len(cols))
	for i, colName := range cols {
		found := false
		for j := 0; j < structVal.NumField(); j++ {
			field := structVal.Field(j)
			if !field.CanSet() {
				continue
			}
			fieldType := structType.Field(j)
			tag := fieldType.Tag.Get("golinq")
			if tag != "" {
				parts := strings.Split(tag, ",")
				for _, part := range parts {
					if part == "skip" {
						continue
					}
					if after, ok := strings.CutPrefix(part, "column="); ok {
						// Извлекаем имя колонки без префикса (после последней точки)
						simpleColName := colName
						if idx := strings.LastIndex(colName, "."); idx != -1 {
							simpleColName = colName[idx+1:]
						}
						// Сравниваем как с полным именем, так и с простым
						if after == colName || after == simpleColName {
							fieldAddrs[i] = field.Addr().Interface()
							found = true
							break
						}
					}
				}
				if found {
					break
				}
			} else {
				if convertToSnakeCase(fieldType.Name) == colName {
					fieldAddrs[i] = field.Addr().Interface()
					found = true
					break
				}
			}
		}
		if !found {
			fieldAddrs[i] = new(interface{})
		}
	}
	return rows.Scan(fieldAddrs...)
}

func QueryRows[T any](ctx context.Context, db *DB, query string, args ...any) ([]T, error) {
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var dest T
		if err := scanStruct(rows, &dest); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}
	return results, rows.Err()
}

func QueryRow[T any](ctx context.Context, db *DB, query string, args ...any) (T, error) {
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		var zero T
		return zero, err
	}
	defer rows.Close()

	if !rows.Next() {
		var zero T
		return zero, sql.ErrNoRows
	}
	var dest T
	if err := scanStruct(rows, &dest); err != nil {
		return dest, err
	}
	return dest, nil
}
