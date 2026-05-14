package golinq

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/Neratus/golinq/internal/dialect"
	postgres_dialect "github.com/Neratus/golinq/internal/dialect/postgres"
)

func extractHostPort(hostport string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
		portStr = ""
	}
	port := 5432
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return "", 0, fmt.Errorf("invalid port: %s", portStr)
		}
		port = p
	}
	return host, port, nil
}

func Connect(dataSourceName string) (*DB, error) {
	u, err := url.Parse(dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}

	var d *dialect.SQLDialect
	var driverName string

	host, port, err := extractHostPort(u.Host)
	if err != nil {
		return nil, fmt.Errorf("cannot parse host:port: %w", err)
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database server: %w", err)
	}
	conn.Close()

	switch u.Scheme {
	case "postgres", "postgresql":
		driverName = "pgx"
		d = postgres_dialect.NewPostgresDialect()
	default:
		return nil, fmt.Errorf("unsupported database scheme: %s", u.Scheme)
	}

	sqlDB, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	var one int
	err = sqlDB.QueryRow("SELECT 1").Scan(&one)
	if err != nil || one != 1 {
		sqlDB.Close()
		return nil, fmt.Errorf("basic query failed: %w", err)
	}

	return &DB{conn: sqlDB, dialect: d}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}
