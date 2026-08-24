package telemetry

import (
	"errors"
	"time"
)

var (
	ErrConflict     = errors.New("telemetry conflict")
	ErrInvalid      = errors.New("telemetry invalid")
	ErrUnavailable  = errors.New("telemetry unavailable")
	ErrUnauthorized = errors.New("telemetry unauthorized")
)

type Scope struct {
	TenantID  string
	MissionID string
	BuoyID    string
	CaptureID string
}

type Segment struct {
	Index   int
	Digest  string
	Payload []byte
}

type Stream struct {
	Key              string
	Scope            Scope
	ExpectedSegments int
	Segments         map[int]Segment
	Labels           map[string]string
	State            string
	Version          int64
}

type Record struct {
	StreamKey string
	Checksum  string
	Payload   []byte
	SealedAt  time.Time
}

type DecodeError struct {
	Codec string
	Cause error
}

func (e *DecodeError) Error() string { return "decode " + e.Codec + ": " + e.Cause.Error() }
func (e *DecodeError) Unwrap() error { return e.Cause }

type Decoder interface {
	Decode([]byte) ([]byte, error)
}

type DecoderFunc func([]byte) ([]byte, error)

func (f DecoderFunc) Decode(input []byte) ([]byte, error) { return f(input) }
