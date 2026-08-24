package domain

import "time"

type AuditSummary struct {
	ObjectType string
	ObjectID   string
	Action     string
	Outcome    string
	At         time.Time
}

func (a AuditSummary) Successful() bool { return a.Outcome == "success" }
