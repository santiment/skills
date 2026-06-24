package score

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"santiment.net/san-skills/internal/platform/config"
	"santiment.net/san-skills/internal/platform/exitcode"
	"santiment.net/san-skills/internal/platform/output"
)

// decodeObject parses out as a JSON object. Defined here (no build tag) so both
// the offline and integration test files can use it.
func decodeObject(t *testing.T, out string) (map[string]any, bool) {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		return nil, false
	}
	obj, ok := v.(map[string]any)
	return obj, ok
}

// runCmd runs the command tree with a buffer-backed JSON printer and the exact
// args given (no forced overrides), for offline command-surface tests.
func runCmd(t *testing.T, args ...string) (*App, string) {
	t.Helper()
	app := &App{}
	var buf bytes.Buffer
	app.printer = &output.Printer{JSON: true, Out: &buf, Err: &buf}
	root := newRootCmd(app)
	root.SetArgs(args)
	root.SetOut(&buf)
	root.SetErr(&buf)
	_ = root.Execute()
	return app, buf.String()
}

// auth login persists the API token into the profile, and auth status reads it
// back (masked, authorized=true). One token authenticates both backends, so a
// single login is enough. Exercises the real config.SaveTokens round-trip.
func TestAuthLoginRoundTrip(t *testing.T) {
	// Clear env credentials so the file layer is what status reflects.
	t.Setenv(config.EnvSanrToken, "")
	t.Setenv(config.EnvArenaAPIKey, "")

	cfg := filepath.Join(t.TempDir(), "config.yaml")

	// Leading "Bearer " and whitespace must be tolerated.
	app, out := runCmd(t, "--config", cfg, "auth", "login", "--token", "  Bearer jwt-abcdef  ")
	if app.exitCode != exitcode.OK {
		t.Fatalf("login exit=%d, out=%s", app.exitCode, out)
	}

	app, out = runCmd(t, "--config", cfg, "auth", "status")
	if app.exitCode != exitcode.OK {
		t.Fatalf("status exit=%d, out=%s", app.exitCode, out)
	}
	obj, ok := decodeObject(t, out)
	if !ok {
		t.Fatalf("status output not a JSON object: %s", out)
	}
	if obj["authorized"] != true {
		t.Errorf("authorized: want true, got %v", obj["authorized"])
	}
	if tok, _ := obj["token"].(string); tok == "" || tok == "jwt-abcdef" {
		t.Errorf("token should be present and masked, got %q", tok)
	}
}

// logout clears the cached token: status then reports authorized=false.
func TestAuthLogout(t *testing.T) {
	t.Setenv(config.EnvSanrToken, "")
	t.Setenv(config.EnvArenaAPIKey, "")
	cfg := filepath.Join(t.TempDir(), "config.yaml")

	runCmd(t, "--config", cfg, "auth", "login", "--token", "jwt-abcdef")
	app, out := runCmd(t, "--config", cfg, "auth", "logout")
	if app.exitCode != exitcode.OK {
		t.Fatalf("logout exit=%d, out=%s", app.exitCode, out)
	}
	_, out = runCmd(t, "--config", cfg, "auth", "status")
	if obj, _ := decodeObject(t, out); obj["authorized"] != false {
		t.Errorf("after logout authorized must be false, got %v", obj["authorized"])
	}
}

func TestAuthLoginMissingTokenIsUsageError(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	app, out := runCmd(t, "--config", cfg, "auth", "login")
	if app.exitCode != exitcode.Usage {
		t.Errorf("missing --token must be usage error (2), got %d\nout=%s", app.exitCode, out)
	}
}

func TestAuthStatusShape(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "absent.yaml")
	app, out := runCmd(t, "--config", cfg, "auth", "status")
	if app.exitCode != exitcode.OK {
		t.Fatalf("status exit=%d, out=%s", app.exitCode, out)
	}
	obj, ok := decodeObject(t, out)
	if !ok {
		t.Fatalf("status output not a JSON object: %s", out)
	}
	for _, key := range []string{"profile", "config", "sanrBaseURL", "arenaBaseURL", "token", "authorized"} {
		if _, ok := obj[key]; !ok {
			t.Errorf("status missing key %q: %s", key, out)
		}
	}
}
