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

// auth set-token persists a JWT into the profile, and auth status reads it back
// (masked). Exercises the real config.SaveTokens round-trip without network.
func TestAuthSetTokenRoundTrip(t *testing.T) {
	// Clear env credentials so the file layer is what status reflects.
	t.Setenv(config.EnvSanrToken, "")
	t.Setenv(config.EnvSanrRefresh, "")
	t.Setenv(config.EnvArenaAPIKey, "")

	cfg := filepath.Join(t.TempDir(), "config.yaml")

	app, out := runCmd(t, "--config", cfg, "auth", "set-token", "--token", "jwt-abcdef", "--refresh", "ref-xyz")
	if app.exitCode != exitcode.OK {
		t.Fatalf("set-token exit=%d, out=%s", app.exitCode, out)
	}

	app, out = runCmd(t, "--config", cfg, "auth", "status")
	if app.exitCode != exitcode.OK {
		t.Fatalf("status exit=%d, out=%s", app.exitCode, out)
	}
	obj, ok := decodeObject(t, out)
	if !ok {
		t.Fatalf("status output not a JSON object: %s", out)
	}
	if obj["sanrAuthorized"] != true {
		t.Errorf("sanrAuthorized: want true, got %v", obj["sanrAuthorized"])
	}
	if tok, _ := obj["sanrToken"].(string); tok == "" || tok == "jwt-abcdef" {
		t.Errorf("sanrToken should be present and masked, got %q", tok)
	}
}

func TestAuthSetTokenMissingTokenIsUsageError(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	app, out := runCmd(t, "--config", cfg, "auth", "set-token")
	if app.exitCode != exitcode.Usage {
		t.Errorf("missing --token must be usage error (2), got %d\nout=%s", app.exitCode, out)
	}
}

func TestAuthLoginMissingFlagsIsUsageError(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	app, out := runCmd(t, "--config", cfg, "auth", "login")
	if app.exitCode != exitcode.Usage {
		t.Errorf("missing login flags must be usage error (2), got %d\nout=%s", app.exitCode, out)
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
	for _, key := range []string{"profile", "config", "sanrBaseURL", "arenaBaseURL", "sanrAuthorized", "arenaAuthorized"} {
		if _, ok := obj[key]; !ok {
			t.Errorf("status missing key %q: %s", key, out)
		}
	}
}
