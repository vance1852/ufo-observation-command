package config

import "strings"

func RedactDatabaseURL(url string) string {
	if index := strings.Index(url, "@"); index >= 0 {
		if scheme := strings.Index(url, "://"); scheme >= 0 {
			return url[:scheme+3] + "***:***" + url[index:]
		}
	}
	return url
}

func (c Config) Public() map[string]any {
	return map[string]any{"addr": c.Addr, "database": RedactDatabaseURL(c.DatabaseURL), "max_conns": c.DatabaseMaxConns, "min_conns": c.DatabaseMinConns, "worker_interval": c.WorkerInterval.String()}
}
