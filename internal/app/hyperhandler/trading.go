package hyperhandler

import (
	"os"

	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/hyperliquid"
	"santiment.net/san-skills/internal/hyperhandler/models"
	"santiment.net/san-skills/internal/hyperhandler/service"
	"santiment.net/san-skills/internal/hyperhandler/wallet"
	"santiment.net/san-skills/internal/platform/exitcode"
)

// loadSignal reads a trading signal from a file path, or from stdin when path is
// empty or "-".
func (app *App) loadSignal(path string) (*models.TradingSignal, error) {
	if path == "" || path == "-" {
		return service.ParseSignalReader(os.Stdin)
	}
	return service.ParseSignalFile(path)
}

// resolveAddress returns the explicit override address, or the address of the
// configured key for the network.
func (app *App) resolveAddress(network, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	addr, err := wallet.NewWalletManager().GetAddress(network)
	if err != nil {
		return "", err
	}
	if addr == "" {
		return "", coded(exitcode.Auth, "no key configured and no --address given for "+network)
	}
	return addr, nil
}

func (app *App) clientOpts() []hyperliquid.Option {
	return service.ClientOptions(app.cfg.Settings().Trading)
}

func newExecCmd(app *App) *cobra.Command {
	var (
		signalPath string
		vault      string
		dryRun     bool
		confirm    bool
	)
	c := &cobra.Command{
		Use:   "exec",
		Short: "Execute a trading signal (manual mode). STATE-CHANGING on mainnet.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			signal, err := app.loadSignal(signalPath)
			if err != nil {
				return app.fail(err)
			}
			if !dryRun {
				if err := app.requireMainnetConfirm(network, confirm); err != nil {
					return app.fail(err)
				}
			}
			_, s, err := app.signerFor(network)
			if err != nil {
				return app.fail(err)
			}

			app.logger.Info("exec",
				"network", network, "pair", signal.Pair, "side", string(signal.Side),
				"type", string(signal.OrderType), "size", signal.Size.String(),
				"leverage", signal.Leverage, "dry_run", dryRun)

			ctx, cancel := app.context(cmd)
			defer cancel()

			ex := &service.Executor{
				Config:   app.cfg,
				Signer:   s,
				Reporter: func(msg string) { app.logger.Info(msg) },
			}
			var vaultPtr *string
			if vault != "" {
				vaultPtr = &vault
			}
			res, err := ex.Exec(ctx, service.ExecRequest{
				Signal:  signal,
				Network: network,
				Vault:   vaultPtr,
				DryRun:  dryRun,
			})
			if err != nil {
				return app.fail(err)
			}
			return app.emitValue(res)
		},
	}
	f := c.Flags()
	f.StringVar(&signalPath, "signal", "", "path to signal JSON file (default: stdin)")
	f.StringVar(&vault, "vault", "", "vault address to trade on behalf of")
	f.BoolVar(&dryRun, "dry-run", false, "validate and price without sending orders")
	f.BoolVar(&confirm, "confirm", false, "confirm a state-changing action on mainnet")
	return c
}

func newValidateCmd(app *App) *cobra.Command {
	var signalPath string
	c := &cobra.Command{
		Use:   "validate",
		Short: "Validate a trading signal against config limits without executing",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			signal, err := app.loadSignal(signalPath)
			if err != nil {
				return app.fail(err)
			}
			vr := service.ValidatorFromConfig(app.cfg).Validate(signal, nil)
			return app.emitValue(map[string]any{
				"valid":    vr.Valid,
				"errors":   vr.Errors,
				"warnings": vr.Warnings,
				"signal":   signal,
			})
		},
	}
	c.Flags().StringVar(&signalPath, "signal", "", "path to signal JSON file (default: stdin)")
	return c
}

func newStatusCmd(app *App) *cobra.Command {
	var address string
	c := &cobra.Command{
		Use:   "status",
		Short: "Show account state (margin, balances) for the address",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			addr, err := app.resolveAddress(network, address)
			if err != nil {
				return app.fail(err)
			}
			netCfg, err := app.cfg.NetworkConfig(network)
			if err != nil {
				return app.fail(err)
			}
			ctx, cancel := app.context(cmd)
			defer cancel()
			info := hyperliquid.NewInfoClient(netCfg, app.clientOpts()...)
			state, err := info.GetAccountState(ctx, addr)
			if err != nil {
				return app.fail(err)
			}
			return app.emitValue(map[string]any{"network": network, "address": addr, "state": state})
		},
	}
	c.Flags().StringVar(&address, "address", "", "address to query (default: configured key)")
	return c
}

func newPositionsCmd(app *App) *cobra.Command {
	var address string
	c := &cobra.Command{
		Use:   "positions",
		Short: "List open positions for the address",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			addr, err := app.resolveAddress(network, address)
			if err != nil {
				return app.fail(err)
			}
			netCfg, err := app.cfg.NetworkConfig(network)
			if err != nil {
				return app.fail(err)
			}
			ctx, cancel := app.context(cmd)
			defer cancel()
			info := hyperliquid.NewInfoClient(netCfg, app.clientOpts()...)
			positions, err := info.GetPositions(ctx, addr)
			if err != nil {
				return app.fail(err)
			}
			return app.emitValue(map[string]any{"network": network, "address": addr, "positions": positions})
		},
	}
	c.Flags().StringVar(&address, "address", "", "address to query (default: configured key)")
	return c
}

func newOrdersCmd(app *App) *cobra.Command {
	var address string
	c := &cobra.Command{
		Use:   "orders",
		Short: "List open orders for the address",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			addr, err := app.resolveAddress(network, address)
			if err != nil {
				return app.fail(err)
			}
			netCfg, err := app.cfg.NetworkConfig(network)
			if err != nil {
				return app.fail(err)
			}
			ctx, cancel := app.context(cmd)
			defer cancel()
			info := hyperliquid.NewInfoClient(netCfg, app.clientOpts()...)
			orders, err := info.GetOpenOrders(ctx, addr)
			if err != nil {
				return app.fail(err)
			}
			return app.emitValue(map[string]any{"network": network, "address": addr, "orders": orders})
		},
	}
	c.Flags().StringVar(&address, "address", "", "address to query (default: configured key)")
	return c
}

func newCancelCmd(app *App) *cobra.Command {
	var (
		orderID int64
		pair    string
		all     bool
		vault   string
		confirm bool
	)
	c := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel open orders by id, pair, or all. STATE-CHANGING on mainnet.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			if orderID == 0 && pair == "" && !all {
				return app.fail(coded(exitcode.Usage, "specify one of --order-id, --pair, or --all"))
			}
			if err := app.requireMainnetConfirm(network, confirm); err != nil {
				return app.fail(err)
			}
			_, s, err := app.signerFor(network)
			if err != nil {
				return app.fail(err)
			}
			netCfg, err := app.cfg.NetworkConfig(network)
			if err != nil {
				return app.fail(err)
			}
			req := service.CancelRequest{Network: network, All: all}
			if orderID != 0 {
				req.OrderID = &orderID
			}
			if pair != "" {
				req.Pair = &pair
			}
			if vault != "" {
				req.Vault = &vault
			}
			app.logger.Info("cancel", "network", network, "order_id", orderID, "pair", pair, "all", all)

			ctx, cancel := app.context(cmd)
			defer cancel()
			n, err := service.CancelOrders(ctx, netCfg, s, req, app.clientOpts()...)
			if err != nil {
				return app.fail(err)
			}
			return app.emitValue(map[string]any{"network": network, "canceled": n})
		},
	}
	f := c.Flags()
	f.Int64Var(&orderID, "order-id", 0, "cancel a single order by id")
	f.StringVar(&pair, "pair", "", "cancel all orders for this pair")
	f.BoolVar(&all, "all", false, "cancel all open orders")
	f.StringVar(&vault, "vault", "", "vault address to act on behalf of")
	f.BoolVar(&confirm, "confirm", false, "confirm a state-changing action on mainnet")
	return c
}

func newFaucetCmd(app *App) *cobra.Command {
	var address string
	c := &cobra.Command{
		Use:   "faucet",
		Short: "Request testnet funds (testnet only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			if network != "testnet" {
				return app.fail(coded(exitcode.Usage, "faucet is only available on testnet"))
			}
			addr, err := app.resolveAddress(network, address)
			if err != nil {
				return app.fail(err)
			}
			netCfg, err := app.cfg.NetworkConfig(network)
			if err != nil {
				return app.fail(err)
			}
			ctx, cancel := app.context(cmd)
			defer cancel()
			info := hyperliquid.NewInfoClient(netCfg, app.clientOpts()...)
			res, err := info.Faucet(ctx, addr)
			if err != nil {
				return app.fail(err)
			}
			return app.emitValue(map[string]any{"network": network, "address": addr, "response": res})
		},
	}
	c.Flags().StringVar(&address, "address", "", "address to fund (default: configured key)")
	return c
}
