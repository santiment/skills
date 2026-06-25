package service

import (
	"context"
	"strings"

	"santiment.net/san-skills/internal/clients/hyperliquid"
	"santiment.net/san-skills/internal/hyperhandler/config"
	"santiment.net/san-skills/internal/hyperhandler/signer"
)

// CancelRequest selects which open orders to cancel. At least one of OrderID,
// Pair or All must be set (the cli enforces this before calling).
type CancelRequest struct {
	Network string
	Vault   *string
	OrderID *int64
	Pair    *string
	All     bool
}

// CancelOrders cancels matching open orders for the account (vault when set,
// else the signer) and returns the number of orders canceled.
func CancelOrders(ctx context.Context, netCfg config.NetworkConfig, s *signer.Signer, req CancelRequest, opts ...hyperliquid.Option) (int, error) {
	address := s.Address()
	if req.Vault != nil {
		address = *req.Vault
	}

	// Slippage is irrelevant for cancellations (no market order is priced).
	info := hyperliquid.NewInfoClient(netCfg, opts...)
	exch := hyperliquid.NewExchangeClient(netCfg, s, fallbackSlippage, opts...)

	orders, err := info.GetOpenOrders(ctx, address)
	if err != nil {
		return 0, err
	}

	canceled := 0
	for _, order := range orders {
		shouldCancel := false
		switch {
		case req.All:
			shouldCancel = true
		case req.OrderID != nil && order.OrderID == *req.OrderID:
			shouldCancel = true
		case req.Pair != nil && strings.EqualFold(order.Coin, *req.Pair):
			shouldCancel = true
		}
		if !shouldCancel {
			continue
		}

		assetIndex, err := info.GetAssetIndex(ctx, order.Coin)
		if err != nil {
			return canceled, err
		}
		if exch.CancelOrder(ctx, assetIndex, order.OrderID, req.Vault) {
			canceled++
		}
	}
	return canceled, nil
}
