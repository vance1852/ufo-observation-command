package idempotency

import (
	"testing"
)

func TestHashRequestIsStableAndSensitiveToBody(t *testing.T) {
	first := HashRequest([]byte(`{"a":1}`))
	if first != HashRequest([]byte(`{"a":1}`)) {
		t.Fatal("hash is not stable")
	}
	if first == HashRequest([]byte(`{"a":2}`)) {
		t.Fatal("different bodies share hash")
	}
}

func TestRecordMatchesOnlySameRequestBody(t *testing.T) {
	record, err := NewRecord("key-1", []byte("body"), 201, map[string]string{"id": "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if !record.Matches([]byte("body")) {
		t.Fatal("same body did not match")
	}
	if record.Matches([]byte("other")) {
		t.Fatal("different body matched")
	}
	if len(record.ResponseBody) == 0 || record.ResponseCode != 201 {
		t.Fatalf("record = %+v", record)
	}
}

func TestRecordRequiresKey(t *testing.T) {
	if _, err := NewRecord("", []byte("body"), 200, nil); err == nil {
		t.Fatal("empty key accepted")
	}
}
