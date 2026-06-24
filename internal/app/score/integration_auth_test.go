//go:build integration

package score

import (
	"os"
	"testing"

	"santiment.net/san-skills/internal/platform/config"
	"santiment.net/san-skills/internal/platform/exitcode"
)

// With real credentials in the env, auth status must report both backends as
// authorized (proves the env credentials resolve into the runtime settings).
func TestLiveAuthStatusAuthorized(t *testing.T) {
	requireSanrAuth(t)
	requireArenaAuth(t)
	app, out := runLive(t, "auth", "status")
	assertOK(t, app, out)
	obj, ok := decodeObject(t, out)
	if !ok {
		t.Fatalf("status output not a JSON object: %s", out)
	}
	if obj["sanrAuthorized"] != true {
		t.Errorf("sanrAuthorized: want true, got %v", obj["sanrAuthorized"])
	}
	if obj["arenaAuthorized"] != true {
		t.Errorf("arenaAuthorized: want true, got %v", obj["arenaAuthorized"])
	}
}

// auth refresh exchanges the cached refresh token for a fresh session.
func TestLiveAuthRefresh(t *testing.T) {
	requireSanrAuth(t)
	if os.Getenv(config.EnvSanrRefresh) == "" {
		t.Skipf("set %s to run the refresh test", config.EnvSanrRefresh)
	}
	app, out := runLive(t, "auth", "refresh")
	assertOK(t, app, out)
}

// auth login with an invalid signature must be rejected by the backend (a 4xx),
// not succeed. Confirms the login request path reaches Sanr and surfaces the
// error through the stable exit-code contract.
func TestLiveAuthLoginBadSignature(t *testing.T) {
	app, out := runLive(t, "auth", "login",
		"--original-message", "score integration test",
		"--signed-message", "0xdeadbeef")
	if app.exitCode == exitcode.OK {
		t.Fatalf("invalid signature must not yield exit OK\noutput: %s", out)
	}
	t.Logf("login with bad signature produced exit code %d", app.exitCode)
}
