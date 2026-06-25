//go:build integration

package hyperhandler

import (
	"strings"
	"testing"

	"santiment.net/san-skills/internal/platform/exitcode"
)

// TestIntegrationReadTestnet drives the read-only CLI commands against the live
// testnet for a derived (unfunded) address. Each must exit 0 with valid JSON —
// an unfunded account legitimately returns empty state/positions/orders.
func TestIntegrationReadTestnet(t *testing.T) {
	addr := testReadAddress(t)

	t.Run("describe", func(t *testing.T) {
		app, out := runLiveTestnet(t, "describe")
		assertLiveOK(t, app, out)
		if !strings.Contains(out, `"name":"hyperhandler"`) {
			t.Errorf("describe missing tool name: %s", out)
		}
	})

	t.Run("status", func(t *testing.T) {
		app, out := runLiveTestnet(t, "status", "--address", addr)
		assertLiveOK(t, app, out)
	})

	t.Run("positions", func(t *testing.T) {
		app, out := runLiveTestnet(t, "positions", "--address", addr)
		assertLiveOK(t, app, out)
	})

	t.Run("orders", func(t *testing.T) {
		app, out := runLiveTestnet(t, "orders", "--address", addr)
		assertLiveOK(t, app, out)
	})
}

// TestIntegrationUnknownNetwork confirms a bad --network is a usage error (exit
// 2) before any network call.
func TestIntegrationUnknownNetwork(t *testing.T) {
	app := &App{}
	root := newRootCmd(app)
	root.SetArgs([]string{"--network", "bogusnet", "status", "--address", "0x0"})
	_ = root.Execute()
	if app.exitCode != exitcode.Usage && app.exitCode != exitcode.Generic {
		t.Fatalf("unknown network: expected a non-zero usage/generic exit, got %d", app.exitCode)
	}
}
