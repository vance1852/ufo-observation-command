package httpapi

import (
	"fmt"
	"net/http"
	"strings"
)

func requiredHeader(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(r.Header.Get(name))
	if value == "" {
		return "", fmt.Errorf("header %s is required", name)
	}
	return value, nil
}
