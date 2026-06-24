//go:build integration

package score

import (
	"encoding/json"
	"strings"
	"testing"
)

// readCase is one read-only command invocation expected to succeed.
type readCase struct {
	name string
	args []string
	// dataKey, when set, asserts the JSON response is an object with that key.
	dataKey string
}

func runReadCases(t *testing.T, cases []readCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, out := runLive(t, tc.args...)
			assertOK(t, app, out)
			assertJSON(t, out)
			if tc.dataKey != "" {
				assertHasKey(t, out, tc.dataKey)
			}
		})
	}
}

// Public Sanr read endpoints — no credentials required.
func TestLiveReadSanrPublic(t *testing.T) {
	runReadCases(t, []readCase{
		{name: "markets-list", args: []string{"markets", "list", "--take", "2"}, dataKey: "data"},
		{name: "markets-filters", args: []string{"markets", "filters"}},
		{name: "markets-status", args: []string{"markets", "status"}},
		{name: "predictions-list", args: []string{"predictions", "list", "--take", "2"}, dataKey: "data"},
		{name: "predictions-merkle", args: []string{"predictions", "merkle"}},
		{name: "leaderboards-forecasts", args: []string{"leaderboards", "forecasts", "--take", "2"}},
		{name: "leaderboards-players", args: []string{"leaderboards", "players", "--take", "2"}},
		{name: "competitions-list", args: []string{"competitions", "list", "--take", "2"}},
		{name: "pairs-list", args: []string{"pairs", "list", "--take", "2"}},
		{name: "pairs-categories", args: []string{"pairs", "categories"}},
		{name: "prices", args: []string{"prices"}},
	})
}

// Authenticated Sanr read endpoints — require a valid SANR_TOKEN. Reaching exit
// OK (not 3/Auth) is itself the proof that the cached token is accepted.
func TestLiveReadSanrAuthed(t *testing.T) {
	requireSanrAuth(t)
	runReadCases(t, []readCase{
		{name: "profile-get", args: []string{"profile", "get"}},
		{name: "profile-notifications-status", args: []string{"profile", "notifications-status"}},
		{name: "pairs-favorites", args: []string{"pairs", "favorites", "--take", "2"}}, // requires auth (401 when anonymous)
	})
}

// Portfolio reads are authenticated but also depend on the account actually
// having an issuer/portfolio. They tolerate the "no data for this account"
// business error so the suite passes for both populated and cleared accounts.
func TestLiveReadPortfolio(t *testing.T) {
	requireSanrAuth(t)
	cases := []readCase{
		{name: "portfolio-balances", args: []string{"portfolio", "balances", "--take", "2"}},
		{name: "portfolio-historical", args: []string{"portfolio", "historical-balances", "--take", "2"}},
		{name: "portfolio-distribution", args: []string{"portfolio", "distribution"}},
		{name: "portfolio-positions", args: []string{"portfolio", "positions"}},
		{name: "portfolio-orders", args: []string{"portfolio", "orders"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, out := runLive(t, tc.args...)
			assertAuthedRead(t, app, out)
		})
	}
}

// Arena read endpoints — require a valid ARENA_API_KEY.
func TestLiveReadArena(t *testing.T) {
	requireArenaAuth(t)
	runReadCases(t, []readCase{
		{name: "issuers-list", args: []string{"issuers", "list", "--limit", "2"}, dataKey: "data"},
		{name: "issuers-hyperliquid", args: []string{"issuers", "hyperliquid"}},
		{name: "stakes", args: []string{"stakes", "--limit", "2"}},
		{name: "events", args: []string{"events", "--limit", "2"}},
		{name: "contracts-list", args: []string{"contracts", "list"}},
	})
}

// Detail endpoints driven by ids discovered from their list endpoints, so the
// suite never hard-codes ids that rot. Each derivation skips (not fails) when
// the upstream list is empty or shaped unexpectedly.
func TestLiveReadDynamicSanr(t *testing.T) {
	_, listOut := runLive(t, "predictions", "list", "--take", "1")
	if id, ok := firstListID(listOut, "id"); ok {
		t.Run("predictions-get", func(t *testing.T) {
			app, out := runLive(t, "predictions", "get", id)
			assertOK(t, app, out)
			assertJSON(t, out)
		})
	} else {
		t.Log("skip predictions-get: no prediction id in list response")
	}

	// pairs aggregated rejects an empty/unknown filter (WRONG_SORT_OR_FILTER_FIELD),
	// and the set of valid filter fields is endpoint-specific. Exercise it only
	// when the caller supplies a known-good filter (without outer braces, e.g.
	// "asset":"BTC"); otherwise skip rather than guess.
	if filter := envOrEmpty("SCORE_TEST_PAIRS_FILTER"); filter != "" {
		t.Run("pairs-aggregated", func(t *testing.T) {
			app, out := runLive(t, "pairs", "aggregated", "--filter", filter)
			assertOK(t, app, out)
			assertJSON(t, out)
		})
	} else {
		t.Log("skip pairs-aggregated: set SCORE_TEST_PAIRS_FILTER to a valid filter")
	}

	_, playersOut := runLive(t, "leaderboards", "players", "--take", "1")
	username, ok := listField(playersOut, "username", "name", "handle")
	if !ok {
		t.Log("skip leaderboards player detail: no username in list response")
		return
	}
	runReadCases(t, []readCase{
		{name: "leaderboards-player", args: []string{"leaderboards", "player", username}},
		{name: "leaderboards-player-forecasts", args: []string{"leaderboards", "player-forecasts", username, "--take", "2"}},
		{name: "leaderboards-player-competition-points", args: []string{"leaderboards", "player-competition-points", username}},
		{name: "leaderboards-player-performance-points", args: []string{"leaderboards", "player-performance-points", username}},
		{name: "leaderboards-player-signals-by-symbol", args: []string{"leaderboards", "player-signals-by-symbol", username}},
	})
}

func TestLiveReadDynamicArena(t *testing.T) {
	requireArenaAuth(t)
	_, listOut := runLive(t, "issuers", "list", "--limit", "1")
	id, ok := firstListID(listOut, "id")
	if !ok {
		t.Skip("no issuer id in Arena list response")
	}
	runReadCases(t, []readCase{
		{name: "issuers-get", args: []string{"issuers", "get", id}},
		{name: "issuers-metrics", args: []string{"issuers", "metrics", id}},
		{name: "issuers-positions", args: []string{"issuers", "positions", id, "--limit", "2"}},
		{name: "issuers-trades", args: []string{"issuers", "trades", id, "--limit", "2"}},
		{name: "issuers-snapshots", args: []string{"issuers", "snapshots", id, "--limit", "2"}},
	})

	_, contractsOut := runLive(t, "contracts", "list")
	if addr, ok := listField(contractsOut, "address", "contractAddress", "contract_address"); ok {
		t.Run("contracts-abi", func(t *testing.T) {
			app, out := runLive(t, "contracts", "abi", addr)
			assertOK(t, app, out)
			assertJSON(t, out)
		})
	} else {
		t.Log("skip contracts-abi: no contract address in list response")
	}
}

// Pagination flags must reach the backend: Sanr --take and Arena --limit should
// each cap the returned data array.
func TestLivePaginationSanr(t *testing.T) {
	_, out := runLive(t, "markets", "list", "--take", "1")
	if n := dataLen(out); n > 1 {
		t.Errorf("--take 1 should return at most 1 item, got %d", n)
	}
}

func TestLivePaginationArena(t *testing.T) {
	requireArenaAuth(t)
	_, out := runLive(t, "issuers", "list", "--limit", "1")
	if n := dataLen(out); n > 1 {
		t.Errorf("--limit 1 should return at most 1 item, got %d", n)
	}
}

// The generic --param escape hatch must be plumbed through to the query string;
// here it is an alternate way to express the Sanr take parameter.
func TestLiveParamEscapeHatch(t *testing.T) {
	app, out := runLive(t, "markets", "list", "--param", "take=1")
	assertOK(t, app, out)
	if n := dataLen(out); n > 1 {
		t.Errorf("--param take=1 should cap to 1 item, got %d", n)
	}
}

// Human (non-JSON) mode should pretty-print the body with indentation.
func TestLiveHumanModePretty(t *testing.T) {
	app, out := runLiveNoJSON(t, "markets", "list", "--take", "1")
	assertOK(t, app, out)
	if !strings.Contains(out, "\n  ") {
		t.Errorf("human mode should indent JSON, got: %s", out)
	}
}

// listField returns data[0].<field> trying several candidate field names.
func listField(out string, fields ...string) (string, bool) {
	for _, f := range fields {
		if v, ok := firstListID(out, f); ok {
			return v, true
		}
	}
	return "", false
}

// dataLen returns len(data) for a {"data":[...]} response, or 0 otherwise.
func dataLen(out string) int {
	var doc struct {
		Data []json.RawMessage `json:"data"`
	}
	if json.Unmarshal([]byte(out), &doc) != nil {
		return 0
	}
	return len(doc.Data)
}
