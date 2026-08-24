package httpapi

import (
	"net/http"
	"strings"
)

func boolQuery(r *http.Request, key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}

func setNoCache(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store, max-age=0") }
