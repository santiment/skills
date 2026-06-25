package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"santiment.net/san-skills/internal/clients/hyperliquid"
	"santiment.net/san-skills/internal/hyperhandler/config"
	"santiment.net/san-skills/internal/hyperhandler/models"
	"santiment.net/san-skills/internal/hyperhandler/signer"
)

// fallbackSlippage is the market-order slippage used where no trading config is
// available (e.g. CancelOrders, which never prices a market order). Matches the
// ExchangeClient/OrderBuilder default Decimal("0.005").
var fallbackSlippage = decimal.RequireFromString("0.005")

// ExecOutcome is the discriminated result of an exec run.
type ExecOutcome string

// Exec outcomes.
const (
	OutcomeDryRun   ExecOutcome = "dry_run"
	OutcomeExecuted ExecOutcome = "executed"
)

// ExecRequest is the input to Executor.Exec. The tool runs in manual mode only:
// the order size, leverage and stop/take levels come straight from the signal.
type ExecRequest struct {
	Signal  *models.TradingSignal
	Network string
	Vault   *string
	DryRun  bool
}

// ExecResult is the structured outcome of an exec run. Results is populated only
// for the executed outcome.
type ExecResult struct {
	Outcome  ExecOutcome
	Warnings []string
	Signal   *models.TradingSignal
	Results  []models.OrderResult
}

// ValidationError carries the validator's error list so the caller can surface
// each failed rule.
type ValidationError struct{ Errors []string }

func (e *ValidationError) Error() string {
	return fmt.Sprintf("signal validation failed: %v", e.Errors)
}

// Executor orchestrates manual-mode signal execution. It owns the
// side-effecting dependencies (config, signer) and an optional Reporter for
// progress lines.
type Executor struct {
	Config   *config.Config
	Signer   *signer.Signer
	Reporter func(string) // progress lines ("Setting leverage..."); nil discards
}

func (e *Executor) report(format string, args ...any) {
	if e.Reporter != nil {
		e.Reporter(fmt.Sprintf(format, args...))
	}
}

// Exec validates and (unless dry-run) executes a signal as-is. The validator
// enforces the config "security" limits (max leverage, max position size USD,
// require-stop-loss) before anything is signed or sent.
func (e *Executor) Exec(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	signal := req.Signal

	// Validate against config limits before signing/sending anything.
	validator := ValidatorFromConfig(e.Config)
	vr := validator.Validate(signal, nil)
	if !vr.Valid {
		return nil, &ValidationError{Errors: vr.Errors}
	}

	netCfg, err := e.Config.NetworkConfig(req.Network)
	if err != nil {
		return nil, err
	}

	// Client tuning + market-order slippage from the trading config section.
	trading := e.Config.Settings().Trading
	opts := ClientOptions(trading)
	slippage := Slippage(trading)

	result := &ExecResult{Warnings: vr.Warnings, Signal: signal}
	if req.DryRun {
		result.Outcome = OutcomeDryRun
		return result, nil
	}

	info := hyperliquid.NewInfoClient(netCfg, opts...)
	exch := hyperliquid.NewExchangeClient(netCfg, e.Signer, slippage, opts...)

	assetIndex, err := info.GetAssetIndex(ctx, signal.Pair)
	if err != nil {
		return nil, err
	}
	assetInfo, err := info.GetAssetInfo(ctx, signal.Pair)
	if err != nil {
		return nil, err
	}
	szDecimals := assetInfo.SzDecimals

	var currentPrice *decimal.Decimal
	if signal.IsMarket() {
		mid, err := info.GetMidPrice(ctx, signal.Pair)
		if err != nil {
			return nil, err
		}
		currentPrice = &mid
	}

	e.report("Setting leverage to %dx...", signal.Leverage)
	exch.SetLeverage(ctx, assetIndex, signal.Leverage, true, req.Vault)

	e.report("Placing orders...")
	results, err := exch.PlaceOrderFromSignal(ctx, signal, assetIndex, currentPrice, req.Vault, szDecimals)
	if err != nil {
		return nil, err
	}

	result.Outcome = OutcomeExecuted
	result.Results = results
	return result, nil
}
