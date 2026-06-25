//go:build integration

// Live integration harness for the hyperhandler CLI. Read tests drive the real
// command tree against the Hyperliquid testnet through the same buffer-backed
// printer the unit tests use — no mocks. Write tests are gated behind a funded
// testnet key and an explicit opt-in so the suite degrades gracefully (skips,
// never fails) when they are absent.
//
// Run with: go test -tags=integration ./internal/app/hyperhandler/
//
//	HL_TESTNET_PRIVATE_KEY   funded testnet key; enables the signed write test
//	HH_TEST_WRITES=1         opt in to the signed place/cancel test
package hyperhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"santiment.net/san-skills/internal/hyperhandler/wallet"
	"santiment.net/san-skills/internal/platform/exitcode"
	"santiment.net/san-skills/internal/platform/output"
)

// testReadKey is a well-known throwaway key (hardhat account #0). It is used
// only to derive a stable address for read-only queries — it holds no funds.
const testReadKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

// runLiveTestnet runs the command tree in --json mode against the live testnet
// and returns the App (for exitCode) plus combined stdout+stderr.
func runLiveTestnet(t *testing.T, args ...string) (*App, string) {
	t.Helper()
	app := &App{}
	var buf bytes.Buffer
	app.printer = &output.Printer{JSON: true, Out: &buf, Err: &buf}
	root := newRootCmd(app)
	// Force an absent config file so the suite never reads a developer's real
	// ~/.hyperhandler profile, and pin the network to testnet.
	full := append([]string{
		"--config", filepath.Join(t.TempDir(), "absent.yaml"),
		"--network", "testnet",
	}, args...)
	root.SetArgs(full)
	root.SetOut(&buf)
	root.SetErr(&buf)
	_ = root.ExecuteContext(context.Background())
	return app, buf.String()
}

// testReadAddress derives a stable address from testReadKey via the real wallet
// derivation path (exercises internal secp256k1 → address, no mock).
func testReadAddress(t *testing.T) string {
	t.Helper()
	addr, err := wallet.DeriveAddress(testReadKey)
	require.NoError(t, err)
	return addr
}

func assertLiveOK(t *testing.T, app *App, out string) {
	t.Helper()
	if app.exitCode != exitcode.OK {
		t.Fatalf("exit code: want 0, got %d\noutput: %s", app.exitCode, out)
	}
	assertLiveJSON(t, out)
}

func assertLiveJSON(t *testing.T, out string) {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
}

// requireWriteOptIn skips unless a funded testnet key and the explicit opt-in
// are both present.
func requireWriteOptIn(t *testing.T) string {
	t.Helper()
	key := os.Getenv("HL_TESTNET_PRIVATE_KEY")
	if key == "" {
		t.Skip("set HL_TESTNET_PRIVATE_KEY (funded testnet account) to run the signed write test")
	}
	if os.Getenv("HH_TEST_WRITES") != "1" {
		t.Skip("set HH_TEST_WRITES=1 to run the signed place/cancel test")
	}
	return key
}
