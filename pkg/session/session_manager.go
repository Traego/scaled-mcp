package session

import "errors"

type SessionManager interface {
	// GenerateSessionId creates a session id, using the principalId from the optional Auth handler
	GenerateSessionId(principalId *string) (string, error)
	VerifySessionId(sessionId string, checkPrincipalId *string) error

	CanSaveSession() bool
	SaveSession(sessionId string, sessionData interface{}) error
	LoadSession(sessionId string, checkUniqueId *string) (interface{}, error)
}

var (
	ErrSessionStateNotFound = errors.New("session state not found")
)

func NewErrSessionInvalid(reason string) *ErrSessionInvalid {
	return &ErrSessionInvalid{
		Reason: reason,
	}
}

// NewErrSessionInvalidWrap creates a new ErrSessionInvalid that wraps another error
func NewErrSessionInvalidWrap(reason string, wrapped error) *ErrSessionInvalid {
	return &ErrSessionInvalid{
		Reason:  reason,
		wrapped: wrapped,
	}
}

type ErrSessionInvalid struct {
	Reason  string
	wrapped error
}

func (e *ErrSessionInvalid) Error() string {
	if e.wrapped != nil {
		return "session invalid: " + e.Reason + ": " + e.wrapped.Error()
	}
	return "session invalid: " + e.Reason
}

// Unwrap returns the wrapped error, if any
func (e *ErrSessionInvalid) Unwrap() error {
	return e.wrapped
}
