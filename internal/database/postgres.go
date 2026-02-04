// Copyright 2024-2026 GAGOS Project
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PostgresConfig represents PostgreSQL connection configuration
type PostgresConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode"`
}

func (c *PostgresConfig) ConnectionString() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, sslMode)
}

// PostgresConnectionResult represents connection test result
type PostgresConnectionResult struct {
	Success      bool    `json:"success"`
	Version      string  `json:"version,omitempty"`
	ResponseTime float64 `json:"response_time_ms,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// PostgresInfo represents database information
type PostgresInfo struct {
	Version        string           `json:"version"`
	DatabaseSize   string           `json:"database_size"`
	TableCount     int              `json:"table_count"`
	ActiveConns    int              `json:"active_connections"`
	MaxConns       int              `json:"max_connections"`
	Uptime         string           `json:"uptime"`
	Tables         []PostgresTable  `json:"tables,omitempty"`
	Error          string           `json:"error,omitempty"`
}

// PostgresTable represents table information
type PostgresTable struct {
	Schema    string `json:"schema"`
	Name      string `json:"name"`
	RowCount  int64  `json:"row_count"`
	Size      string `json:"size"`
	IndexSize string `json:"index_size"`
}

// PostgresQueryResult represents query execution result
type PostgresQueryResult struct {
	Columns      []string        `json:"columns,omitempty"`
	Rows         [][]interface{} `json:"rows,omitempty"`
	RowsAffected int64           `json:"rows_affected"`
	Duration     float64         `json:"duration_ms"`
	Error        string          `json:"error,omitempty"`
}

// PostgresDumpResult represents dump operation result
type PostgresDumpResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output,omitempty"`
	Size     int64  `json:"size_bytes,omitempty"`
	Duration float64 `json:"duration_ms"`
	Error    string `json:"error,omitempty"`
}

// TestPostgresConnection tests PostgreSQL connection
func TestPostgresConnection(ctx context.Context, config PostgresConfig) PostgresConnectionResult {
	start := time.Now()

	db, err := sql.Open("postgres", config.ConnectionString())
	if err != nil {
		return PostgresConnectionResult{
			Success: false,
			Error:   "Failed to open connection: " + err.Error(),
		}
	}
	defer db.Close()

	db.SetConnMaxLifetime(10 * time.Second)
	db.SetMaxOpenConns(1)

	var version string
	err = db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return PostgresConnectionResult{
			Success: false,
			Error:   "Failed to query: " + err.Error(),
		}
	}

	return PostgresConnectionResult{
		Success:      true,
		Version:      version,
		ResponseTime: float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

// GetPostgresInfo retrieves database information
func GetPostgresInfo(ctx context.Context, config PostgresConfig) PostgresInfo {
	db, err := sql.Open("postgres", config.ConnectionString())
	if err != nil {
		return PostgresInfo{Error: "Failed to connect: " + err.Error()}
	}
	defer db.Close()

	info := PostgresInfo{}

	// Get version
	db.QueryRowContext(ctx, "SELECT version()").Scan(&info.Version)

	// Get database size
	db.QueryRowContext(ctx, "SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&info.DatabaseSize)

	// Get table count
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')").Scan(&info.TableCount)

	// Get connection info
	db.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'").Scan(&info.ActiveConns)
	db.QueryRowContext(ctx, "SELECT setting::int FROM pg_settings WHERE name = 'max_connections'").Scan(&info.MaxConns)

	// Get uptime
	db.QueryRowContext(ctx, "SELECT current_timestamp - pg_postmaster_start_time()").Scan(&info.Uptime)

	// Get tables info
	rows, err := db.QueryContext(ctx, `
		SELECT
			schemaname,
			tablename,
			n_live_tup,
			pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)),
			pg_size_pretty(pg_indexes_size(schemaname || '.' || tablename))
		FROM pg_stat_user_tables
		ORDER BY pg_total_relation_size(schemaname || '.' || tablename) DESC
		LIMIT 50
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t PostgresTable
			rows.Scan(&t.Schema, &t.Name, &t.RowCount, &t.Size, &t.IndexSize)
			info.Tables = append(info.Tables, t)
		}
	}

	return info
}

// ExecutePostgresQuery executes a SQL query
func ExecutePostgresQuery(ctx context.Context, config PostgresConfig, query string, readonly bool) PostgresQueryResult {
	start := time.Now()

	db, err := sql.Open("postgres", config.ConnectionString())
	if err != nil {
		return PostgresQueryResult{Error: "Failed to connect: " + err.Error()}
	}
	defer db.Close()

	query = strings.TrimSpace(query)
	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(query), "SHOW") ||
		strings.HasPrefix(strings.ToUpper(query), "EXPLAIN")

	if readonly && !isSelect {
		return PostgresQueryResult{Error: "Only SELECT queries allowed in read-only mode"}
	}

	if isSelect {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return PostgresQueryResult{
				Error:    err.Error(),
				Duration: float64(time.Since(start).Microseconds()) / 1000.0,
			}
		}
		defer rows.Close()

		cols, _ := rows.Columns()
		result := PostgresQueryResult{
			Columns:  cols,
			Rows:     make([][]interface{}, 0),
			Duration: 0,
		}

		for rows.Next() {
			values := make([]interface{}, len(cols))
			valuePtrs := make([]interface{}, len(cols))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			rows.Scan(valuePtrs...)

			row := make([]interface{}, len(cols))
			for i, v := range values {
				if b, ok := v.([]byte); ok {
					row[i] = string(b)
				} else {
					row[i] = v
				}
			}
			result.Rows = append(result.Rows, row)

			if len(result.Rows) >= 1000 {
				break // Limit rows
			}
		}

		result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
		return result
	}

	// Execute non-SELECT query
	res, err := db.ExecContext(ctx, query)
	if err != nil {
		return PostgresQueryResult{
			Error:    err.Error(),
			Duration: float64(time.Since(start).Microseconds()) / 1000.0,
		}
	}

	affected, _ := res.RowsAffected()
	return PostgresQueryResult{
		RowsAffected: affected,
		Duration:     float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

// DumpPostgres creates a database dump
func DumpPostgres(ctx context.Context, config PostgresConfig, schemaOnly bool, dataOnly bool, tables []string) PostgresDumpResult {
	start := time.Now()

	args := []string{
		"-h", config.Host,
		"-p", fmt.Sprintf("%d", config.Port),
		"-U", config.User,
		"-d", config.Database,
		"--no-password",
	}

	if schemaOnly {
		args = append(args, "--schema-only")
	}
	if dataOnly {
		args = append(args, "--data-only")
	}
	for _, t := range tables {
		if t != "" {
			args = append(args, "-t", t)
		}
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.Password))

	output, err := cmd.Output()
	if err != nil {
		return PostgresDumpResult{
			Success:  false,
			Error:    "pg_dump failed: " + err.Error(),
			Duration: float64(time.Since(start).Microseconds()) / 1000.0,
		}
	}

	return PostgresDumpResult{
		Success:  true,
		Output:   string(output),
		Size:     int64(len(output)),
		Duration: float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

// GetPostgresDatabases lists all databases
func GetPostgresDatabases(ctx context.Context, config PostgresConfig) ([]string, error) {
	// Connect to postgres database to list all databases
	connConfig := config
	connConfig.Database = "postgres"

	db, err := sql.Open("postgres", connConfig.ConnectionString())
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		databases = append(databases, name)
	}
	return databases, nil
}
