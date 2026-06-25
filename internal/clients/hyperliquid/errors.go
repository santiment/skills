package hyperliquid

import "santiment.net/san-skills/internal/platform/exitcode"

// APIError is the base error for all Hyperliquid API failures. StatusCode is 0
// when the failure was not an HTTP status (timeout, decode, mapped "err"
// response).
type APIError struct {
	Message    string
	StatusCode int
	Response   any
}

func (e *APIError) Error() string { return e.Message }

// ExitCode maps the API error onto a stable process exit code. A status-bearing
// error maps by HTTP status; a status-less error (mapped "err" envelope, decode
// failure) is a generic runtime error.
func (e *APIError) ExitCode() int {
	if e.StatusCode == 0 {
		return exitcode.Generic
	}
	return exitcode.FromHTTPStatus(e.StatusCode)
}

// newAPIError builds an *APIError. status 0 means "no HTTP status".
func newAPIError(message string, status int, response any) *APIError {
	return &APIError{Message: message, StatusCode: status, Response: response}
}

// The following typed errors map the substring-matched HL error strings to Go
// types so callers can branch with errors.As. Each embeds *APIError and unwraps
// to it, so errors.As(err, &apiErr) also succeeds for the base type. Each also
// declares its own stable exit code via ExitCode (exitcode.Coder).

// RateLimitError signals an HTTP 429.
type RateLimitError struct{ *APIError }

// SignatureError signals an invalid-signature response.
type SignatureError struct{ *APIError }

// InsufficientMarginError signals an out-of-margin response.
type InsufficientMarginError struct{ *APIError }

// AssetNotFoundError signals an unknown asset/pair.
type AssetNotFoundError struct{ *APIError }

func (e *RateLimitError) Unwrap() error          { return e.APIError }
func (e *SignatureError) Unwrap() error          { return e.APIError }
func (e *InsufficientMarginError) Unwrap() error { return e.APIError }
func (e *AssetNotFoundError) Unwrap() error      { return e.APIError }

// ExitCode reports the stable exit code for a rate-limit failure.
func (e *RateLimitError) ExitCode() int { return exitcode.RateLimited }

// ExitCode reports the stable exit code for a signature/auth failure.
func (e *SignatureError) ExitCode() int { return exitcode.Auth }

// ExitCode reports the stable exit code for an insufficient-margin failure.
func (e *InsufficientMarginError) ExitCode() int { return exitcode.Generic }

// ExitCode reports the stable exit code for an unknown-asset failure.
func (e *AssetNotFoundError) ExitCode() int { return exitcode.NotFound }
