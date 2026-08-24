package audit

import (
	"encoding/csv"
	"fmt"
	"io"
)

func ExportCSV(w io.Writer, events []Event) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"request_id", "object_type", "object_id", "action", "outcome", "created_at"}); err != nil {
		return fmt.Errorf("write audit header: %w", err)
	}
	for _, event := range events {
		if err := event.Validate(); err != nil {
			return err
		}
		if err := writer.Write([]string{event.RequestID, event.ObjectType, event.ObjectID, event.Action, event.Outcome, event.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")}); err != nil {
			return fmt.Errorf("write audit row: %w", err)
		}
	}
	writer.Flush()
	return writer.Error()
}
