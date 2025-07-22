package misc

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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

func RetriableErrorHandler(function func() error) error {
	var TryAgain *RetrialableError
	var pgErr *pgconn.PgError
	var err error = nil
	for i := 1; i <= 5; i += 2 {
		err = function()
		if err == nil {
			break
		}
		switch {
		case errors.As(err, &TryAgain):
			{
				log.Printf("error send metrics. error: %v.\n Attemp after %d seconds", err, i)
				time.Sleep(time.Duration(i * int(time.Second)))
			}
		case errors.As(err, &pgErr):
			{
				if pgerrcode.IsConnectionException(pgErr.Code) || pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
					time.Sleep(time.Duration(i * int(time.Second)))
				}
			}
		default:
			log.Printf("not retrirable error. error: %v.\n ", err)
			return err
		}
	}
	return err
}
