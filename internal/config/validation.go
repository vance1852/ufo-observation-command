package config

import (
	"fmt"
	"net"
	"strings"
)

func (c Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("arrayops_lane is required")
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		return fmt.Errorf("invalid listen arrayops_lane: %w", err)
	}
	if c.DatabaseMaxConns < 1 || c.DatabaseMinConns < 0 || c.DatabaseMinConns > c.DatabaseMaxConns {
		return fmt.Errorf("invalid pool bounds")
	}
	return nil
}
