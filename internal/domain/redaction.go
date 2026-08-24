package domain

import "strings"

func RedactTaskCode(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func RedactArrayOpsLane(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 3 {
		return "***"
	}
	return value[:1] + "***" + value[len(value)-1:]
}
