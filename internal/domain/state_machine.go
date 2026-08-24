package domain

import (
	"fmt"
	"sort"
)

type StateMachine struct {
	states map[string]map[string]struct{}
}

func NewStateMachine(edges map[string][]string) StateMachine {
	graph := make(map[string]map[string]struct{}, len(edges))
	for from, destinations := range edges {
		graph[from] = make(map[string]struct{}, len(destinations))
		for _, to := range destinations {
			graph[from][to] = struct{}{}
		}
	}
	return StateMachine{states: graph}
}

func (m StateMachine) Allows(from, to string) bool {
	destinations, ok := m.states[from]
	if !ok {
		return false
	}
	_, ok = destinations[to]
	return ok
}

func (m StateMachine) ValidatePath(path []string) error {
	if len(path) < 2 {
		return fmt.Errorf("state path needs at least two states: %w", ErrInvalidTransition)
	}
	for i := 1; i < len(path); i++ {
		if !m.Allows(path[i-1], path[i]) {
			return fmt.Errorf("%s -> %s is not allowed: %w", path[i-1], path[i], ErrInvalidTransition)
		}
	}
	return nil
}

func (m StateMachine) Reachable(start string) []string {
	visited := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for next := range m.states[current] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	result := make([]string, 0, len(visited))
	for state := range visited {
		result = append(result, state)
	}
	sort.Strings(result)
	return result
}

func DefaultSurveyMissionMachine() StateMachine {
	return NewStateMachine(map[string][]string{"draft": {"scheduled"}, "scheduled": {"active", "closed"}, "active": {"closed"}})
}

func DefaultRecoveryJobMachine() StateMachine {
	return NewStateMachine(map[string][]string{"queued": {"completed"}, "completed": {"activation_pending"}, "activation_pending": {"accepted"}, "accepted": {"in_progress"}, "in_progress": {"verified", "rejected"}, "verified": {"archived"}, "rejected": {"archived"}})
}

func DefaultDiveWindowMachine() StateMachine {
	return NewStateMachine(map[string][]string{"queued": {"running", "cancelled"}, "running": {"completed", "cancelled"}})
}
