package domain

import "strings"

func NormalizeLabels(labels map[string]string) map[string]string {
	result := make(map[string]string, len(labels))
	for key, value := range labels {
		key, value = strings.TrimSpace(strings.ToLower(key)), strings.TrimSpace(value)
		if key != "" && value != "" {
			result[key] = value
		}
	}
	return result
}
