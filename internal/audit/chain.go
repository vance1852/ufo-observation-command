package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type Chain struct{ previous string }

func (c *Chain) Append(event Event) (string, error) {
	if err := event.Validate(); err != nil {
		return "", err
	}
	if event.Detail == nil {
		event.Detail = map[string]any{}
	}
	body, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("marshal audit event: %w", err)
	}
	payload, err := json.Marshal(struct {
		Previous string          `json:"previous"`
		Event    json.RawMessage `json:"event"`
	}{Previous: c.previous, Event: body})
	if err != nil {
		return "", fmt.Errorf("marshal audit chain: %w", err)
	}
	hash := sha256.Sum256(payload)
	c.previous = hex.EncodeToString(hash[:])
	return c.previous, nil
}

func (c Chain) Head() string { return c.previous }
