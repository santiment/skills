//go:build integration

package hyperhandler

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"santiment.net/san-skills/internal/clients/hyperliquid"
	"santiment.net/san-skills/internal/hyperhandler/config"
	"santiment.net/san-skills/internal/platform/exitcode"
)

// TestIntegrationSignedWriteTestnet exercises the real signed write path on the
// live testnet: it places a limit order far from the mid (so it cannot fill)
// and then cancels it. It is gated behind a funded testnet key + HH_TEST_WRITES=1.
//
// The core assertion is that the request is NOT rejected for a bad signature
// (exit 3): that proves the EIP-712 + msgpack signing path was accepted by the
// venue. A business rejection (e.g. min-notional/margin → exit 1) still counts
// as a successful signing round-trip, so the test stays robust across account
// funding states.
func TestIntegrationSignedWriteTestnet(t *testing.T) {
	key := requireWriteOptIn(t)
	t.Setenv("HL_TESTNET_PRIVATE_KEY", key)

	// Discover a real coin + mid price directly from the live testnet.
	netCfg, err := config.Network("testnet")
	require.NoError(t, err)
	info := hyperliquid.NewInfoClient(netCfg)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	meta, err := info.GetMeta(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, meta.Universe)
	coin := meta.Universe[0].Name
	mid, err := info.GetMidPrice(ctx, coin)
	require.NoError(t, err)
	require.True(t, mid.IsPositive())

	// A limit buy at half the mid will rest, never fill.
	price := mid.Mul(decimal.RequireFromString("0.5"))
	signal := map[string]any{
		"pair":        coin,
		"side":        "long",
		"order_type":  "limit",
		"entry_price": price.String(),
		"size":        "0.01",
		"leverage":    1,
	}
	raw, err := json.Marshal(signal)
	require.NoError(t, err)
	sigPath := filepath.Join(t.TempDir(), "signal.json")
	require.NoError(t, os.WriteFile(sigPath, raw, 0o600))

	app, out := runLiveTestnet(t, "exec", "--signal", sigPath)
	t.Logf("exec exit=%d output=%s", app.exitCode, out)

	// Signature rejection (exit 3) is the failure we care about.
	require.NotEqual(t, exitcode.Auth, app.exitCode,
		"signed exec was rejected at the signature/auth layer: %s", out)

	// Best-effort cleanup: cancel any resting order for the pair.
	capp, cout := runLiveTestnet(t, "cancel", "--pair", coin)
	t.Logf("cancel exit=%d output=%s", capp.exitCode, cout)
}
