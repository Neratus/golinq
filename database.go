package golinq

import (
	"context"
	"database/sql"

	"github.com/Neratus/golinq/internal/dialect"
)

type DB struct {
	conn    *sql.DB
	dialect *dialect.SQLDialect
}

func (db *DB) DB() *sql.DB {
	return db.conn
}

func (db *DB) Dialect() *dialect.SQLDialect {
	return db.dialect
}

func (db *DB) Raw(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.conn.QueryContext(ctx, query, args)
}

func (db *DB) RawExec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.conn.ExecContext(ctx, query, args)
}
