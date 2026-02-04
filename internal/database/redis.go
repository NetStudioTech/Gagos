package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig represents Redis connection configuration
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	UseTLS   bool   `json:"use_tls"`
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// RedisConnectionResult represents connection test result
type RedisConnectionResult struct {
	Success      bool    `json:"success"`
	Version      string  `json:"version,omitempty"`
	Mode         string  `json:"mode,omitempty"`
	ResponseTime float64 `json:"response_time_ms,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// RedisInfo represents Redis server information
type RedisInfo struct {
	// Server
	Version       string `json:"version"`
	Mode          string `json:"mode"`
	OS            string `json:"os"`
	Uptime        int64  `json:"uptime_seconds"`
	UptimeHuman   string `json:"uptime_human"`

	// Clients
	ConnectedClients int64 `json:"connected_clients"`
	BlockedClients   int64 `json:"blocked_clients"`

	// Memory
	UsedMemory       int64  `json:"used_memory"`
	UsedMemoryHuman  string `json:"used_memory_human"`
	UsedMemoryPeak   int64  `json:"used_memory_peak"`
	UsedMemoryPeakHuman string `json:"used_memory_peak_human"`
	MaxMemory        int64  `json:"maxmemory"`
	MaxMemoryHuman   string `json:"maxmemory_human"`
	MemFragRatio     float64 `json:"mem_fragmentation_ratio"`

	// Stats
	TotalConnections int64 `json:"total_connections_received"`
	TotalCommands    int64 `json:"total_commands_processed"`
	OpsPerSec        int64 `json:"instantaneous_ops_per_sec"`
	KeyspaceHits     int64 `json:"keyspace_hits"`
	KeyspaceMisses   int64 `json:"keyspace_misses"`
	HitRate          float64 `json:"hit_rate_percent"`

	// Replication
	Role             string `json:"role"`
	ConnectedSlaves  int64  `json:"connected_slaves"`
	MasterHost       string `json:"master_host,omitempty"`
	MasterPort       int    `json:"master_port,omitempty"`
	MasterLinkStatus string `json:"master_link_status,omitempty"`
	MasterSyncInProgress bool `json:"master_sync_in_progress,omitempty"`

	// Keyspace
	TotalKeys   int64            `json:"total_keys"`
	ExpiringKeys int64           `json:"expiring_keys"`
	Databases   []RedisDatabaseInfo `json:"databases,omitempty"`

	Error string `json:"error,omitempty"`
}

// RedisDatabaseInfo represents keyspace info for a database
type RedisDatabaseInfo struct {
	DB      int   `json:"db"`
	Keys    int64 `json:"keys"`
	Expires int64 `json:"expires"`
	AvgTTL  int64 `json:"avg_ttl"`
}

// RedisClusterInfo represents cluster information
type RedisClusterInfo struct {
	Enabled     bool              `json:"enabled"`
	State       string            `json:"state,omitempty"`
	SlotsOK     int64             `json:"slots_ok,omitempty"`
	SlotsFail   int64             `json:"slots_fail,omitempty"`
	KnownNodes  int64             `json:"known_nodes,omitempty"`
	ClusterSize int64             `json:"cluster_size,omitempty"`
	Nodes       []RedisClusterNode `json:"nodes,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// RedisClusterNode represents a cluster node
type RedisClusterNode struct {
	ID        string   `json:"id"`
	Addr      string   `json:"addr"`
	Flags     string   `json:"flags"`
	Master    string   `json:"master,omitempty"`
	PingSent  int64    `json:"ping_sent"`
	PongRecv  int64    `json:"pong_recv"`
	ConfigEpoch int64  `json:"config_epoch"`
	LinkState string   `json:"link_state"`
	Slots     []string `json:"slots,omitempty"`
}

// RedisKeyInfo represents key information
type RedisKeyInfo struct {
	Key      string      `json:"key"`
	Type     string      `json:"type"`
	TTL      int64       `json:"ttl"`
	Size     int64       `json:"size"`
	Value    interface{} `json:"value,omitempty"`
	Encoding string      `json:"encoding,omitempty"`
}

// RedisScanResult represents key scan result
type RedisScanResult struct {
	Keys   []RedisKeyInfo `json:"keys"`
	Cursor uint64         `json:"cursor"`
	Total  int64          `json:"total_scanned"`
	Error  string         `json:"error,omitempty"`
}

// RedisCommandResult represents command execution result
type RedisCommandResult struct {
	Result   interface{} `json:"result"`
	Type     string      `json:"type"`
	Duration float64     `json:"duration_ms"`
	Error    string      `json:"error,omitempty"`
}

// TestRedisConnection tests Redis connection
func TestRedisConnection(ctx context.Context, config RedisConfig) RedisConnectionResult {
	start := time.Now()

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr(),
		Password: config.Password,
		DB:       config.DB,
	})
	defer client.Close()

	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return RedisConnectionResult{
			Success: false,
			Error:   "Connection failed: " + err.Error(),
		}
	}

	if pong != "PONG" {
		return RedisConnectionResult{
			Success: false,
			Error:   "Unexpected response: " + pong,
		}
	}

	// Get version and mode
	info, _ := client.Info(ctx, "server").Result()
	version := parseRedisInfoValue(info, "redis_version")
	mode := parseRedisInfoValue(info, "redis_mode")

	return RedisConnectionResult{
		Success:      true,
		Version:      version,
		Mode:         mode,
		ResponseTime: float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

// GetRedisInfo retrieves Redis server information
func GetRedisInfo(ctx context.Context, config RedisConfig) RedisInfo {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr(),
		Password: config.Password,
		DB:       config.DB,
	})
	defer client.Close()

	info := RedisInfo{}

	// Get all info sections
	allInfo, err := client.Info(ctx).Result()
	if err != nil {
		return RedisInfo{Error: "Failed to get info: " + err.Error()}
	}

	// Parse server info
	info.Version = parseRedisInfoValue(allInfo, "redis_version")
	info.Mode = parseRedisInfoValue(allInfo, "redis_mode")
	info.OS = parseRedisInfoValue(allInfo, "os")
	info.Uptime, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "uptime_in_seconds"), 10, 64)
	info.UptimeHuman = formatUptime(info.Uptime)

	// Parse clients info
	info.ConnectedClients, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "connected_clients"), 10, 64)
	info.BlockedClients, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "blocked_clients"), 10, 64)

	// Parse memory info
	info.UsedMemory, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "used_memory"), 10, 64)
	info.UsedMemoryHuman = parseRedisInfoValue(allInfo, "used_memory_human")
	info.UsedMemoryPeak, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "used_memory_peak"), 10, 64)
	info.UsedMemoryPeakHuman = parseRedisInfoValue(allInfo, "used_memory_peak_human")
	info.MaxMemory, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "maxmemory"), 10, 64)
	info.MaxMemoryHuman = parseRedisInfoValue(allInfo, "maxmemory_human")
	info.MemFragRatio, _ = strconv.ParseFloat(parseRedisInfoValue(allInfo, "mem_fragmentation_ratio"), 64)

	// Parse stats
	info.TotalConnections, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "total_connections_received"), 10, 64)
	info.TotalCommands, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "total_commands_processed"), 10, 64)
	info.OpsPerSec, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "instantaneous_ops_per_sec"), 10, 64)
	info.KeyspaceHits, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "keyspace_hits"), 10, 64)
	info.KeyspaceMisses, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "keyspace_misses"), 10, 64)

	if info.KeyspaceHits+info.KeyspaceMisses > 0 {
		info.HitRate = float64(info.KeyspaceHits) / float64(info.KeyspaceHits+info.KeyspaceMisses) * 100
	}

	// Parse replication
	info.Role = parseRedisInfoValue(allInfo, "role")
	info.ConnectedSlaves, _ = strconv.ParseInt(parseRedisInfoValue(allInfo, "connected_slaves"), 10, 64)
	info.MasterHost = parseRedisInfoValue(allInfo, "master_host")
	info.MasterPort, _ = strconv.Atoi(parseRedisInfoValue(allInfo, "master_port"))
	info.MasterLinkStatus = parseRedisInfoValue(allInfo, "master_link_status")
	info.MasterSyncInProgress = parseRedisInfoValue(allInfo, "master_sync_in_progress") == "1"

	// Parse keyspace
	info.Databases = parseRedisKeyspace(allInfo)
	for _, db := range info.Databases {
		info.TotalKeys += db.Keys
		info.ExpiringKeys += db.Expires
	}

	return info
}

// GetRedisClusterInfo retrieves cluster information
func GetRedisClusterInfo(ctx context.Context, config RedisConfig) RedisClusterInfo {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr(),
		Password: config.Password,
		DB:       config.DB,
	})
	defer client.Close()

	// Check if cluster is enabled
	info, err := client.Info(ctx, "cluster").Result()
	if err != nil {
		return RedisClusterInfo{Error: "Failed to get cluster info: " + err.Error()}
	}

	enabled := parseRedisInfoValue(info, "cluster_enabled") == "1"
	if !enabled {
		return RedisClusterInfo{Enabled: false}
	}

	result := RedisClusterInfo{Enabled: true}

	// Get cluster info
	clusterInfo, err := client.ClusterInfo(ctx).Result()
	if err != nil {
		result.Error = "Failed to get cluster details: " + err.Error()
		return result
	}

	result.State = parseRedisInfoValue(clusterInfo, "cluster_state")
	result.SlotsOK, _ = strconv.ParseInt(parseRedisInfoValue(clusterInfo, "cluster_slots_ok"), 10, 64)
	result.SlotsFail, _ = strconv.ParseInt(parseRedisInfoValue(clusterInfo, "cluster_slots_fail"), 10, 64)
	result.KnownNodes, _ = strconv.ParseInt(parseRedisInfoValue(clusterInfo, "cluster_known_nodes"), 10, 64)
	result.ClusterSize, _ = strconv.ParseInt(parseRedisInfoValue(clusterInfo, "cluster_size"), 10, 64)

	// Get cluster nodes
	nodes, err := client.ClusterNodes(ctx).Result()
	if err == nil {
		result.Nodes = parseRedisClusterNodes(nodes)
	}

	return result
}

// ScanRedisKeys scans keys matching a pattern
func ScanRedisKeys(ctx context.Context, config RedisConfig, pattern string, cursor uint64, count int64) RedisScanResult {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr(),
		Password: config.Password,
		DB:       config.DB,
	})
	defer client.Close()

	if pattern == "" {
		pattern = "*"
	}
	if count <= 0 {
		count = 100
	}

	keys, newCursor, err := client.Scan(ctx, cursor, pattern, count).Result()
	if err != nil {
		return RedisScanResult{Error: "Scan failed: " + err.Error()}
	}

	result := RedisScanResult{
		Keys:   make([]RedisKeyInfo, 0, len(keys)),
		Cursor: newCursor,
		Total:  int64(len(keys)),
	}

	// Get info for each key
	for _, key := range keys {
		keyInfo := RedisKeyInfo{Key: key}

		keyType, _ := client.Type(ctx, key).Result()
		keyInfo.Type = keyType

		ttl, _ := client.TTL(ctx, key).Result()
		keyInfo.TTL = int64(ttl.Seconds())
		if keyInfo.TTL < 0 {
			keyInfo.TTL = -1 // No expiry
		}

		// Get memory usage if available
		memUsage, err := client.MemoryUsage(ctx, key).Result()
		if err == nil {
			keyInfo.Size = memUsage
		}

		result.Keys = append(result.Keys, keyInfo)
	}

	return result
}

// GetRedisKeyValue retrieves a key's value
func GetRedisKeyValue(ctx context.Context, config RedisConfig, key string) RedisKeyInfo {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr(),
		Password: config.Password,
		DB:       config.DB,
	})
	defer client.Close()

	info := RedisKeyInfo{Key: key}

	keyType, err := client.Type(ctx, key).Result()
	if err != nil {
		return RedisKeyInfo{Key: key, Type: "error"}
	}
	info.Type = keyType

	ttl, _ := client.TTL(ctx, key).Result()
	info.TTL = int64(ttl.Seconds())

	// Get encoding
	encoding, _ := client.ObjectEncoding(ctx, key).Result()
	info.Encoding = encoding

	// Get value based on type
	switch keyType {
	case "string":
		val, _ := client.Get(ctx, key).Result()
		info.Value = val
	case "list":
		val, _ := client.LRange(ctx, key, 0, 99).Result()
		info.Value = val
	case "set":
		val, _ := client.SMembers(ctx, key).Result()
		info.Value = val
	case "zset":
		val, _ := client.ZRangeWithScores(ctx, key, 0, 99).Result()
		info.Value = val
	case "hash":
		val, _ := client.HGetAll(ctx, key).Result()
		info.Value = val
	case "stream":
		val, _ := client.XRange(ctx, key, "-", "+").Result()
		if len(val) > 100 {
			val = val[:100]
		}
		info.Value = val
	}

	return info
}

// ExecuteRedisCommand executes a Redis command
func ExecuteRedisCommand(ctx context.Context, config RedisConfig, command string) RedisCommandResult {
	start := time.Now()

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr(),
		Password: config.Password,
		DB:       config.DB,
	})
	defer client.Close()

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return RedisCommandResult{Error: "Empty command"}
	}

	// Build args
	args := make([]interface{}, len(parts))
	for i, p := range parts {
		args[i] = p
	}

	result, err := client.Do(ctx, args...).Result()
	if err != nil {
		return RedisCommandResult{
			Error:    err.Error(),
			Duration: float64(time.Since(start).Microseconds()) / 1000.0,
		}
	}

	return RedisCommandResult{
		Result:   result,
		Type:     fmt.Sprintf("%T", result),
		Duration: float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

// Helper functions

func parseRedisInfoValue(info, key string) string {
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+":") {
			return strings.TrimPrefix(line, key+":")
		}
	}
	return ""
}

func parseRedisKeyspace(info string) []RedisDatabaseInfo {
	var databases []RedisDatabaseInfo
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "db") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		dbNum, _ := strconv.Atoi(strings.TrimPrefix(parts[0], "db"))
		db := RedisDatabaseInfo{DB: dbNum}

		// Parse keys=X,expires=Y,avg_ttl=Z
		for _, kv := range strings.Split(parts[1], ",") {
			kvParts := strings.Split(kv, "=")
			if len(kvParts) != 2 {
				continue
			}
			val, _ := strconv.ParseInt(kvParts[1], 10, 64)
			switch kvParts[0] {
			case "keys":
				db.Keys = val
			case "expires":
				db.Expires = val
			case "avg_ttl":
				db.AvgTTL = val
			}
		}
		databases = append(databases, db)
	}
	return databases
}

func parseRedisClusterNodes(nodesStr string) []RedisClusterNode {
	var nodes []RedisClusterNode
	lines := strings.Split(nodesStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 8 {
			continue
		}
		node := RedisClusterNode{
			ID:        parts[0],
			Addr:      parts[1],
			Flags:     parts[2],
			Master:    parts[3],
			LinkState: parts[7],
		}
		node.PingSent, _ = strconv.ParseInt(parts[4], 10, 64)
		node.PongRecv, _ = strconv.ParseInt(parts[5], 10, 64)
		node.ConfigEpoch, _ = strconv.ParseInt(parts[6], 10, 64)
		if len(parts) > 8 {
			node.Slots = parts[8:]
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func formatUptime(seconds int64) string {
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
