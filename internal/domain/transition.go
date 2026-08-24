package domain

import "fmt"

type Transition struct {
	From      string
	To        string
	Actor     string
	RequestID string
}

func (t Transition) Validate() error {
	if t.From == "" || t.To == "" || t.Actor == "" || t.RequestID == "" {
		return fmt.Errorf("transition metadata is incomplete: %w", ErrConflict)
	}
	if t.From == t.To {
		return fmt.Errorf("transition must change state: %w", ErrInvalidTransition)
	}
	return nil
}

func TransitionPath(start string, steps []string) error {
	previous := start
	for _, step := range steps {
		if step == "" || step == previous {
			return fmt.Errorf("invalid transition path: %w", ErrInvalidTransition)
		}
		previous = step
	}
	return nil
}
