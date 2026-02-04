package database

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ESConfig holds Elasticsearch connection configuration
type ESConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	UseSSL   bool   `json:"use_ssl"`
}

// ESConnectionResult holds connection test result
type ESConnectionResult struct {
	Success      bool    `json:"success"`
	ClusterName  string  `json:"cluster_name,omitempty"`
	Version      string  `json:"version,omitempty"`
	ResponseTime float64 `json:"response_time_ms,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// ESClusterHealth holds cluster health info
type ESClusterHealth struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

// ESClusterStats holds cluster statistics
type ESClusterStats struct {
	ClusterName string `json:"cluster_name"`
	ClusterUUID string `json:"cluster_uuid"`
	Status      string `json:"status"`
	Indices     struct {
		Count  int `json:"count"`
		Shards struct {
			Total      int     `json:"total"`
			Primaries  int     `json:"primaries"`
			Replication float64 `json:"replication"`
		} `json:"shards"`
		Docs struct {
			Count   int64 `json:"count"`
			Deleted int64 `json:"deleted"`
		} `json:"docs"`
		Store struct {
			SizeInBytes int64 `json:"size_in_bytes"`
		} `json:"store"`
	} `json:"indices"`
	Nodes struct {
		Count struct {
			Total            int `json:"total"`
			CoordinatingOnly int `json:"coordinating_only"`
			Data             int `json:"data"`
			Ingest           int `json:"ingest"`
			Master           int `json:"master"`
		} `json:"count"`
	} `json:"nodes"`
}

// ESIndex holds index information
type ESIndex struct {
	Index        string `json:"index"`
	Health       string `json:"health"`
	Status       string `json:"status"`
	UUID         string `json:"uuid"`
	Pri          string `json:"pri"`
	Rep          string `json:"rep"`
	DocsCount    string `json:"docs.count"`
	DocsDeleted  string `json:"docs.deleted"`
	StoreSize    string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
}

// ESSearchResult holds search results
type ESSearchResult struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index  string                 `json:"_index"`
			ID     string                 `json:"_id"`
			Score  float64                `json:"_score"`
			Source map[string]interface{} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// ESQueryResult holds generic query result
type ESQueryResult struct {
	StatusCode int             `json:"status_code"`
	Body       json.RawMessage `json:"body"`
	Error      string          `json:"error,omitempty"`
}

func (c *ESConfig) baseURL() string {
	scheme := "http"
	if c.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, c.Host, c.Port)
}

func (c *ESConfig) newClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func (c *ESConfig) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL() + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	return c.newClient().Do(req)
}

// TestESConnection tests connection to Elasticsearch
func TestESConnection(ctx context.Context, config ESConfig) ESConnectionResult {
	start := time.Now()

	resp, err := config.doRequest(ctx, "GET", "/", nil)
	if err != nil {
		return ESConnectionResult{
			Success: false,
			Error:   err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ESConnectionResult{
			Success: false,
			Error:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}

	var result struct {
		Name        string `json:"name"`
		ClusterName string `json:"cluster_name"`
		Version     struct {
			Number string `json:"number"`
		} `json:"version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ESConnectionResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	return ESConnectionResult{
		Success:      true,
		ClusterName:  result.ClusterName,
		Version:      result.Version.Number,
		ResponseTime: float64(time.Since(start).Milliseconds()),
	}
}

// GetESClusterHealth gets cluster health
func GetESClusterHealth(ctx context.Context, config ESConfig) (*ESClusterHealth, error) {
	resp, err := config.doRequest(ctx, "GET", "/_cluster/health", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var health ESClusterHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}

	return &health, nil
}

// GetESClusterStats gets cluster statistics
func GetESClusterStats(ctx context.Context, config ESConfig) (*ESClusterStats, error) {
	resp, err := config.doRequest(ctx, "GET", "/_cluster/stats", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var stats ESClusterStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// ListESIndices lists all indices
func ListESIndices(ctx context.Context, config ESConfig) ([]ESIndex, error) {
	resp, err := config.doRequest(ctx, "GET", "/_cat/indices?format=json&h=index,health,status,uuid,pri,rep,docs.count,docs.deleted,store.size,pri.store.size", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var indices []ESIndex
	if err := json.NewDecoder(resp.Body).Decode(&indices); err != nil {
		return nil, err
	}

	return indices, nil
}

// CreateESIndex creates a new index
func CreateESIndex(ctx context.Context, config ESConfig, index string, settings json.RawMessage) error {
	var body io.Reader
	if settings != nil {
		body = bytes.NewReader(settings)
	}

	resp, err := config.doRequest(ctx, "PUT", "/"+index, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteESIndex deletes an index
func DeleteESIndex(ctx context.Context, config ESConfig, index string) error {
	resp, err := config.doRequest(ctx, "DELETE", "/"+index, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetESIndexMapping gets index mapping
func GetESIndexMapping(ctx context.Context, config ESConfig, index string) (json.RawMessage, error) {
	resp, err := config.doRequest(ctx, "GET", "/"+index+"/_mapping", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// GetESIndexSettings gets index settings
func GetESIndexSettings(ctx context.Context, config ESConfig, index string) (json.RawMessage, error) {
	resp, err := config.doRequest(ctx, "GET", "/"+index+"/_settings", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// SearchESDocuments searches documents in an index
func SearchESDocuments(ctx context.Context, config ESConfig, index, query string, from, size int) (*ESSearchResult, error) {
	var body io.Reader
	if query != "" {
		// If query looks like JSON, use it directly; otherwise wrap as simple query string
		query = strings.TrimSpace(query)
		if !strings.HasPrefix(query, "{") {
			queryJSON := map[string]interface{}{
				"query": map[string]interface{}{
					"query_string": map[string]interface{}{
						"query": query,
					},
				},
				"from": from,
				"size": size,
			}
			jsonBytes, _ := json.Marshal(queryJSON)
			body = bytes.NewReader(jsonBytes)
		} else {
			body = strings.NewReader(query)
		}
	} else {
		queryJSON := map[string]interface{}{
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
			"from": from,
			"size": size,
		}
		jsonBytes, _ := json.Marshal(queryJSON)
		body = bytes.NewReader(jsonBytes)
	}

	path := "/" + index + "/_search"
	resp, err := config.doRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result ESSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetESDocument gets a document by ID
func GetESDocument(ctx context.Context, config ESConfig, index, id string) (json.RawMessage, error) {
	resp, err := config.doRequest(ctx, "GET", "/"+index+"/_doc/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// DeleteESDocument deletes a document by ID
func DeleteESDocument(ctx context.Context, config ESConfig, index, id string) error {
	resp, err := config.doRequest(ctx, "DELETE", "/"+index+"/_doc/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ExecuteESQuery executes a raw REST API query
func ExecuteESQuery(ctx context.Context, config ESConfig, method, path string, body json.RawMessage) (*ESQueryResult, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	resp, err := config.doRequest(ctx, method, path, bodyReader)
	if err != nil {
		return &ESQueryResult{
			Error: err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ESQueryResult{
			StatusCode: resp.StatusCode,
			Error:      err.Error(),
		}, nil
	}

	return &ESQueryResult{
		StatusCode: resp.StatusCode,
		Body:       respBody,
	}, nil
}

// RefreshESIndex refreshes an index
func RefreshESIndex(ctx context.Context, config ESConfig, index string) error {
	resp, err := config.doRequest(ctx, "POST", "/"+index+"/_refresh", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetESNodes gets cluster nodes info
func GetESNodes(ctx context.Context, config ESConfig) (json.RawMessage, error) {
	resp, err := config.doRequest(ctx, "GET", "/_cat/nodes?format=json&h=name,ip,heap.percent,ram.percent,cpu,load_1m,node.role,master", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
