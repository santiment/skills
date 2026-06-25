// Package hyperhandler implements the `hyperhandler` CLI: an agent-friendly
// client for trading on the Hyperliquid DEX. It is a stateless executor and
// monitor — each command performs one action and emits JSON. Strategy and risk
// decisions are the caller's responsibility.
//
// Safety: the default network is testnet. Any state-changing command on mainnet
// requires an explicit --confirm. Key entry is non-interactive (env/keyring).
package hyperhandler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/hyperhandler/config"
	"santiment.net/san-skills/internal/hyperhandler/service"
	"santiment.net/san-skills/internal/hyperhandler/signer"
	"santiment.net/san-skills/internal/hyperhandler/wallet"
	"santiment.net/san-skills/internal/platform/exitcode"
	"santiment.net/san-skills/internal/platform/output"
)

// errAbort signals that a command already rendered its result/error and set
// App.exitCode; Execute returns that code instead of treating it as usage.
var errAbort = errors.New("abort")

// App holds global flag state and the resolved runtime for one invocation.
type App struct {
	// persistent flags
	jsonOut    bool
	network    string
	configPath string
	timeout    time.Duration
	verbose    bool
	quiet      bool

	// resolved runtime
	cfg      *config.Config
	printer  *output.Printer
	logger   *slog.Logger
	exitCode int
}

// Execute builds the command tree, runs it under a signal-aware context, and
// returns the process exit code.
func Execute() int {
	app := &App{}
	root := newRootCmd(app)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := root.ExecuteContext(ctx); err != nil {
		if errors.Is(err, errAbort) {
			return app.exitCode
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		return exitcode.Usage
	}
	return app.exitCode
}

func newRootCmd(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "hyperhandler",
		Short: "Agent-friendly CLI for trading on the Hyperliquid DEX",
		Long: "hyperhandler executes trading signals and monitors positions/orders on the\n" +
			"Hyperliquid DEX. Designed for AI agents: use --json for machine-readable output,\n" +
			"`hyperhandler describe --json` for a command/flag catalog, and rely on stable exit\n" +
			"codes. Defaults to testnet; mainnet writes require --confirm.",
		SilenceErrors:     true,
		SilenceUsage:      true,
		PersistentPreRunE: app.setup,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	pf := root.PersistentFlags()
	pf.BoolVar(&app.jsonOut, "json", false, "emit machine-readable JSON")
	pf.StringVarP(&app.network, "network", "n", "", "network: mainnet or testnet (default: config, else testnet)")
	pf.StringVar(&app.configPath, "config", "", "path to config file (default: ~/.hyperhandler/config.yaml)")
	pf.DurationVar(&app.timeout, "timeout", 30*time.Second, "per-command timeout")
	pf.BoolVar(&app.verbose, "verbose", false, "verbose diagnostics on stderr")
	pf.BoolVarP(&app.quiet, "quiet", "q", false, "suppress non-essential stderr output")

	root.AddCommand(
		newVersionCmd(app),
		newDescribeCmd(app),
		newExecCmd(app),
		newValidateCmd(app),
		newStatusCmd(app),
		newPositionsCmd(app),
		newOrdersCmd(app),
		newCancelCmd(app),
		newFaucetCmd(app),
		newConfigCmd(app),
		newWalletCmd(app),
	)
	return root
}

// setup loads configuration and constructs the printer + logger before any
// command runs.
func (app *App) setup(_ *cobra.Command, _ []string) error {
	if app.printer == nil { // tests may inject a buffer-backed printer
		app.printer = output.New(app.jsonOut)
	}

	level := slog.LevelWarn
	switch {
	case app.quiet:
		level = slog.LevelError
	case app.verbose:
		level = slog.LevelDebug
	}
	app.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	cfg, err := config.Load(app.configPath)
	if err != nil {
		app.exitCode = app.printer.EmitError(err)
		return errAbort
	}
	app.cfg = cfg
	return nil
}

// resolveNetwork returns the --network flag value, or the config default.
func (app *App) resolveNetwork() string {
	if app.network != "" {
		return app.network
	}
	return app.cfg.Network()
}

// context derives a per-command context honoring the --timeout flag and the
// root signal-aware context.
func (app *App) context(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	if app.timeout <= 0 {
		return context.WithCancel(cmd.Context())
	}
	return context.WithTimeout(cmd.Context(), app.timeout)
}

// signerFor resolves the configured key for network and builds a signer.
// Returns a coded auth error when no key is configured.
func (app *App) signerFor(network string) (*wallet.WalletManager, *signer.Signer, error) {
	mgr, s, err := service.WalletAndSigner(network)
	if err != nil {
		return nil, nil, err
	}
	if s == nil {
		return mgr, nil, coded(exitcode.Auth, fmt.Sprintf(
			"no private key configured for %s; set HL_%s_PRIVATE_KEY or run `hyperhandler config set-key`",
			network, upper(network)))
	}
	return mgr, s, nil
}

// requireMainnetConfirm enforces the mainnet write guard: a state-changing
// command on mainnet must pass --confirm. testnet is always allowed.
func (app *App) requireMainnetConfirm(network string, confirm bool) error {
	if network == "mainnet" && !confirm {
		return coded(exitcode.Usage,
			"refusing to run a state-changing command on mainnet without --confirm")
	}
	return nil
}

// emitValue renders a Go value as JSON and records the exit code.
func (app *App) emitValue(v any) error {
	if err := app.printer.EmitValue(v); err != nil {
		return app.fail(err)
	}
	return nil
}

// fail renders an error and records its stable exit code.
func (app *App) fail(err error) error {
	app.exitCode = app.printer.EmitError(err)
	return nil
}

// coded builds an error that declares its own stable exit code.
func coded(code int, msg string) error { return codedError{msg: msg, code: code} }

type codedError struct {
	msg  string
	code int
}

func (e codedError) Error() string { return e.msg }
func (e codedError) ExitCode() int { return e.code }

func upper(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 'a' - 'A'
		}
	}
	return string(b)
}
