package domain

import (
	"slices"
	"testing"
)

func TestAssignmentCompletionRequiresActiveState(t *testing.T) {
	allowed, ok := AssignmentSourcesFor("completed")
	if !ok {
		t.Fatal("completed transition is not recognized")
	}
	if !slices.Equal(allowed, []string{"active"}) {
		t.Fatalf("allowed sources=%v", allowed)
	}
}
