//go:build integration

package score

import (
	"testing"

	"santiment.net/san-skills/internal/platform/exitcode"
)

// The stable exit-code contract, exercised against real backend responses.

// A non-existent prediction id must map to NotFound (4). Gated on auth so a
// missing token can't surface as 401 and mask the 404 we want to observe.
func TestLiveExitNotFound(t *testing.T) {
	requireSanrAuth(t)
	app, out := runLive(t, "predictions", "get", "999999999")
	assertExit(t, app, exitcode.NotFound, out)
}

// A bogus bearer token on an authenticated endpoint must map to Auth (3).
// Needs no real credentials: the deliberately-invalid token is what we test.
func TestLiveExitAuth(t *testing.T) {
	app, out := runLive(t, "--token", "bogus-token", "profile", "get")
	assertExit(t, app, exitcode.Auth, out)
}

// A non-integer path id is rejected locally as a usage error (2), no network.
func TestLiveExitUsageBadID(t *testing.T) {
	app, out := runLive(t, "predictions", "get", "not-an-int")
	assertExit(t, app, exitcode.Usage, out)
}

// A malformed --param (missing '=') is a usage error (2), no network.
func TestLiveExitUsageBadParam(t *testing.T) {
	app, out := runLive(t, "predictions", "list", "--param", "no-equals-sign")
	assertExit(t, app, exitcode.Usage, out)
}
