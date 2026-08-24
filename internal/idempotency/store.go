package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type Record struct {
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
}

func HashRequest(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

func NewRecord(key string, body []byte, code int, response any) (Record, error) {
	if key == "" {
		return Record{}, fmt.Errorf("idempotency key is required")
	}
	payload, err := json.Marshal(response)
	if err != nil {
		return Record{}, fmt.Errorf("marshal response: %w", err)
	}
	return Record{Key: key, RequestHash: HashRequest(body), ResponseCode: code, ResponseBody: payload}, nil
}

func (r Record) Matches(body []byte) bool { return r.RequestHash == HashRequest(body) }
