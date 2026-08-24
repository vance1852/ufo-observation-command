package audit

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func validEvent(action, outcome string) Event {
	return Event{RequestID: "req-1", ObjectType: "task", ObjectID: "s-1", Action: action, Outcome: outcome, CreatedAt: time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)}
}

func TestChainChangesHeadAndPreservesSequence(t *testing.T) {
	var chain Chain
	first, err := chain.Append(validEvent("create", "success"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := chain.Append(validEvent("complete", "success"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second || chain.Head() != second {
		t.Fatalf("head=%s first=%s second=%s", chain.Head(), first, second)
	}
}

func TestFilterEventsMatchesAllProvidedFields(t *testing.T) {
	events := []Event{validEvent("create", "success"), validEvent("review", "success"), validEvent("review", "failure")}
	filtered := FilterEvents(events, Filter{Action: "review", Outcome: "success"})
	if len(filtered) != 1 || filtered[0].Action != "review" {
		t.Fatalf("filtered=%+v", filtered)
	}
	if len(FilterEvents(events, Filter{ObjectID: "missing"})) != 0 {
		t.Fatal("unmatched object returned")
	}
}

func TestExportCSVHasHeaderAndRows(t *testing.T) {
	var output bytes.Buffer
	if err := ExportCSV(&output, []Event{validEvent("create", "success")}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "request_id,object_type") || strings.Count(text, "req-1") != 1 {
		t.Fatalf("csv=%s", text)
	}
}
