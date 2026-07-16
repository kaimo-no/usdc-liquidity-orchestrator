// Package errors defines stable string codes for the liquidity orchestrator.
package errors

import (
	"errors"
	"fmt"
)

// Stable codes (agent/HTTP clients branch on these).
const (
	CodeInvalidQuery             = "invalid_query"
	CodeInsufficientLiquidity    = "insufficient_liquidity"
	CodeLiquidityRailUnavailable = "liquidity_rail_unavailable"
)

// Error is a typed orchestrator error with a stable Code.
type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// New builds a coded error with a formatted message.
func New(code, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap attaches a cause under a stable code.
func Wrap(code string, err error, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...), Err: err}
}

// CodeOf returns the stable code if err is *Error, else "".
func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) && e != nil {
		return e.Code
	}
	return ""
}
