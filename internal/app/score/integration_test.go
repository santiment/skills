//go:build integration

// Integration tests run against the live Sanr and Arena APIs.
// Run with: task test-integration  (or: go test -tags=integration ./...)
//
// They exercise only public, read-only endpoints so no credentials are needed.
// Endpoints that require auth are covered by the httptest-based unit tests.
package score

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"santiment.net/san-skills/internal/platform/exitcode"
	"santiment.net/san-skills/internal/platform/output"
)

func runLive(t *testing.T, args ...string) (*App, string) {
	t.Helper()
	app := &App{}
	var buf bytes.Buffer
	app.printer = &output.Printer{JSON: true, Out: &buf, Err: &buf}
	root := newRootCmd(app)
	root.SetArgs(append([]string{"--config", filepath.Join(t.TempDir(), "absent.yaml")}, args...))
	root.SetOut(&buf)
	root.SetErr(&buf)
	_ = root.Execute()
	return app, buf.String()
}

func TestLiveHealthBothUp(t *testing.T) {
	app, out := runLive(t, "health")
	if app.exitCode != exitcode.OK {
		t.Fatalf("health exit=%d, out=%s", app.exitCode, out)
	}
	if !strings.Contains(out, `"sanr"`) || !strings.Contains(out, `"arena"`) {
		t.Errorf("health output missing a backend: %s", out)
	}
}

func TestLiveMarketsList(t *testing.T) {
	app, out := runLive(t, "markets", "list", "--take", "1")
	if app.exitCode != exitcode.OK {
		t.Fatalf("markets list exit=%d, out=%s", app.exitCode, out)
	}
	if !strings.Contains(out, "data") {
		t.Errorf("markets list output unexpected: %s", out)
	}
}
