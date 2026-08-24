package pagination

import "testing"

func TestNormalizeBoundsQuery(t *testing.T) {
	if got := Normalize(-1, 0); got.Offset != 0 || got.Limit != DefaultLimit {
		t.Fatalf("default query = %+v", got)
	}
	if got := Normalize(3, MaxLimit+10); got.Offset != 3 || got.Limit != MaxLimit {
		t.Fatalf("bounded query = %+v", got)
	}
}

func TestPageCopiesItems(t *testing.T) {
	items := []string{"a", "b"}
	page := From(items, 2, Query{Offset: 0, Limit: 50})
	page.Items[0] = "changed"
	if items[0] != "a" {
		t.Fatal("source slice was polluted")
	}
}
