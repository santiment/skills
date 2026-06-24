//go:build integration

// Live write tests. These mutate data on the test stand, so they are gated:
//   - SCORE_TEST_WRITES=1        enables the reversible writes below
//   - SCORE_TEST_DESTRUCTIVE=1   additionally enables financial/irreversible ones
//
// Request bodies are supplied via environment variables (not hard-coded) so the
// payloads always match the test stand's current schema and the suite never
// invents a body that could corrupt data. Each test skips when its body var is
// unset. `profile gdpr` is deliberately never invoked — see TestLiveGdprNotInvoked.
package score

import (
	"encoding/json"
	"testing"
)

// objID extracts <field> from a response object, checking both the top level
// and a nested "data" object: {"id":..} or {"data":{"id":..}}.
func objID(out string, fields ...string) (string, bool) {
	var top map[string]json.RawMessage
	if json.Unmarshal([]byte(out), &top) != nil {
		return "", false
	}
	scopes := []map[string]json.RawMessage{top}
	if nested, ok := top["data"]; ok {
		var inner map[string]json.RawMessage
		if json.Unmarshal(nested, &inner) == nil {
			scopes = append(scopes, inner)
		}
	}
	for _, scope := range scopes {
		for _, f := range fields {
			raw, ok := scope[f]
			if !ok {
				continue
			}
			var s string
			if json.Unmarshal(raw, &s) == nil {
				return s, true
			}
			var n json.Number
			if json.Unmarshal(raw, &n) == nil {
				return n.String(), true
			}
		}
	}
	return "", false
}

// profile update with a caller-supplied (ideally no-op) body.
func TestLiveWriteProfileUpdate(t *testing.T) {
	requireWrites(t)
	requireSanrAuth(t)
	body := requireEnvBody(t, "SCORE_TEST_PROFILE_BODY")
	app, out := runLive(t, "profile", "update", "--data", body)
	assertOK(t, app, out)
}

// portfolio distribution-save with a caller-supplied body.
func TestLiveWriteDistributionSave(t *testing.T) {
	requireWrites(t)
	requireSanrAuth(t)
	body := requireEnvBody(t, "SCORE_TEST_DISTRIBUTION_BODY")
	app, out := runLive(t, "portfolio", "distribution-save", "--data", body)
	assertOK(t, app, out)
}

// Prediction lifecycle: create -> get (verify) -> optional update -> close.
func TestLiveWritePredictionLifecycle(t *testing.T) {
	requireWrites(t)
	requireSanrAuth(t)
	body := requireEnvBody(t, "SCORE_TEST_PREDICTION_BODY")

	app, out := runLive(t, "predictions", "create", "--data", body)
	assertOK(t, app, out)

	id, ok := objID(out, "id")
	if !ok {
		t.Fatalf("could not extract created prediction id from: %s", out)
	}
	t.Logf("created prediction id=%s", id)

	// Verify it is now readable.
	getApp, getOut := runLive(t, "predictions", "get", id)
	assertOK(t, getApp, getOut)

	// Optional in-place update.
	if upd := envOrEmpty("SCORE_TEST_PREDICTION_UPDATE_BODY"); upd != "" {
		uApp, uOut := runLive(t, "predictions", "update", id, "--data", upd)
		assertOK(t, uApp, uOut)
	}

	// Close it to clean up the lifecycle.
	cApp, cOut := runLive(t, "predictions", "close", id)
	assertOK(t, cApp, cOut)
}

// --- Destructive / financial writes (second opt-in) ---

// order-place then order-cancel, each with a caller-supplied body.
func TestLiveWriteOrderPlaceCancel(t *testing.T) {
	requireDestructive(t)
	requireSanrAuth(t)
	placeBody := requireEnvBody(t, "SCORE_TEST_ORDER_BODY")
	cancelBody := requireEnvBody(t, "SCORE_TEST_ORDER_CANCEL_BODY")

	app, out := runLive(t, "portfolio", "order-place", "--data", placeBody)
	assertOK(t, app, out)

	cApp, cOut := runLive(t, "portfolio", "order-cancel", "--data", cancelBody)
	assertOK(t, cApp, cOut)
}

func TestLiveWriteDeposit(t *testing.T) {
	requireDestructive(t)
	requireSanrAuth(t)
	body := requireEnvBody(t, "SCORE_TEST_DEPOSIT_BODY")
	app, out := runLive(t, "portfolio", "deposit", "--data", body)
	assertOK(t, app, out)
}

func TestLiveWriteWithdrawal(t *testing.T) {
	requireDestructive(t)
	requireSanrAuth(t)
	body := requireEnvBody(t, "SCORE_TEST_WITHDRAWAL_BODY")
	app, out := runLive(t, "portfolio", "withdrawal", "--data", body)
	assertOK(t, app, out)
}

func TestLiveWriteRecalcHistorical(t *testing.T) {
	requireDestructive(t)
	requireSanrAuth(t)
	body := requireEnvBody(t, "SCORE_TEST_RECALC_BODY")
	app, out := runLive(t, "portfolio", "recalc-historical", "--data", body)
	assertOK(t, app, out)
}

func TestLiveWriteDisableNotifications(t *testing.T) {
	requireDestructive(t)
	requireSanrAuth(t)
	body := requireEnvBody(t, "SCORE_TEST_DISABLE_NOTIFICATIONS_BODY")
	app, out := runLive(t, "profile", "disable-notifications", "--data", body)
	assertOK(t, app, out)
}

func TestLiveWriteDisableNotificationsGroup(t *testing.T) {
	requireDestructive(t)
	requireSanrAuth(t)
	body := requireEnvBody(t, "SCORE_TEST_DISABLE_NOTIFICATIONS_GROUP_BODY")
	app, out := runLive(t, "profile", "disable-notifications-group", "--data", body)
	assertOK(t, app, out)
}

// profile gdpr initiates account deletion and is intentionally never executed
// by this suite. This test documents that decision and always skips; there is
// no env flag that turns it into a real call.
func TestLiveGdprNotInvoked(t *testing.T) {
	t.Skip("profile gdpr is irreversible (account deletion) and is never invoked by tests")
}
