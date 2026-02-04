package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// DiffResult represents the result of a diff operation
type DiffResult struct {
	Type       string     `json:"type"`
	Additions  int        `json:"additions"`
	Deletions  int        `json:"deletions"`
	Changes    int        `json:"changes"`
	Diff       string     `json:"diff"`
	DiffLines  []DiffLine `json:"diff_lines,omitempty"`
	Identical  bool       `json:"identical"`
	Error      string     `json:"error,omitempty"`
}

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type    string `json:"type"` // "add", "delete", "unchanged", "context"
	LineNum int    `json:"line_num"`
	Content string `json:"content"`
}

// TextDiff computes a line-by-line diff of two texts
func TextDiff(text1, text2 string) DiffResult {
	lines1 := strings.Split(text1, "\n")
	lines2 := strings.Split(text2, "\n")

	// Simple LCS-based diff
	lcs := computeLCS(lines1, lines2)
	diff, additions, deletions := generateDiff(lines1, lines2, lcs)

	identical := additions == 0 && deletions == 0

	return DiffResult{
		Type:      "text",
		Additions: additions,
		Deletions: deletions,
		Changes:   additions + deletions,
		Diff:      strings.Join(formatDiffLines(diff), "\n"),
		DiffLines: diff,
		Identical: identical,
	}
}

// JSONDiff computes a diff between two JSON strings
func JSONDiff(json1, json2 string) DiffResult {
	var data1, data2 interface{}

	if err := json.Unmarshal([]byte(json1), &data1); err != nil {
		return DiffResult{Type: "json", Error: "Failed to parse first JSON: " + err.Error()}
	}
	if err := json.Unmarshal([]byte(json2), &data2); err != nil {
		return DiffResult{Type: "json", Error: "Failed to parse second JSON: " + err.Error()}
	}

	// Format both for consistent comparison
	formatted1, _ := json.MarshalIndent(data1, "", "  ")
	formatted2, _ := json.MarshalIndent(data2, "", "  ")

	return TextDiff(string(formatted1), string(formatted2))
}

// YAMLDiff computes a diff between two YAML strings
func YAMLDiff(yaml1, yaml2 string) DiffResult {
	var data1, data2 interface{}

	if err := yaml.Unmarshal([]byte(yaml1), &data1); err != nil {
		return DiffResult{Type: "yaml", Error: "Failed to parse first YAML: " + err.Error()}
	}
	if err := yaml.Unmarshal([]byte(yaml2), &data2); err != nil {
		return DiffResult{Type: "yaml", Error: "Failed to parse second YAML: " + err.Error()}
	}

	// Convert to JSON for consistent formatting, then back to YAML
	jsonBytes1, _ := json.MarshalIndent(data1, "", "  ")
	jsonBytes2, _ := json.MarshalIndent(data2, "", "  ")

	var normalized1, normalized2 interface{}
	json.Unmarshal(jsonBytes1, &normalized1)
	json.Unmarshal(jsonBytes2, &normalized2)

	formatted1, _ := yaml.Marshal(normalized1)
	formatted2, _ := yaml.Marshal(normalized2)

	result := TextDiff(string(formatted1), string(formatted2))
	result.Type = "yaml"
	return result
}

// computeLCS computes the longest common subsequence
func computeLCS(a, b []string) [][]int {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	return dp
}

// generateDiff generates diff lines using the LCS matrix
func generateDiff(a, b []string, lcs [][]int) ([]DiffLine, int, int) {
	var diff []DiffLine
	additions, deletions := 0, 0

	i, j := len(a), len(b)
	var stack []DiffLine

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			stack = append(stack, DiffLine{Type: "unchanged", Content: a[i-1], LineNum: i})
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			stack = append(stack, DiffLine{Type: "add", Content: b[j-1], LineNum: j})
			additions++
			j--
		} else if i > 0 && (j == 0 || lcs[i][j-1] < lcs[i-1][j]) {
			stack = append(stack, DiffLine{Type: "delete", Content: a[i-1], LineNum: i})
			deletions++
			i--
		}
	}

	// Reverse the stack to get correct order
	for k := len(stack) - 1; k >= 0; k-- {
		diff = append(diff, stack[k])
	}

	return diff, additions, deletions
}

// formatDiffLines formats diff lines for display
func formatDiffLines(lines []DiffLine) []string {
	var result []string
	for _, line := range lines {
		var prefix string
		switch line.Type {
		case "add":
			prefix = "+ "
		case "delete":
			prefix = "- "
		default:
			prefix = "  "
		}
		result = append(result, fmt.Sprintf("%s%s", prefix, line.Content))
	}
	return result
}
