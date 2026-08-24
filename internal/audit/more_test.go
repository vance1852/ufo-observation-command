package audit

import (
	"strings"
	"testing"
	"time"
)

func TestExportCSVRejectsInvalidEvent(t *testing.T) {
	var output strings.Builder
	err := ExportCSV(&output, []Event{{ObjectType: "task"}})
	if err == nil {
		t.Fatal("invalid event exported")
	}
}

func TestFilterRequiresRequestCorrelation(t *testing.T) {
	event := Event{ObjectType: "task", ObjectID: "s1", Action: "create", Outcome: "success", CreatedAt: time.Now()}
	if (Filter{}).Match(event) {
		t.Fatal("uncorrelated event matched")
	}
}

func TestChainRejectsInvalidEvent(t *testing.T) {
	var chain Chain
	if _, err := chain.Append(Event{}); err == nil {
		t.Fatal("invalid event appended")
	}
}
