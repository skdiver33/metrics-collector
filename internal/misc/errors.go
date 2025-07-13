package misc

import (
	"fmt"
)

type RetrialableError struct {
	Err error
}

func NewRetrialableError(err error) *RetrialableError {
	wrapErr := fmt.Errorf("retrialable error - %w", err)
	return &RetrialableError{Err: wrapErr}
}

func (retError RetrialableError) Error() string {
	return retError.Err.Error()
}

func (retError RetrialableError) Unwrap() error {
	return retError.Err
}
