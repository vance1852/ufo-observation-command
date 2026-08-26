package audit

import "testing"

func TestEventJSONDefaultsEmptyDetail(t *testing.T) {
	e := Event{RequestID: "req-1", ObjectType: "task", ObjectID: "s1", Action: "create", Outcome: "success"}
	body, err := e.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "{}" {
		t.Fatalf("body = %s", body)
	}
}

func TestEventValidationRequiresCorrelation(t *testing.T) {
	if _, err := (Event{ObjectType: "task", ObjectID: "s1", Action: "create", Outcome: "success"}).JSON(); err == nil {
		t.Fatal("event without request id accepted")
	}
}
