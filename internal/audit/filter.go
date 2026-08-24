package audit

import "strings"

type Filter struct {
	ObjectType string
	ObjectID   string
	Action     string
	Outcome    string
}

func (f Filter) Match(event Event) bool {
	if f.ObjectType != "" && event.ObjectType != f.ObjectType {
		return false
	}
	if f.ObjectID != "" && event.ObjectID != f.ObjectID {
		return false
	}
	if f.Action != "" && event.Action != f.Action {
		return false
	}
	if f.Outcome != "" && event.Outcome != f.Outcome {
		return false
	}
	return strings.TrimSpace(event.RequestID) != ""
}

func FilterEvents(events []Event, filter Filter) []Event {
	out := make([]Event, 0)
	for _, event := range events {
		if filter.Match(event) {
			out = append(out, event)
		}
	}
	return out
}
