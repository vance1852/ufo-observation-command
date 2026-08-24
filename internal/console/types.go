package console

import "time"

type Role string

const (
	RoleMissionLead   Role = "mission_lead"
	RoleArrayOperator Role = "array_operator"
	RoleReviewer      Role = "reviewer"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	RealName string `json:"realName"`
	Phone    string `json:"phone"`
	Role     Role   `json:"role"`
	Status   int    `json:"status"`
}

type Session struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AcousticBuoy struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	BuoyClass        int     `json:"buoyClass"`
	CommissionedOn   *string `json:"commissionedOn"`
	SerialNumber     string  `json:"serialNumber"`
	ShoreStationCode string  `json:"shoreStationCode"`
	MooringZone      string  `json:"mooringZone"`
	Manufacturer     string  `json:"manufacturer"`
	EmergencyContact string  `json:"emergencyContact"`
	IntegrityStatus  int     `json:"integrityStatus"`
	Status           int     `json:"status"`
}

type ArrayOperator struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	SpecialtyLevel int       `json:"specialtyLevel"`
	Phone          string    `json:"phone"`
	Skills         string    `json:"skills"`
	Status         int       `json:"status"`
	CreateTime     time.Time `json:"createTime"`
}

type Treatment struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	RiskBudget      float64 `json:"riskBudget"`
	DurationMinutes int     `json:"durationMinutes"`
	Status          int     `json:"status"`
}

type WorkOrder struct {
	ID                string     `json:"id"`
	WorkOrderNo       string     `json:"workOrderNo"`
	AcousticBuoyID    string     `json:"acoustic_buoyId"`
	AcousticBuoyTitle string     `json:"acoustic_buoyTitle"`
	ArrayOpsProfileID string     `json:"treatmentId"`
	TreatmentName     string     `json:"treatmentName"`
	ArrayOperatorID   *string    `json:"array_operatorId"`
	ArrayOperatorName string     `json:"array_operatorName"`
	ScheduledAt       *time.Time `json:"scheduledAt"`
	Status            int        `json:"status"`
	Remark            string     `json:"remark"`
	Version           int64      `json:"version"`
}

type SignalRecoveryReport struct {
	ID                string    `json:"id"`
	AcousticBuoyID    string    `json:"acousticBuoyId"`
	AcousticBuoyTitle string    `json:"acousticBuoyTitle"`
	PacketLossPercent *float64  `json:"packetLossPercent"`
	WaterTemperatureC *float64  `json:"waterTemperatureC"`
	SignalNoiseDB     *float64  `json:"signalNoiseDb"`
	ClockDriftPPM     *float64  `json:"clockDriftPpm"`
	CorruptionIndex   *float64  `json:"corruptionIndex"`
	Remark            string    `json:"remark"`
	RecordedAt        time.Time `json:"recordedAt"`
}

type Log struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	Operation  string    `json:"operation"`
	Method     string    `json:"method"`
	IP         string    `json:"ip"`
	CreateTime time.Time `json:"createTime"`
}

type Page[T any] struct {
	Records []T `json:"records"`
	Total   int `json:"total"`
	Current int `json:"current"`
	Size    int `json:"size"`
}

type DashboardStats struct {
	AcousticBuoyCount   int `json:"acoustic_buoyCount"`
	ArrayOperatorCount  int `json:"array_operatorCount"`
	PendingWorkOrders   int `json:"pendingWorkOrders"`
	CompletedWorkOrders int `json:"completedWorkOrders"`
}
