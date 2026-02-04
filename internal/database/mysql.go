package database

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLConfig represents MySQL/MariaDB connection configuration
type MySQLConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=10s",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

// MySQLConnectionResult represents connection test result
type MySQLConnectionResult struct {
	Success      bool    `json:"success"`
	Version      string  `json:"version,omitempty"`
	ServerType   string  `json:"server_type,omitempty"`
	ResponseTime float64 `json:"response_time_ms,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// MySQLInfo represents database information
type MySQLInfo struct {
	Version        string       `json:"version"`
	ServerType     string       `json:"server_type"`
	DatabaseSize   string       `json:"database_size"`
	TableCount     int          `json:"table_count"`
	Uptime         int64        `json:"uptime_seconds"`
	UptimeHuman    string       `json:"uptime_human"`
	Connections    int          `json:"connections"`
	MaxConnections int          `json:"max_connections"`
	QueriesPerSec  float64      `json:"queries_per_sec"`
	Tables         []MySQLTable `json:"tables,omitempty"`
	Error          string       `json:"error,omitempty"`
}

// MySQLTable represents table information
type MySQLTable struct {
	Name      string `json:"name"`
	Engine    string `json:"engine"`
	RowCount  int64  `json:"row_count"`
	DataSize  string `json:"data_size"`
	IndexSize string `json:"index_size"`
}

// MySQLQueryResult represents query execution result
type MySQLQueryResult struct {
	Columns      []string        `json:"columns,omitempty"`
	Rows         [][]interface{} `json:"rows,omitempty"`
	RowsAffected int64           `json:"rows_affected"`
	Duration     float64         `json:"duration_ms"`
	Error        string          `json:"error,omitempty"`
}

// MySQLDumpResult represents dump operation result
type MySQLDumpResult struct {
	Success  bool    `json:"success"`
	Output   string  `json:"output,omitempty"`
	Size     int64   `json:"size_bytes,omitempty"`
	Duration float64 `json:"duration_ms"`
	Error    string  `json:"error,omitempty"`
}

// TestMySQLConnection tests MySQL/MariaDB connection
func TestMySQLConnection(ctx context.Context, config MySQLConfig) MySQLConnectionResult {
	start := time.Now()

	db, err := sql.Open("mysql", config.DSN())
	if err != nil {
		return MySQLConnectionResult{
			Success: false,
			Error:   "Failed to open connection: " + err.Error(),
		}
	}
	defer db.Close()

	db.SetConnMaxLifetime(10 * time.Second)
	db.SetMaxOpenConns(1)

	var version string
	err = db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	if err != nil {
		return MySQLConnectionResult{
			Success: false,
			Error:   "Failed to query: " + err.Error(),
		}
	}

	serverType := "MySQL"
	if strings.Contains(strings.ToLower(version), "mariadb") {
		serverType = "MariaDB"
	}

	return MySQLConnectionResult{
		Success:      true,
		Version:      version,
		ServerType:   serverType,
		ResponseTime: float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

// GetMySQLInfo retrieves database information
func GetMySQLInfo(ctx context.Context, config MySQLConfig) MySQLInfo {
	db, err := sql.Open("mysql", config.DSN())
	if err != nil {
		return MySQLInfo{Error: "Failed to connect: " + err.Error()}
	}
	defer db.Close()

	info := MySQLInfo{}

	// Get version
	var version string
	db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	info.Version = version
	info.ServerType = "MySQL"
	if strings.Contains(strings.ToLower(version), "mariadb") {
		info.ServerType = "MariaDB"
	}

	// Get database size
	db.QueryRowContext(ctx, `
		SELECT CONCAT(ROUND(SUM(data_length + index_length) / 1024 / 1024, 2), ' MB')
		FROM information_schema.tables
		WHERE table_schema = ?
	`, config.Database).Scan(&info.DatabaseSize)

	// Get table count
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?
	`, config.Database).Scan(&info.TableCount)

	// Get status variables
	rows, err := db.QueryContext(ctx, "SHOW GLOBAL STATUS WHERE Variable_name IN ('Uptime', 'Threads_connected', 'Questions')")
	if err == nil {
		defer rows.Close()
		var questions int64
		for rows.Next() {
			var name string
			var value string
			rows.Scan(&name, &value)
			switch name {
			case "Uptime":
				fmt.Sscanf(value, "%d", &info.Uptime)
			case "Threads_connected":
				fmt.Sscanf(value, "%d", &info.Connections)
			case "Questions":
				fmt.Sscanf(value, "%d", &questions)
			}
		}
		if info.Uptime > 0 {
			info.QueriesPerSec = float64(questions) / float64(info.Uptime)
			info.UptimeHuman = formatMySQLUptime(info.Uptime)
		}
	}

	// Get max_connections
	db.QueryRowContext(ctx, "SELECT @@max_connections").Scan(&info.MaxConnections)

	// Get tables info
	tableRows, err := db.QueryContext(ctx, `
		SELECT
			table_name,
			COALESCE(engine, 'Unknown'),
			COALESCE(table_rows, 0),
			CONCAT(ROUND(data_length / 1024 / 1024, 2), ' MB'),
			CONCAT(ROUND(index_length / 1024 / 1024, 2), ' MB')
		FROM information_schema.tables
		WHERE table_schema = ?
		ORDER BY data_length DESC
		LIMIT 50
	`, config.Database)
	if err == nil {
		defer tableRows.Close()
		for tableRows.Next() {
			var t MySQLTable
			tableRows.Scan(&t.Name, &t.Engine, &t.RowCount, &t.DataSize, &t.IndexSize)
			info.Tables = append(info.Tables, t)
		}
	}

	return info
}

// ExecuteMySQLQuery executes a SQL query
func ExecuteMySQLQuery(ctx context.Context, config MySQLConfig, query string, readonly bool) MySQLQueryResult {
	start := time.Now()

	db, err := sql.Open("mysql", config.DSN())
	if err != nil {
		return MySQLQueryResult{Error: "Failed to connect: " + err.Error()}
	}
	defer db.Close()

	query = strings.TrimSpace(query)
	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(query), "SHOW") ||
		strings.HasPrefix(strings.ToUpper(query), "DESCRIBE") ||
		strings.HasPrefix(strings.ToUpper(query), "EXPLAIN")

	if readonly && !isSelect {
		return MySQLQueryResult{Error: "Only SELECT queries allowed in read-only mode"}
	}

	if isSelect {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return MySQLQueryResult{
				Error:    err.Error(),
				Duration: float64(time.Since(start).Microseconds()) / 1000.0,
			}
		}
		defer rows.Close()

		cols, _ := rows.Columns()
		result := MySQLQueryResult{
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
		return MySQLQueryResult{
			Error:    err.Error(),
			Duration: float64(time.Since(start).Microseconds()) / 1000.0,
		}
	}

	affected, _ := res.RowsAffected()
	return MySQLQueryResult{
		RowsAffected: affected,
		Duration:     float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

// DumpMySQL creates a database dump
func DumpMySQL(ctx context.Context, config MySQLConfig, schemaOnly bool, dataOnly bool, tables []string) MySQLDumpResult {
	start := time.Now()

	args := []string{
		"-h", config.Host,
		"-P", fmt.Sprintf("%d", config.Port),
		"-u", config.User,
		fmt.Sprintf("-p%s", config.Password),
		config.Database,
	}

	if schemaOnly {
		args = append(args, "--no-data")
	}
	if dataOnly {
		args = append(args, "--no-create-info")
	}
	for _, t := range tables {
		if t != "" {
			args = append(args, t)
		}
	}

	cmd := exec.CommandContext(ctx, "mysqldump", args...)

	output, err := cmd.Output()
	if err != nil {
		return MySQLDumpResult{
			Success:  false,
			Error:    "mysqldump failed: " + err.Error(),
			Duration: float64(time.Since(start).Microseconds()) / 1000.0,
		}
	}

	return MySQLDumpResult{
		Success:  true,
		Output:   string(output),
		Size:     int64(len(output)),
		Duration: float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

// GetMySQLDatabases lists all databases
func GetMySQLDatabases(ctx context.Context, config MySQLConfig) ([]string, error) {
	connConfig := config
	connConfig.Database = "information_schema"

	db, err := sql.Open("mysql", connConfig.DSN())
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		// Skip system databases
		if name != "information_schema" && name != "performance_schema" && name != "mysql" && name != "sys" {
			databases = append(databases, name)
		}
	}
	return databases, nil
}

func formatMySQLUptime(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	mins := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
