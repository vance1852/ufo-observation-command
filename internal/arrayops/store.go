package arrayops

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

type Store struct {
	mu                  sync.Mutex
	calibration_bundles map[string]CalibrationBundle
	buoys               map[string]Buoy
	survey_missions     map[string]SurveyMission
	callbacks           map[string]Callback
	idempotency         map[string][]byte
	sessions            map[string]Session
	events              []Event
}

func NewStore() *Store {
	return &Store{
		calibration_bundles: make(map[string]CalibrationBundle),
		buoys:               make(map[string]Buoy),
		survey_missions:     make(map[string]SurveyMission),
		callbacks:           make(map[string]Callback),
		idempotency:         make(map[string][]byte),
		sessions:            make(map[string]Session),
	}
}

func (s *Store) PutCalibrationBundle(calibration_bundle CalibrationBundle) error {
	if calibration_bundle.ID == "" || calibration_bundle.TenantID == "" {
		return fmt.Errorf("calibration_bundle identity is missing: %w", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calibration_bundles[calibration_bundle.ID] = CloneCalibrationBundle(calibration_bundle)
	return nil
}

func (s *Store) CalibrationBundle(id string) (CalibrationBundle, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	calibration_bundle, ok := s.calibration_bundles[id]
	return CloneCalibrationBundle(calibration_bundle), ok
}

func (s *Store) PutBuoy(buoy Buoy) error {
	if buoy.ID == "" || buoy.TenantID == "" {
		return fmt.Errorf("buoy identity is missing: %w", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buoys[buoy.ID] = buoy
	return nil
}

func (s *Store) Buoy(id string) (Buoy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	buoy, ok := s.buoys[id]
	return buoy, ok
}

func (s *Store) PutSurveyMission(survey_mission SurveyMission) error {
	if survey_mission.ID == "" || survey_mission.TenantID == "" {
		return fmt.Errorf("survey_mission identity is missing: %w", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.survey_missions[survey_mission.ID] = CloneSurveyMission(survey_mission)
	return nil
}

func (s *Store) SurveyMission(id string) (SurveyMission, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	survey_mission, ok := s.survey_missions[id]
	return CloneSurveyMission(survey_mission), ok
}

func (s *Store) UpdateSurveyMission(id string, expected int64, update func(*SurveyMission) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	survey_mission, ok := s.survey_missions[id]
	if !ok {
		return fmt.Errorf("survey_mission not found: %w", ErrConflict)
	}
	if survey_mission.Version != expected {
		return fmt.Errorf("survey_mission version changed: %w", ErrConflict)
	}
	if err := update(&survey_mission); err != nil {
		return err
	}
	survey_mission.Version++
	s.survey_missions[id] = CloneSurveyMission(survey_mission)
	return nil
}

func (s *Store) RecordCallback(callback Callback) (bool, error) {
	key, err := CallbackKey(callback)
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.callbacks[key]; exists {
		return false, nil
	}
	s.callbacks[key] = callback
	return true, nil
}

func (s *Store) SaveIdempotent(key string, body []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.idempotency[key]; exists {
		return false
	}
	s.idempotency[key] = append([]byte(nil), body...)
	return true
}

func (s *Store) Idempotent(key string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.idempotency[key]
	return append([]byte(nil), value...), ok
}

func (s *Store) PutSession(session Session) error {
	if session.Token == "" || session.TenantID == "" || session.UserID == "" {
		return fmt.Errorf("session identity is missing: %w", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.Token] = session
	return nil
}

func (s *Store) Session(token string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[token]
	return session, ok
}

func (s *Store) RevokeSession(token string, at time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[token]
	if !ok || session.RevokedAt != nil {
		return false
	}
	session.RevokedAt = &at
	s.sessions[token] = session
	return true
}

func (s *Store) AppendEvent(event Event) error {
	if event.TenantID == "" || event.RequestID == "" || event.ObjectID == "" || event.Action == "" {
		return fmt.Errorf("audit correlation is incomplete: %w", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *Store) QueryBuoys(query Query) Page[Buoy] {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := make([]Buoy, 0)
	for _, buoy := range s.buoys {
		if buoy.TenantID != query.TenantID || query.Class != "" && buoy.Class != query.Class {
			continue
		}
		filtered = append(filtered, buoy)
	}
	slices.SortFunc(filtered, func(left, right Buoy) int { return compareString(left.ID, right.ID) })
	total := len(filtered)
	start := min(max(query.Offset, 0), total)
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	end := min(start+limit, total)
	return Page[Buoy]{Items: CloneBuoys(filtered[start:end]), Total: total}
}

func compareString(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
