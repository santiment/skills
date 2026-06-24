// Package score implements the `score` CLI: a single, agent-friendly entry
// point over the two Santiment Score backends (Sanr and Arena). Backend
// selection is automatic per command; callers never choose a backend.
package score

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/arena"
	"santiment.net/san-skills/internal/clients/sanr"
	"santiment.net/san-skills/internal/platform/config"
	"santiment.net/san-skills/internal/platform/exitcode"
	"santiment.net/san-skills/internal/platform/httpx"
	"santiment.net/san-skills/internal/platform/output"
)

// errAbort signals that a command has already rendered its result/error and set
// App.exitCode; Execute should return that code rather than treating it as usage.
var errAbort = errors.New("abort")

// App holds global flag state and the resolved runtime for one invocation.
type App struct {
	// persistent flags
	jsonOut      bool
	profile      string
	configPath   string
	sanrBaseURL  string
	arenaBaseURL string
	token        string
	apiKey       string
	timeout      time.Duration
	verbose      bool
	quiet        bool

	// resolved runtime
	settings *config.Settings
	printer  *output.Printer
	exitCode int
}

// Execute builds the command tree, runs it, and returns the process exit code.
func Execute() int {
	app := &App{}
	root := newRootCmd(app)
	root.SilenceErrors = true
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
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
		Use:   "score",
		Short: "Agent-friendly CLI for the Santiment Score APIs",
		Long: "score is a command-line client for the Santiment Score product. It talks to\n" +
			"two backends (Sanr and Arena) and routes each command to the right one\n" +
			"automatically. Designed for AI agents: use --json for machine-readable output,\n" +
			"`score describe --json` for a command/flag catalog, and rely on stable exit codes.",
		SilenceErrors:     true,
		SilenceUsage:      true,
		PersistentPreRunE: app.setup,
	}

	pf := root.PersistentFlags()
	pf.BoolVar(&app.jsonOut, "json", false, "emit machine-readable JSON (raw API passthrough)")
	pf.StringVar(&app.profile, "profile", "", "config profile to use (default: current_profile or 'default')")
	pf.StringVar(&app.configPath, "config", "", "path to config file (default: ~/.config/score/config.yaml)")
	pf.StringVar(&app.sanrBaseURL, "base-url-sanr", "", "override Sanr backend base URL")
	pf.StringVar(&app.arenaBaseURL, "base-url-arena", "", "override Arena backend base URL")
	pf.StringVar(&app.token, "token", "", "Sanr bearer token (overrides env/config)")
	pf.StringVar(&app.apiKey, "api-key", "", "Arena x-api-key (overrides env/config)")
	pf.DurationVar(&app.timeout, "timeout", 30*time.Second, "per-request timeout")
	pf.BoolVar(&app.verbose, "verbose", false, "verbose diagnostics on stderr")
	pf.BoolVarP(&app.quiet, "quiet", "q", false, "suppress non-essential stderr output")

	root.AddCommand(
		newVersionCmd(app),
		newDescribeCmd(app),
		newHealthCmd(app),
		newAuthCmd(app),
		newPredictionsCmd(app),
		newMarketsCmd(app),
		newPortfolioCmd(app),
		newLeaderboardsCmd(app),
		newCompetitionsCmd(app),
		newPairsCmd(app),
		newPricesCmd(app),
		newProfileCmd(app),
		newIssuersCmd(app),
		newStakesCmd(app),
		newEventsCmd(app),
		newContractsCmd(app),
	)
	return root
}

// setup resolves configuration and constructs the output printer before any
// command runs. Config errors are rendered immediately and abort the run.
func (app *App) setup(_ *cobra.Command, _ []string) error {
	if app.printer == nil { // tests may inject a buffer-backed printer
		app.printer = output.New(app.jsonOut)
	}

	s, err := config.Resolve(config.Overrides{
		ConfigPath:   app.configPath,
		Profile:      app.profile,
		SanrBaseURL:  app.sanrBaseURL,
		ArenaBaseURL: app.arenaBaseURL,
		Token:        app.token,
		APIKey:       app.apiKey,
	})
	if err != nil {
		app.exitCode = app.printer.EmitError(err)
		return errAbort
	}
	app.settings = s
	return nil
}

// httpOptions builds the shared HTTP options for this invocation.
func (app *App) httpOptions() httpx.Options {
	return httpx.Options{Timeout: app.timeout, UserAgent: userAgent()}
}

func (app *App) sanr() (*sanr.ClientWithResponses, error) {
	return sanr.New(app.settings, app.httpOptions())
}

func (app *App) arena() (*arena.ClientWithResponses, error) {
	return arena.New(app.settings, app.httpOptions())
}

// emit renders a raw API response and records the exit code. It always returns
// nil so the cobra command completes cleanly (Execute reads app.exitCode).
func (app *App) emit(backend string, status int, body []byte) error {
	app.exitCode = app.printer.EmitResult(backend, status, body)
	return nil
}

// fail renders a transport/runtime error and records the exit code.
func (app *App) fail(err error) error {
	app.exitCode = app.printer.EmitError(err)
	return nil
}

// ctx returns the command context (cobra wires signal handling into it).
func (app *App) ctx(cmd *cobra.Command) context.Context {
	return cmd.Context()
}
