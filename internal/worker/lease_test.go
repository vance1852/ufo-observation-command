package worker

import (
	"testing"
	"time"
)

func TestLeaseAllowsOnlyOneOwner(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	var lease Lease
	if !lease.Acquire("worker-a", now, time.Minute) {
		t.Fatal("first acquire failed")
	}
	if lease.Acquire("worker-b", now, time.Minute) {
		t.Fatal("second owner acquired active lease")
	}
	if !lease.HeldBy("worker-a", now.Add(30*time.Second)) {
		t.Fatal("owner does not hold lease")
	}
	if !lease.Release("worker-b") {
		if lease.Release("worker-a") == false {
			t.Fatal("owner could not release lease")
		}
	}
}

func TestLeaseCanBeTakenAfterExpiry(t *testing.T) {
	now := time.Now().UTC()
	var lease Lease
	if !lease.Acquire("a", now, time.Second) {
		t.Fatal("acquire failed")
	}
	if !lease.Acquire("b", now.Add(2*time.Second), time.Second) {
		t.Fatal("expired lease not replaced")
	}
}
