//go:build integration

package score

import "testing"

// With a real token in the env, auth status must report authorized=true. A
// single token covers both backends, so this is the whole auth surface to check
// live (login/logout are exercised offline in auth_test.go).
func TestLiveAuthStatusAuthorized(t *testing.T) {
	requireSanrAuth(t)
	app, out := runLive(t, "auth", "status")
	assertOK(t, app, out)
	obj, ok := decodeObject(t, out)
	if !ok {
		t.Fatalf("status output not a JSON object: %s", out)
	}
	if obj["authorized"] != true {
		t.Errorf("authorized: want true, got %v", obj["authorized"])
	}
}
