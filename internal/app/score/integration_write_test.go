//go:build integration

// Live write tests. These mutate data on the test stand, so they are gated:
//   - SCORE_TEST_WRITES=1        enables the reversible writes below
//   - SCORE_TEST_DESTRUCTIVE=1   additionally enables financial/irreversible ones
//
// Where a valid body can be built safely from the OpenAPI schema (predictions
// create needs only a real marketId + direction; profile update is a no-op flag
// echo) the test constructs it. Financial/portfolio bodies carry account-specific
// amounts, prices and ids, so those are supplied via env (capture a real payload
// from the web client) rather than invented. `profile gdpr` (account deletion)
// has no test at all — it must never run.
package score

import (
	"encoding/json"
	"fmt"
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

// profile update with a no-op body: re-send the current value of a cosmetic
// boolean flag so the request mutates nothing observable. Body built from the
// PutV1Issuer schema (all fields optional).
func TestLiveWriteProfileUpdate(t *testing.T) {
	requireWrites(t)
	requireSanrAuth(t)

	_, getOut := runLive(t, "profile", "get")
	cur := false
	if obj, ok := decodeObject(t, getOut); ok {
		if b, ok := obj["closedNoteHint"].(bool); ok {
			cur = b
		}
	}
	body := fmt.Sprintf(`{"closedNoteHint":%t}`, cur)
	app, out := runLive(t, "profile", "update", "--data", body)
	assertOK(t, app, out)
}

// Prediction lifecycle: create -> get (verify) -> update -> close. The body is
// built from a real market id (PostV2Predictions requires marketId + direction);
// the prediction is closed at the end so the lifecycle cleans up after itself.
func TestLiveWritePredictionLifecycle(t *testing.T) {
	requireWrites(t)
	requireSanrAuth(t)

	_, mkOut := runLive(t, "markets", "list", "--take", "1")
	marketID, ok := firstListID(mkOut, "id")
	if !ok {
		t.Skip("no market id available to build a prediction body")
	}
	body := fmt.Sprintf(`{"marketId":%q,"direction":"up"}`, marketID)

	app, out := runLive(t, "predictions", "create", "--data", body)
	assertOK(t, app, out)

	id, ok := objID(out, "id")
	if !ok {
		t.Fatalf("could not extract created prediction id from: %s", out)
	}
	t.Logf("created prediction id=%s (market %s)", id, marketID)

	// Verify it is now readable.
	getApp, getOut := runLive(t, "predictions", "get", id)
	assertOK(t, getApp, getOut)

	// Update take-profit in place (PutV2PredictionsId accepts stop/take prices).
	uApp, uOut := runLive(t, "predictions", "update", id, "--data", `{"takeProfitPrice":1000000}`)
	assertOK(t, uApp, uOut)

	// Close it to clean up the lifecycle.
	cApp, cOut := runLive(t, "predictions", "close", id)
	assertOK(t, cApp, cOut)
}

// portfolio distribution-save with a caller-supplied body. The payload carries
// asset amounts, prices and percentages that are account/portfolio specific, so
// it is supplied via env (capture a real one from the web client) rather than
// invented here.
func TestLiveWriteDistributionSave(t *testing.T) {
	requireWrites(t)
	requireSanrAuth(t)
	body := requireEnvBody(t, "SCORE_TEST_DISTRIBUTION_BODY")
	app, out := runLive(t, "portfolio", "distribution-save", "--data", body)
	assertOK(t, app, out)
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
