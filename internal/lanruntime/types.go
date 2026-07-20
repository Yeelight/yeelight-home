package lanruntime

import (
	"errors"
	"fmt"
	"time"

	"github.com/yeelight/yeelight-home/internal/lanmcp"
)

type ErrorKind string

const (
	ErrorPreCall     ErrorKind = "pre-call"
	ErrorUnsupported ErrorKind = "unsupported"
	ErrorRejected    ErrorKind = "rejected"
	ErrorUncertain   ErrorKind = "uncertain"
)

type Error struct {
	Kind    ErrorKind
	Stage   string
	Message string
	Cause   error
}

func (err *Error) Error() string {
	if err.Stage == "" {
		return err.Message
	}
	return fmt.Sprintf("%s: %s", err.Stage, err.Message)
}

func (err *Error) Unwrap() error { return err.Cause }

func KindOf(err error) ErrorKind {
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Kind
	}
	return ErrorPreCall
}

type Target struct {
	HouseID string
	Type    string
	ID      string
	Name    string
	Room    string
}

type PropertyRequest struct {
	RequestID string
	Target    Target
	Property  string
	Value     any
}

type PropertiesRequest struct {
	RequestID  string
	Target     Target
	Properties map[string]any
}

type BatchPropertyRequest struct {
	RequestID string
	Targets   []Target
	Property  string
	Value     any
}

type AdjustRequest struct {
	RequestID string
	Target    Target
	Property  string
	Delta     float64
	Min       float64
	Max       float64
}

type SceneRequest struct {
	RequestID string
	Target    Target
}

type ActionRequest struct {
	RequestID  string
	Target     Target
	ActionName string
	Payload    map[string]any
	Duration   any
	Delay      any
}

type FlowRequest struct {
	RequestID string
	Target    Target
	Flow      any
	Payload   map[string]any
	Duration  any
	Delay     any
}

type Outcome string

const (
	OutcomeApplied     Outcome = "applied"
	OutcomeNotApplied  Outcome = "not_applied"
	OutcomeUnverified  Outcome = "unverified"
	OutcomeUncertain   Outcome = "uncertain"
	OutcomeReadSuccess Outcome = "read_success"
)

type Result struct {
	Outcome       Outcome `json:"outcome"`
	Tool          string  `json:"tool"`
	Target        Target  `json:"target"`
	Property      string  `json:"property,omitempty"`
	ExpectedValue any     `json:"expectedValue,omitempty"`
	Value         any     `json:"value,omitempty"`
	Verified      bool    `json:"verified"`
	Evidence      string  `json:"evidence,omitempty"`
	Data          any     `json:"data,omitempty"`
	CallError     string  `json:"callError,omitempty"`
}

type Options struct {
	Client               *lanmcp.Client
	VerificationAttempts int
	VerificationInterval time.Duration
}
