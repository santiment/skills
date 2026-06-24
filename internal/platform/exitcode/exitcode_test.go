package exitcode

import (
	"context"
	"errors"
	"testing"

	"santiment.net/san-skills/internal/platform/apierr"
)

func TestFromHTTPStatus(t *testing.T) {
	cases := map[int]int{
		200: OK, 204: OK,
		401: Auth, 403: Auth,
		404: NotFound,
		429: RateLimited,
		500: ServerError, 503: ServerError,
		400: Generic, 418: Generic,
	}
	for status, want := range cases {
		if got := FromHTTPStatus(status); got != want {
			t.Errorf("status %d: want %d, got %d", status, want, got)
		}
	}
}

func TestFromErrorAPIError(t *testing.T) {
	err := &apierr.Error{Backend: "sanr", HTTPStatus: 404}
	if got := FromError(err); got != NotFound {
		t.Errorf("want NotFound, got %d", got)
	}
}

func TestFromErrorContextDeadline(t *testing.T) {
	if got := FromError(context.DeadlineExceeded); got != Network {
		t.Errorf("want Network, got %d", got)
	}
}

type coderErr struct{}

func (coderErr) Error() string { return "boom" }
func (coderErr) ExitCode() int { return Usage }

func TestFromErrorCoder(t *testing.T) {
	if got := FromError(coderErr{}); got != Usage {
		t.Errorf("want Usage, got %d", got)
	}
}

func TestFromErrorNil(t *testing.T) {
	if got := FromError(nil); got != OK {
		t.Errorf("want OK, got %d", got)
	}
}

func TestFromErrorGeneric(t *testing.T) {
	if got := FromError(errors.New("plain")); got != Generic {
		t.Errorf("want Generic, got %d", got)
	}
}
