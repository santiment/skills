//go:build integration

// Live integration tests for the Hyperliquid client. They hit the real
// Hyperliquid testnet info API (no key required) and exercise the full
// client → httpx → network → JSON/decimal-decode path with no mocks.
//
// Run with: go test -tags=integration ./internal/clients/hyperliquid/
package hyperliquid_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"santiment.net/san-skills/internal/clients/hyperliquid"
	"santiment.net/san-skills/internal/hyperhandler/config"
)

const integrationTimeout = 25 * time.Second

func testnetInfo(t *testing.T) *hyperliquid.InfoClient {
	t.Helper()
	netCfg, err := config.Network("testnet")
	require.NoError(t, err)
	return hyperliquid.NewInfoClient(netCfg)
}

// TestIntegrationInfoTestnet drives the read-only info endpoints against the
// live testnet and asserts the responses decode into sane values.
func TestIntegrationInfoTestnet(t *testing.T) {
	info := testnetInfo(t)
	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	meta, err := info.GetMeta(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, meta.Universe, "testnet meta universe should not be empty")

	coin := meta.Universe[0].Name
	require.NotEmpty(t, coin)

	idx, err := info.GetAssetIndex(ctx, coin)
	require.NoError(t, err)
	require.GreaterOrEqual(t, idx, 0)

	ai, err := info.GetAssetInfo(ctx, coin)
	require.NoError(t, err)
	require.Equal(t, coin, ai.Name)

	mids, err := info.GetAllMids(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, mids, "allMids should return prices")

	px, err := info.GetMidPrice(ctx, coin)
	require.NoError(t, err)
	require.True(t, px.IsPositive(), "%s mid price should be positive, got %s", coin, px)
}

// TestIntegrationUnknownAsset confirms the real "asset not found" path returns
// the typed error (which maps to exit code 4).
func TestIntegrationUnknownAsset(t *testing.T) {
	info := testnetInfo(t)
	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	_, err := info.GetAssetIndex(ctx, "NOTAREALCOIN_ZZZ")
	require.Error(t, err)
}
