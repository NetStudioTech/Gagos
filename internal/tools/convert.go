package tools

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// ConvertResult represents the result of a format conversion
type ConvertResult struct {
	Input  string `json:"input,omitempty"`
	Output string `json:"output"`
	From   string `json:"from"`
	To     string `json:"to"`
	Error  string `json:"error,omitempty"`
}

// CSVToJSON converts CSV to JSON
func CSVToJSON(input string) ConvertResult {
	reader := csv.NewReader(strings.NewReader(input))
	records, err := reader.ReadAll()
	if err != nil {
		return ConvertResult{From: "CSV", To: "JSON", Error: "Failed to parse CSV: " + err.Error()}
	}

	if len(records) < 2 {
		return ConvertResult{From: "CSV", To: "JSON", Error: "CSV must have header row and at least one data row"}
	}

	headers := records[0]
	var result []map[string]string

	for _, row := range records[1:] {
		item := make(map[string]string)
		for i, value := range row {
			if i < len(headers) {
				item[headers[i]] = value
			}
		}
		result = append(result, item)
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ConvertResult{From: "CSV", To: "JSON", Error: "Failed to generate JSON: " + err.Error()}
	}

	return ConvertResult{
		Output: string(output),
		From:   "CSV",
		To:     "JSON",
	}
}

// JSONToYAML converts JSON to YAML
func JSONToYAML(input string) ConvertResult {
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return ConvertResult{From: "JSON", To: "YAML", Error: "Failed to parse JSON: " + err.Error()}
	}

	output, err := yaml.Marshal(data)
	if err != nil {
		return ConvertResult{From: "JSON", To: "YAML", Error: "Failed to generate YAML: " + err.Error()}
	}

	return ConvertResult{
		Output: string(output),
		From:   "JSON",
		To:     "YAML",
	}
}

// YAMLToJSON converts YAML to JSON
func YAMLToJSON(input string) ConvertResult {
	var data interface{}
	if err := yaml.Unmarshal([]byte(input), &data); err != nil {
		return ConvertResult{From: "YAML", To: "JSON", Error: "Failed to parse YAML: " + err.Error()}
	}

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ConvertResult{From: "YAML", To: "JSON", Error: "Failed to generate JSON: " + err.Error()}
	}

	return ConvertResult{
		Output: string(output),
		From:   "YAML",
		To:     "JSON",
	}
}

// XMLToJSON converts XML to JSON
func XMLToJSON(input string) ConvertResult {
	var data interface{}

	// Parse XML
	decoder := xml.NewDecoder(strings.NewReader(input))
	var stack []map[string]interface{}
	var current map[string]interface{}

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			newMap := make(map[string]interface{})
			if current != nil {
				stack = append(stack, current)
				if existing, ok := current[t.Name.Local]; ok {
					switch v := existing.(type) {
					case []interface{}:
						current[t.Name.Local] = append(v, newMap)
					default:
						current[t.Name.Local] = []interface{}{v, newMap}
					}
				} else {
					current[t.Name.Local] = newMap
				}
			}
			current = newMap
			// Add attributes
			for _, attr := range t.Attr {
				current["@"+attr.Name.Local] = attr.Value
			}
		case xml.EndElement:
			if len(stack) > 0 {
				current = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" && current != nil {
				current["#text"] = text
			}
		}
	}

	if current == nil {
		return ConvertResult{From: "XML", To: "JSON", Error: "Failed to parse XML"}
	}

	data = current
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ConvertResult{From: "XML", To: "JSON", Error: "Failed to generate JSON: " + err.Error()}
	}

	return ConvertResult{
		Output: string(output),
		From:   "XML",
		To:     "JSON",
	}
}

// TOMLToYAML converts TOML to YAML
func TOMLToYAML(input string) ConvertResult {
	var data interface{}
	if err := toml.Unmarshal([]byte(input), &data); err != nil {
		return ConvertResult{From: "TOML", To: "YAML", Error: "Failed to parse TOML: " + err.Error()}
	}

	output, err := yaml.Marshal(data)
	if err != nil {
		return ConvertResult{From: "TOML", To: "YAML", Error: "Failed to generate YAML: " + err.Error()}
	}

	return ConvertResult{
		Output: string(output),
		From:   "TOML",
		To:     "YAML",
	}
}

// YAMLToTOML converts YAML to TOML
func YAMLToTOML(input string) ConvertResult {
	var data interface{}
	if err := yaml.Unmarshal([]byte(input), &data); err != nil {
		return ConvertResult{From: "YAML", To: "TOML", Error: "Failed to parse YAML: " + err.Error()}
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return ConvertResult{From: "YAML", To: "TOML", Error: "Failed to generate TOML: " + err.Error()}
	}

	return ConvertResult{
		Output: buf.String(),
		From:   "YAML",
		To:     "TOML",
	}
}

// PropertiesToYAML converts Java properties format to YAML
func PropertiesToYAML(input string) ConvertResult {
	data := make(map[string]interface{})

	lines := strings.Split(input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}

		// Split on = or :
		var key, value string
		if idx := strings.Index(line, "="); idx != -1 {
			key = strings.TrimSpace(line[:idx])
			value = strings.TrimSpace(line[idx+1:])
		} else if idx := strings.Index(line, ":"); idx != -1 {
			key = strings.TrimSpace(line[:idx])
			value = strings.TrimSpace(line[idx+1:])
		} else {
			continue
		}

		// Handle nested keys (e.g., server.port)
		parts := strings.Split(key, ".")
		setNestedValue(data, parts, value)
	}

	output, err := yaml.Marshal(data)
	if err != nil {
		return ConvertResult{From: "Properties", To: "YAML", Error: "Failed to generate YAML: " + err.Error()}
	}

	return ConvertResult{
		Output: string(output),
		From:   "Properties",
		To:     "YAML",
	}
}

func setNestedValue(data map[string]interface{}, keys []string, value string) {
	for i, key := range keys {
		if i == len(keys)-1 {
			data[key] = value
		} else {
			if _, ok := data[key]; !ok {
				data[key] = make(map[string]interface{})
			}
			if nested, ok := data[key].(map[string]interface{}); ok {
				data = nested
			} else {
				return
			}
		}
	}
}

// FormatJSON formats/prettifies JSON
func FormatJSON(input string) ConvertResult {
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return ConvertResult{From: "JSON", To: "JSON", Error: "Failed to parse JSON: " + err.Error()}
	}

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ConvertResult{From: "JSON", To: "JSON", Error: "Failed to format JSON: " + err.Error()}
	}

	return ConvertResult{
		Output: string(output),
		From:   "JSON",
		To:     "JSON (formatted)",
	}
}

// MinifyJSON minifies JSON
func MinifyJSON(input string) ConvertResult {
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return ConvertResult{From: "JSON", To: "JSON", Error: "Failed to parse JSON: " + err.Error()}
	}

	output, err := json.Marshal(data)
	if err != nil {
		return ConvertResult{From: "JSON", To: "JSON", Error: "Failed to minify JSON: " + err.Error()}
	}

	return ConvertResult{
		Output: string(output),
		From:   "JSON",
		To:     "JSON (minified)",
	}
}
