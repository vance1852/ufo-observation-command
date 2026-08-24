package domain

import "testing"

func TestNormalizeLabelsCopiesAndCleansValues(t *testing.T) {
	labels := map[string]string{" Acoustic_Buoy ": " North ", "": "ignored", "empty": " "}
	got := NormalizeLabels(labels)
	if got["acoustic_buoy"] != "North" || len(got) != 1 {
		t.Fatalf("labels=%v", got)
	}
	got["acoustic_buoy"] = "changed"
	if labels[" Acoustic_Buoy "] != " North " {
		t.Fatal("source labels were mutated")
	}
}
