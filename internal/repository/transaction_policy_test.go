package repository

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestTransactionRetryClassificationUsesPostgresCodes(t *testing.T) {
	for _, code := range []string{"40001", "40P01"} {
		err := fmt.Errorf("transaction failed: %w", &pgconn.PgError{Code: code})
		if !retryableTransactionError(err) {
			t.Fatalf("code %s was not retryable", code)
		}
	}
	if retryableTransactionError(fmt.Errorf("customer note contains serialization text")) {
		t.Fatal("plain application text was classified as a database retry")
	}
}
