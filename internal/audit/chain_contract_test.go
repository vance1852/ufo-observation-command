package audit

import "testing"

func TestChainCommitsToEventIdentity(t *testing.T) {
	firstEvent := validEvent("create", "success")
	secondEvent := firstEvent
	secondEvent.RequestID = "req-2"

	var firstChain, secondChain Chain
	first, err := firstChain.Append(firstEvent)
	if err != nil {
		t.Fatal(err)
	}
	second, err := secondChain.Append(secondEvent)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("events with different correlation ids produced the same chain head")
	}
}
