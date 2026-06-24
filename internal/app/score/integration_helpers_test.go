//go:build integration

// Shared harness for the live integration tests. Every helper here runs the
// real `score` command tree against the configured backends (Sanr + Arena) and
// reports through the same buffer-backed printer the unit tests use.
//
// Credentials and write gates come from the environment so the suite degrades
// gracefully: tests skip (never fail) when a credential or opt-in flag is
// absent. See README / SKILL.md for the full list. Quick reference:
//
//	SANR_TOKEN, SANR_REFRESH_TOKEN   Sanr bearer session (authed reads, refresh)
//	ARENA_API_KEY                    Arena x-api-key (authed reads)
//	SCORE_TEST_WRITES=1              opt in to reversible write endpoints
//	SCORE_TEST_DESTRUCTIVE=1         opt in to financial/irreversible writes
//
// The CLI already injects SANR_TOKEN / SANR_REFRESH_TOKEN / ARENA_API_KEY via
// config.Resolve, so runLive needs no special wiring: setting the env var is
// enough for the request to carry the credential.
package score

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"santiment.net/san-skills/internal/platform/config"
	"santiment.net/san-skills/internal/platform/exitcode"
	"santiment.net/san-skills/internal/platform/output"
)

// runLive runs the command tree in --json mode against the live backends and
// returns the App (for exitCode) and the combined stdout+stderr.
func runLive(t *testing.T, args ...string) (*App, string) {
	t.Helper()
	return runLiveMode(t, true, args...)
}

// runLiveNoJSON runs the command tree in human (pretty-print) mode.
func runLiveNoJSON(t *testing.T, args ...string) (*App, string) {
	t.Helper()
	return runLiveMode(t, false, args...)
}

func runLiveMode(t *testing.T, jsonMode bool, args ...string) (*App, string) {
	t.Helper()
	app := &App{}
	var buf bytes.Buffer
	app.printer = &output.Printer{JSON: jsonMode, Out: &buf, Err: &buf}
	root := newRootCmd(app)
	// Force an absent config file so the suite never reads the developer's real
	// ~/.config/score profile; credentials still flow in via the env layer.
	root.SetArgs(append([]string{"--config", filepath.Join(t.TempDir(), "absent.yaml")}, args...))
	root.SetOut(&buf)
	root.SetErr(&buf)
	_ = root.Execute()
	return app, buf.String()
}

// requireSanrAuth skips the test unless a Sanr token is present in the env.
func requireSanrAuth(t *testing.T) {
	t.Helper()
	if os.Getenv(config.EnvSanrToken) == "" {
		t.Skipf("set %s to run Sanr authenticated tests", config.EnvSanrToken)
	}
}

// requireArenaAuth skips the test unless an Arena API key is present in the env.
func requireArenaAuth(t *testing.T) {
	t.Helper()
	if os.Getenv(config.EnvArenaAPIKey) == "" {
		t.Skipf("set %s to run Arena authenticated tests", config.EnvArenaAPIKey)
	}
}

// requireWrites gates reversible write tests behind an explicit opt-in.
func requireWrites(t *testing.T) {
	t.Helper()
	if os.Getenv("SCORE_TEST_WRITES") != "1" {
		t.Skip("set SCORE_TEST_WRITES=1 to run write tests against the test stand")
	}
}

// requireDestructive gates financial/irreversible write tests behind a second
// explicit opt-in (in addition to SCORE_TEST_WRITES).
func requireDestructive(t *testing.T) {
	t.Helper()
	requireWrites(t)
	if os.Getenv("SCORE_TEST_DESTRUCTIVE") != "1" {
		t.Skip("set SCORE_TEST_DESTRUCTIVE=1 to run financial/irreversible write tests")
	}
}

// envOrEmpty returns the named env var's value (empty string if unset).
func envOrEmpty(name string) string { return os.Getenv(name) }

// requireEnvBody returns the JSON body supplied via the named env var, or skips
// the test when it is absent. Write payloads are intentionally externalized so
// the suite never hard-codes a body that could be invalid for the test stand.
func requireEnvBody(t *testing.T, name string) string {
	t.Helper()
	body := os.Getenv(name)
	if body == "" {
		t.Skipf("set %s to a JSON body to run this write test", name)
	}
	return body
}

func assertOK(t *testing.T, app *App, out string) {
	t.Helper()
	assertExit(t, app, exitcode.OK, out)
}

func assertExit(t *testing.T, app *App, want int, out string) {
	t.Helper()
	if app.exitCode != want {
		t.Fatalf("exit code: want %d, got %d\noutput: %s", want, app.exitCode, out)
	}
}

// assertJSON fails unless out parses as a single JSON document.
func assertJSON(t *testing.T, out string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	return v
}

// knownEmptyAccountMsgs are backend error messages that mean "the token is
// valid but this account simply has no data for this endpoint" — e.g. a
// GDPR-cleared test account that no longer has an issuer/portfolio record. We
// treat them as an acceptable (skipped) outcome so the suite stays green across
// account states while still failing on genuine breakage (5xx, network, etc.).
var knownEmptyAccountMsgs = map[string]bool{
	"ISSUER_NOT_FOUND": true,
}

// assertAuthedRead accepts a successful authed read, or a recognized
// "no data for this account" business error (which it reports as a skip).
func assertAuthedRead(t *testing.T, app *App, out string) {
	t.Helper()
	if app.exitCode == exitcode.OK {
		assertJSON(t, out)
		return
	}
	if obj, ok := decodeObject(t, out); ok {
		if e, ok := obj["error"].(map[string]any); ok {
			if msg, _ := e["message"].(string); knownEmptyAccountMsgs[msg] {
				t.Skipf("token valid but account has no data here: %s", msg)
			}
		}
	}
	t.Fatalf("authed read failed: exit=%d\noutput: %s", app.exitCode, out)
}

// assertHasKey fails unless out is a JSON object containing key.
func assertHasKey(t *testing.T, out, key string) {
	t.Helper()
	obj, ok := assertJSON(t, out).(map[string]any)
	if !ok {
		t.Fatalf("expected a JSON object with key %q, got: %s", key, out)
	}
	if _, ok := obj[key]; !ok {
		t.Fatalf("JSON object missing key %q: %s", key, out)
	}
}

// firstListID extracts data[0].<field> from a {"data":[...]} list response as a
// string, returning ("", false) when the list is empty or shaped differently.
// Used to drive detail-endpoint tests with a real, currently-valid id.
func firstListID(out, field string) (string, bool) {
	var doc struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if json.Unmarshal([]byte(out), &doc) != nil || len(doc.Data) == 0 {
		return "", false
	}
	raw, ok := doc.Data[0][field]
	if !ok {
		return "", false
	}
	// Accept both string and numeric ids.
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, true
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String(), true
	}
	return "", false
}
