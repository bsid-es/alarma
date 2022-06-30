package alarma

import (
	"errors"
	"fmt"
)

type errorCode string

const (
	ErrInternal errorCode = "internal"
	ErrInvalid  errorCode = "invalid"
)

// Error is an application error.
type Error struct {
	// Code is a machine-readable error code.
	Code errorCode

	// Description is a human-readable description of the error.
	Description string
}

// Error implements the error interface.
func (e *Error) Error() string {
	return "alarma: " + string(e.Code) + ": " + e.Description
}

func Errorf(code errorCode, format string, args ...any) error {
	return &Error{code, fmt.Sprintf(format, args...)}
}

// ErrorCode returns the error code associated with err, or ErrInternal if err
// isn't an application error.
func ErrorCode(err error) errorCode {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) && e.Code != "" {
		return e.Code
	}
	return ErrInternal
}

// ErrorDescription returns a human-readable description of the error, or
// "internal error" if err isn't an application error.
func ErrorDescription(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) && e.Description != "" {
		return e.Description
	}
	return "internal error"
}
