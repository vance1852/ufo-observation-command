package service

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
	"github.com/google/uuid"
)

type Clock func() time.Time

type Service struct {
	repo repository.Repository
	now  Clock
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) WithClock(clock Clock) *Service {
	if clock != nil {
		s.now = clock
	}
	return s
}

type RequestMeta struct {
	RequestID       string
	ArrayOperatorID *string
}

type CreateSurveyMissionRequest struct {
	Code          string
	Name          string
	Timezone      string
	StartsAt      time.Time
	EndsAt        time.Time
	CreatedBy     string
	AcousticBuoys []repository.AcousticBuoyInput
}

type CreateSurveyMissionResponse struct {
	SurveyMission   domain.SurveyMission `json:"survey_mission"`
	AcousticBuoyIDs []string             `json:"acoustic_buoy_ids"`
}

func (s *Service) CreateSurveyMission(ctx context.Context, meta RequestMeta, in CreateSurveyMissionRequest) (CreateSurveyMissionResponse, error) {
	if in.Code == "" || in.Name == "" || in.CreatedBy == "" || len(in.AcousticBuoys) == 0 {
		return CreateSurveyMissionResponse{}, fmt.Errorf("code, name, creator and acoustic_buoys are required: %w", domain.ErrConflict)
	}
	if err := domain.ValidateBusinessCode(in.Code); err != nil {
		return CreateSurveyMissionResponse{}, err
	}
	if err := validateCode(in.Name, "survey_mission name"); err != nil {
		return CreateSurveyMissionResponse{}, err
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return CreateSurveyMissionResponse{}, fmt.Errorf("invalid survey_mission timezone: %w", domain.ErrConflict)
	}
	seenAcousticBuoys := make(map[string]struct{}, len(in.AcousticBuoys))
	for _, acoustic_buoy := range in.AcousticBuoys {
		if err := (domain.AcousticBuoy{Code: acoustic_buoy.Code, ArrayOpsLane: acoustic_buoy.ArrayOpsLane, RequiredSuccesses: acoustic_buoy.RequiredSuccesses}).Validate(); err != nil {
			return CreateSurveyMissionResponse{}, err
		}
		if err := domain.ValidateBusinessCode(acoustic_buoy.Code); err != nil {
			return CreateSurveyMissionResponse{}, err
		}
		if _, exists := seenAcousticBuoys[acoustic_buoy.Code]; exists {
			return CreateSurveyMissionResponse{}, fmt.Errorf("duplicate acoustic_buoy code %s: %w", acoustic_buoy.Code, domain.ErrConflict)
		}
		seenAcousticBuoys[acoustic_buoy.Code] = struct{}{}
	}
	survey_mission := domain.SurveyMission{ID: uuid.NewString(), Code: in.Code, Name: in.Name, Status: domain.SurveyMissionDraft, Timezone: in.Timezone, StartsAt: in.StartsAt, EndsAt: in.EndsAt, Version: 1, CreatedBy: in.CreatedBy}
	if err := survey_mission.ValidateWindow(s.now()); err != nil {
		return CreateSurveyMissionResponse{}, err
	}
	response := CreateSurveyMissionResponse{SurveyMission: survey_mission, AcousticBuoyIDs: make([]string, 0, len(in.AcousticBuoys))}
	err := s.repo.InTx(ctx, func(tx repository.Repository) error {
		if err := tx.CreateSurveyMission(ctx, &survey_mission); err != nil {
			return err
		}
		for _, acoustic_buoy := range in.AcousticBuoys {
			acoustic_buoy.SurveyMissionID = survey_mission.ID
			id, err := tx.CreateAcousticBuoy(ctx, acoustic_buoy)
			if err != nil {
				return err
			}
			response.AcousticBuoyIDs = append(response.AcousticBuoyIDs, id)
		}
		return tx.WriteAudit(ctx, audit(meta, "acoustic_buoy_survey_mission", survey_mission.ID, "create", "success", nil))
	})
	if err != nil {
		return CreateSurveyMissionResponse{}, err
	}
	return response, nil
}

func (s *Service) ScheduleSurveyMission(ctx context.Context, meta RequestMeta, id string, version int64) error {
	return s.advanceSurveyMission(ctx, meta, id, domain.SurveyMissionScheduled, version, "schedule")
}

func (s *Service) ActivateSurveyMission(ctx context.Context, meta RequestMeta, id string, version int64) error {
	return s.advanceSurveyMission(ctx, meta, id, domain.SurveyMissionActive, version, "activate")
}

func (s *Service) CloseSurveyMission(ctx context.Context, meta RequestMeta, id string, version int64) error {
	return s.advanceSurveyMission(ctx, meta, id, domain.SurveyMissionClosed, version, "close")
}

func (s *Service) advanceSurveyMission(ctx context.Context, meta RequestMeta, id string, next domain.SurveyMissionStatus, version int64, action string) error {
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		survey_mission, err := tx.GetSurveyMission(ctx, id)
		if err != nil {
			return err
		}
		if !survey_mission.Status.CanMoveTo(next) {
			return fmt.Errorf("survey_mission %s: %w", id, domain.ErrInvalidTransition)
		}
		if err := tx.AdvanceSurveyMission(ctx, id, next, version); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "acoustic_buoy_survey_mission", id, action, "success", nil))
	})
}

func (s *Service) CreateRecoveryJob(ctx context.Context, meta RequestMeta, in repository.RecoveryJobInput) (domain.RecoveryJob, error) {
	request := domain.RecoveryJobRequest{SurveyMissionID: in.SurveyMissionID, AcousticBuoyID: in.AcousticBuoyID, TaskCode: in.TaskCode, ExpiresAt: in.ExpiresAt}
	if err := request.Validate(s.now()); err != nil {
		return domain.RecoveryJob{}, err
	}
	if err := domain.ValidateBusinessCode(in.TaskCode); err != nil {
		return domain.RecoveryJob{}, err
	}
	var task domain.RecoveryJob
	err := s.repo.InTx(ctx, func(tx repository.Repository) error {
		placement, ok := tx.(interface {
			ValidateSurveyMissionAcousticBuoy(context.Context, string, string) error
		})
		if !ok {
			return fmt.Errorf("task placement repository unavailable")
		}
		if err := placement.ValidateSurveyMissionAcousticBuoy(ctx, in.SurveyMissionID, in.AcousticBuoyID); err != nil {
			return err
		}
		var err error
		task, err = tx.CreateRecoveryJob(ctx, in)
		if err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "acoustic_buoy_task", task.ID, "create", "success", nil))
	})
	return task, err
}

func (s *Service) CompleteRecoveryJob(ctx context.Context, meta RequestMeta, id string, version int64) error {
	return s.moveRecoveryJob(ctx, meta, id, version, domain.RecoveryJobCompleted, "complete")
}

func (s *Service) ActivationRecoveryJob(ctx context.Context, meta RequestMeta, in repository.ActivationInput, version int64) error {
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		if err := tx.MoveRecoveryJob(ctx, in.RecoveryJobID, domain.RecoveryJobActivationPending, version, in.RecordedAt); err != nil {
			return err
		}
		if err := tx.RecordActivation(ctx, in); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "acoustic_buoy_task", in.RecoveryJobID, "activation", "success", nil))
	})
}

func (s *Service) AcceptRecoveryJob(ctx context.Context, meta RequestMeta, in repository.ActivationInput, version int64) error {
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		if err := tx.MoveRecoveryJob(ctx, in.RecoveryJobID, domain.RecoveryJobAccepted, version, in.RecordedAt); err != nil {
			return err
		}
		if err := tx.RecordActivation(ctx, in); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "acoustic_buoy_task", in.RecoveryJobID, "accept", "success", nil))
	})
}

func (s *Service) CreateDiveWindow(ctx context.Context, meta RequestMeta, in repository.DiveWindowInput, taskIDs []string) (string, error) {
	if len(taskIDs) == 0 || len(taskIDs) > in.Capacity {
		return "", domain.ErrCapacityExceeded
	}
	if err := (domain.DiveWindow{Code: in.Code, Method: in.Method, Capacity: in.Capacity, RecoveryJobs: taskIDs}).Validate(); err != nil {
		return "", err
	}
	if err := domain.ValidateBusinessCode(in.Code); err != nil {
		return "", err
	}
	if err := validateIDs(taskIDs); err != nil {
		return "", err
	}
	var id string
	err := s.repo.InTx(ctx, func(tx repository.Repository) error {
		var err error
		id, err = tx.CreateDiveWindow(ctx, in)
		if err != nil {
			return err
		}
		if err := tx.AttachRecoveryJobs(ctx, id, append([]string(nil), taskIDs...)); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "dive_window", id, "create", "success", nil))
	})
	return id, err
}

func (s *Service) SubmitSignalRecoveryReport(ctx context.Context, meta RequestMeta, in repository.SignalRecoveryReportInput) (string, error) {
	if in.RecoveryJobID == "" || in.DiveWindowID == "" || in.RecorderID == "" || in.ObservedAt.IsZero() {
		return "", fmt.Errorf("signal_recovery_report identifiers and observed_at are required: %w", domain.ErrConflict)
	}
	if err := domain.ValidateSignalRecoveryReport(in.RiskScore, in.AlertThreshold, in.Scale); err != nil {
		return "", err
	}
	var id string
	err := s.repo.InTx(ctx, func(tx repository.Repository) error {
		target, ok := tx.(interface {
			ValidateSignalRecoveryReportTarget(context.Context, string, string) error
			GetArrayOperator(context.Context, string) (domain.ArrayOperator, error)
		})
		if !ok {
			return fmt.Errorf("signal_recovery_report target repository unavailable")
		}
		if err := target.ValidateSignalRecoveryReportTarget(ctx, in.RecoveryJobID, in.DiveWindowID); err != nil {
			return err
		}
		recorder, err := target.GetArrayOperator(ctx, in.RecorderID)
		if err != nil {
			return err
		}
		if !recorder.CanRecordSignalRecoveryReport() {
			return fmt.Errorf("array_operator role %s cannot record signal_recovery_reports: %w", recorder.Role, domain.ErrConflict)
		}
		id, err = tx.CreateSignalRecoveryReport(ctx, in)
		if err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "signal_recovery_report", id, "record", "success", nil))
	})
	return id, err
}

func (s *Service) ReviewSignalRecoveryReport(ctx context.Context, meta RequestMeta, signal_recovery_reportID, taskID string, accepted bool, signal_recovery_reportVersion, taskVersion int64) error {
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		signal_recovery_report, ok := tx.(interface {
			SignalRecoveryReportTaskID(context.Context, string) (string, error)
		})
		if !ok {
			return fmt.Errorf("signal_recovery_report query repository unavailable")
		}
		actualRecoveryJobID, err := signal_recovery_report.SignalRecoveryReportTaskID(ctx, signal_recovery_reportID)
		if err != nil {
			return err
		}
		if actualRecoveryJobID != taskID {
			return fmt.Errorf("signal_recovery_report and task do not match: %w", domain.ErrConflict)
		}
		if err := tx.ReviewSignalRecoveryReportRecord(ctx, signal_recovery_reportID, accepted, signal_recovery_reportVersion, s.now()); err != nil {
			return err
		}
		next := domain.RecoveryJobVerified
		if !accepted {
			next = domain.RecoveryJobRejected
		}
		if err := tx.MoveRecoveryJob(ctx, taskID, next, taskVersion, s.now()); err != nil {
			return err
		}
		if !accepted {
			safety_alertID, err := tx.CreateIntegrityIncident(ctx, repository.IntegrityIncidentInput{RecoveryJobID: taskID, Kind: "reassess", Reason: "risk score exceeded the alert threshold", DueAt: s.now().Add(72 * time.Hour)})
			if err != nil {
				return err
			}
			if err := tx.WriteAudit(ctx, audit(meta, "safety_alert", safety_alertID, "open", "success", nil)); err != nil {
				return err
			}
		}
		return tx.WriteAudit(ctx, audit(meta, "signal_recovery_report", signal_recovery_reportID, "review", "success", nil))
	})
}

func (s *Service) ArchiveRecoveryJob(ctx context.Context, meta RequestMeta, taskID string, version int64) error {
	return s.moveRecoveryJob(ctx, meta, taskID, version, domain.RecoveryJobArchived, "archive")
}

func (s *Service) ListRecoveryJobs(ctx context.Context, offset, limit int, survey_missionID string, status domain.RecoveryJobStatus) (repository.Page, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListRecoveryJobs(ctx, offset, limit, survey_missionID, status)
}

func (s *Service) DueIntegrityIncidents(ctx context.Context, before time.Time, limit int) ([]repository.IntegrityIncidentInput, error) {
	return s.repo.DueIntegrityIncidents(ctx, before, limit)
}

func (s *Service) moveRecoveryJob(ctx context.Context, meta RequestMeta, id string, version int64, next domain.RecoveryJobStatus, action string) error {
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		task, err := tx.GetRecoveryJob(ctx, id)
		if err != nil {
			return err
		}
		if next == domain.RecoveryJobCompleted {
			survey_mission, err := tx.GetSurveyMission(ctx, task.SurveyMissionID)
			if err != nil {
				return err
			}
			if !survey_mission.CanExecuteAt(s.now()) {
				return fmt.Errorf("survey_mission is not active for task execution: %w", domain.ErrInvalidTransition)
			}
		}
		updated, err := task.Move(next, s.now())
		if err != nil {
			return err
		}
		if err := tx.MoveRecoveryJob(ctx, id, updated.Status, version, s.now()); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "acoustic_buoy_task", id, action, "success", nil))
	})
}

func audit(meta RequestMeta, objectType, objectID, action, outcome string, detail []byte) repository.AuditInput {
	return repository.AuditInput{RequestID: meta.RequestID, ArrayOperatorID: meta.ArrayOperatorID, ObjectType: objectType, ObjectID: objectID, Action: action, Outcome: outcome, Detail: detail}
}
