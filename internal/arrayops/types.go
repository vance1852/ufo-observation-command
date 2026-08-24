package arrayops

import (
	"errors"
	"time"
)

var (
	ErrConflict     = errors.New("arrayops conflict")
	ErrUnauthorized = errors.New("arrayops unauthorized")
	ErrUnavailable  = errors.New("arrayops unavailable")
	ErrInvalid      = errors.New("arrayops invalid")
)

type CalibrationBundle struct {
	ID          string
	TenantID    string
	Version     string
	Digest      string
	BuoyClasses []string
	Labels      map[string]string
	Signed      bool
}

type Buoy struct {
	ID              string
	TenantID        string
	Class           string
	CurrentVersion  string
	PreviousVersion string
	Generation      int64
	Quarantined     bool
}

type SurveyMission struct {
	ID                  string
	TenantID            string
	CalibrationBundleID string
	State               string
	RequiredHealthy     int
	Healthy             int
	Failed              int
	BuoyIDs             []string
	ApprovedDigest      string
	Version             int64
}

type Callback struct {
	TenantID            string
	BuoyID              string
	CalibrationBundleID string
	EventID             string
	Status              string
	At                  time.Time
}

type Session struct {
	Token     string
	TenantID  string
	UserID    string
	Role      string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type Event struct {
	TenantID  string
	RequestID string
	ObjectID  string
	Action    string
	Digest    string
	At        time.Time
}

type Query struct {
	TenantID string
	State    string
	Class    string
	Limit    int
	Offset   int
}

type Page[T any] struct {
	Items []T
	Total int
}
