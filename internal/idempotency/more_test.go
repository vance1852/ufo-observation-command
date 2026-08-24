package idempotency

import "testing"

func TestHashHasExpectedLength(t *testing.T) {
	if len(HashRequest([]byte("request"))) != 64 {
		t.Fatal("sha256 hex length is wrong")
	}
}

func TestResponseBodyIsJSON(t *testing.T) {
	record, err := NewRecord("key", []byte("body"), 200, map[string]any{"ok": true})
	if err != nil {
		t.Fatal(err)
	}
	if string(record.ResponseBody) != `{"ok":true}` {
		t.Fatalf("body=%s", record.ResponseBody)
	}
}
