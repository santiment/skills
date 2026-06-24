//go:build integration

// Integration tests run against the live Sanr and Arena APIs.
// Run with: task test-integration  (or: go test -tags=integration ./...)
//
// Public read-only endpoints run with no credentials. Authenticated reads,
// writes, and auth commands are covered by the sibling integration_*_test.go
// files and gate themselves on environment credentials / opt-in flags (see
// integration_helpers_test.go for the harness and the env-var reference).
package score

import (
	"strings"
	"testing"
)

func TestLiveHealthBothUp(t *testing.T) {
	app, out := runLive(t, "health")
	assertOK(t, app, out)
	if !strings.Contains(out, `"sanr"`) || !strings.Contains(out, `"arena"`) {
		t.Errorf("health output missing a backend: %s", out)
	}
}

func TestLiveMarketsList(t *testing.T) {
	app, out := runLive(t, "markets", "list", "--take", "1")
	assertOK(t, app, out)
	assertHasKey(t, out, "data")
}
