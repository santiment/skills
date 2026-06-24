// Package exitcode defines the stable exit codes used by every CLI in the
// family and maps HTTP statuses and errors onto them. The contract is part of
// the machine-facing interface and must stay stable across releases.
package exitcode

import (
	"context"
	"errors"
	"net"

	"santiment.net/san-skills/internal/platform/apierr"
)

// Stable process exit codes. Documented in each tool's SKILL.md.
const (
	OK          = 0 // success
	Generic     = 1 // unclassified runtime error
	Usage       = 2 // bad flags or arguments (also used by cobra)
	Auth        = 3 // authentication/authorization failure (401/403)
	NotFound    = 4 // resource not found (404)
	RateLimited = 5 // rate limited (429)
	ServerError = 6 // upstream server error (5xx)
	Network     = 7 // network failure or timeout
)

// FromHTTPStatus maps an HTTP status code to a stable exit code.
func FromHTTPStatus(status int) int {
	switch {
	case status >= 200 && status < 300:
		return OK
	case status == 401 || status == 403:
		return Auth
	case status == 404:
		return NotFound
	case status == 429:
		return RateLimited
	case status >= 500:
		return ServerError
	case status >= 400:
		return Generic
	default:
		return Generic
	}
}

// Coder lets an error declare its own stable exit code.
type Coder interface{ ExitCode() int }

// FromError classifies an error into a stable exit code. It recognizes errors
// that declare their own code, normalized API errors, context
// cancellation/timeout, and network errors.
func FromError(err error) int {
	if err == nil {
		return OK
	}
	var coder Coder
	if errors.As(err, &coder) {
		return coder.ExitCode()
	}
	var apiErr *apierr.Error
	if errors.As(err, &apiErr) {
		return FromHTTPStatus(apiErr.HTTPStatus)
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return Network
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return Network
	}
	return Generic
}
