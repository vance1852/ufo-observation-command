package clock

import "time"

type Clock interface{ Now() time.Time }

type Real struct{}

func (Real) Now() time.Time { return time.Now().UTC() }

type Fixed struct{ Value time.Time }

func (f Fixed) Now() time.Time { return f.Value }

func UTC(value time.Time) time.Time { return value.UTC() }

func InWindow(now, start, end time.Time) bool {
	now, start, end = now.UTC(), start.UTC(), end.UTC()
	return !now.Before(start) && now.Before(end)
}

func DeadlinePassed(now, deadline time.Time) bool { return now.UTC().After(deadline.UTC()) }
