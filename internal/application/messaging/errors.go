package messaging

import "fmt"

type RetryableError struct {
	Cause error
}

func (e *RetryableError) Error() string {
	if e.Cause == nil {
		return "retryable error"
	}
	return fmt.Sprintf("retryable: %v", e.Cause)
}

func (e *RetryableError) Unwrap() error { return e.Cause }

func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Cause: err}
}

// DeferredError is a soft back-pressure signal (e.g. concurrent quota).
// The consumer must requeue with TTL and must NOT send the event to the DLQ
// when the normal retry budget is exhausted.
type DeferredError struct {
	Cause error
}

func (e *DeferredError) Error() string {
	if e.Cause == nil {
		return "deferred error"
	}
	return fmt.Sprintf("deferred: %v", e.Cause)
}

func (e *DeferredError) Unwrap() error { return e.Cause }

func Deferred(err error) error {
	if err == nil {
		return nil
	}
	return &DeferredError{Cause: err}
}

type NonRetryableError struct {
	Cause error
}

func (e *NonRetryableError) Error() string {
	if e.Cause == nil {
		return "non-retryable error"
	}
	return fmt.Sprintf("non-retryable: %v", e.Cause)
}

func (e *NonRetryableError) Unwrap() error { return e.Cause }

func NonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &NonRetryableError{Cause: err}
}
