// Package exception defines the error types produced by gobatfish.
//
// pybatfish raises Python exceptions; Go code returns errors. The types here
// mirror pybatfish.exception so callers can discriminate with errors.Is and
// errors.As:
//
//	pybatfish.exception.BatfishException          -> *BatfishError
//	pybatfish.exception.BatfishAssertException    -> *BatfishAssertError
//	pybatfish.exception.QuestionValidationException -> *QuestionValidationError
//
// BatfishAssertWarning has no Go error counterpart because Python warnings are
// side effects; gobatfish's soft assertions log a warning and return nil.
package exception

import "fmt"

// BatfishError is the base error for Batfish-related failures. It corresponds
// to pybatfish.exception.BatfishException.
type BatfishError struct {
	// Message is the human readable error message.
	Message string
	// Err is an optional wrapped cause.
	Err error
}

// Error implements the error interface.
func (e *BatfishError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the wrapped cause, enabling errors.Is/As.
func (e *BatfishError) Unwrap() error { return e.Err }

// NewBatfishError creates a BatfishError with the given message.
func NewBatfishError(message string) *BatfishError {
	return &BatfishError{Message: message}
}

// NewBatfishErrorf creates a BatfishError with a formatted message.
func NewBatfishErrorf(format string, args ...any) *BatfishError {
	return &BatfishError{Message: fmt.Sprintf(format, args...)}
}

// WrapBatfishError wraps cause in a BatfishError with the given message.
func WrapBatfishError(message string, cause error) *BatfishError {
	return &BatfishError{Message: message, Err: cause}
}

// BatfishAssertError is returned when a pybatfish assertion fails. It
// corresponds to pybatfish.exception.BatfishAssertException.
type BatfishAssertError struct {
	Message string
	Err     error
}

// Error implements the error interface.
func (e *BatfishAssertError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the wrapped cause.
func (e *BatfishAssertError) Unwrap() error { return e.Err }

// NewBatfishAssertError creates a BatfishAssertError with the given message.
func NewBatfishAssertError(message string) *BatfishAssertError {
	return &BatfishAssertError{Message: message}
}

// NewBatfishAssertErrorf creates a BatfishAssertError with a formatted message.
func NewBatfishAssertErrorf(format string, args ...any) *BatfishAssertError {
	return &BatfishAssertError{Message: fmt.Sprintf(format, args...)}
}

// QuestionValidationError is returned when an invalid Batfish question is
// encountered. It corresponds to pybatfish.exception.QuestionValidationException.
type QuestionValidationError struct {
	Message string
	Err     error
}

// Error implements the error interface.
func (e *QuestionValidationError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the wrapped cause.
func (e *QuestionValidationError) Unwrap() error { return e.Err }

// NewQuestionValidationError creates a QuestionValidationError.
func NewQuestionValidationError(message string) *QuestionValidationError {
	return &QuestionValidationError{Message: message}
}

// NewQuestionValidationErrorf creates a QuestionValidationError with a formatted message.
func NewQuestionValidationErrorf(format string, args ...any) *QuestionValidationError {
	return &QuestionValidationError{Message: fmt.Sprintf(format, args...)}
}
