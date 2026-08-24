package audit

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Event struct {
	RequestID       string         `json:"request_id"`
	ArrayOperatorID *string        `json:"array_operator_id,omitempty"`
	ObjectType      string         `json:"object_type"`
	ObjectID        string         `json:"object_id"`
	Action          string         `json:"action"`
	Outcome         string         `json:"outcome"`
	Detail          map[string]any `json:"detail,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

func (e Event) Validate() error {
	if strings.TrimSpace(e.RequestID) == "" || strings.TrimSpace(e.ObjectType) == "" || strings.TrimSpace(e.ObjectID) == "" {
		return fmt.Errorf("audit identity is required")
	}
	if strings.TrimSpace(e.Action) == "" || strings.TrimSpace(e.Outcome) == "" {
		return fmt.Errorf("audit action and outcome are required")
	}
	return nil
}

func (e Event) JSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if e.Detail == nil {
		e.Detail = map[string]any{}
	}
	return json.Marshal(e.Detail)
}
