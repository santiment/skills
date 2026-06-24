package score

import (
	"strings"

	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/arena"
	"santiment.net/san-skills/internal/clients/sanr"
	"santiment.net/san-skills/internal/platform/config"
	"santiment.net/san-skills/internal/platform/exitcode"
)

// newAuthCmd groups the credential commands. Authentication is a single
// long-lived API token (a Sanr JWT) that works for BOTH backends — it is sent
// as the Sanr bearer token and as the Arena x-api-key. There is exactly one way
// in: `auth login --token <TOKEN>`.
func newAuthCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage the Santiment Score API token (login, status, logout)",
		Long: "Authentication is a single long-lived API token that works for both backends\n" +
			"(Sanr and Arena). Generate one at https://sanr.app/user/<username>/settings →\n" +
			"\"Advanced\" → \"Generate token\", then run `score auth login --token <TOKEN>`.\n" +
			"The token is cached in your config file and reused automatically by every\n" +
			"later command — no need to pass it again or set it per backend.",
	}
	cmd.AddCommand(
		newAuthLoginCmd(app),
		newAuthStatusCmd(app),
		newAuthLogoutCmd(app),
	)
	return cmd
}

// newAuthLoginCmd caches the API token into the active profile. This is the only
// entry point — the token authenticates both backends.
func newAuthLoginCmd(app *App) *cobra.Command {
	var token string
	c := &cobra.Command{
		Use:   "login",
		Short: "Save your API token (authenticates both Sanr and Arena)",
		Long: "Generate a token at https://sanr.app/user/<username>/settings → \"Advanced\" →\n" +
			"\"Generate token\", then run `score auth login --token <TOKEN>`. The token is\n" +
			"written to your config file (default ~/.config/score/config.yaml, mode 0600)\n" +
			"and reused automatically by every later command, for both backends.",
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Tolerate paste noise: surrounding whitespace and a leading "Bearer ".
			tok := strings.TrimPrefix(strings.TrimSpace(token), "Bearer ")
			tok = strings.TrimSpace(tok)
			if tok == "" {
				return app.fail(usageError("--token is required; generate one at https://sanr.app/user/<username>/settings → Advanced → Generate token"))
			}
			if err := config.SaveTokens(app.settings.ConfigPath, app.settings.ProfileName, tok); err != nil {
				return app.fail(err)
			}
			return app.printer.EmitValue(map[string]any{
				"saved":   true,
				"profile": app.settings.ProfileName,
				"config":  app.settings.ConfigPath,
			})
		},
	}
	c.Flags().StringVar(&token, "token", "", "Score API token from your Sanr settings page (required)")
	return c
}

func newAuthStatusCmd(app *App) *cobra.Command {
	var verify bool
	c := &cobra.Command{
		Use:   "status",
		Short: "Show the resolved profile/base URLs and whether the token is configured (--verify checks acceptance)",
		Long: "Without --verify this is an offline check: it only reports whether a token is\n" +
			"configured, NOT whether the backends accept it. A configured-but-rejected\n" +
			"token still shows tokenConfigured:true. Use --verify (or `score health`) to\n" +
			"confirm each backend actually accepts the token — a token can be accepted by\n" +
			"one backend and rejected by the other.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s := app.settings
			tokenConfigured := s.SanrToken != "" || s.ArenaAPIKey != ""

			out := map[string]any{
				"profile":         s.ProfileName,
				"config":          s.ConfigPath,
				"sanrBaseURL":     s.SanrBaseURL,
				"arenaBaseURL":    s.ArenaBaseURL,
				"token":           masked(s.SanrToken),
				"tokenConfigured": tokenConfigured,
				"verified":        verify,
			}

			if !verify {
				// Offline: `authorized` reports only that a token is configured.
				// It is NOT proof the backends accept it — flagged by verified:false.
				out["authorized"] = tokenConfigured
				out["hint"] = "authorized here means a token is configured, not that it works — run `score auth status --verify` (or `score health`) to check the backends actually accept it"
				return app.printer.EmitValue(out)
			}

			// Verify: probe both backends with the configured credential and
			// report per-backend acceptance, so a token accepted by one backend
			// but rejected by the other is visible instead of hidden behind a
			// single misleading "authorized" flag.
			ctx := app.ctx(cmd)
			sanrHealth := probe(func() (int, error) {
				cl, err := app.sanr()
				if err != nil {
					return 0, err
				}
				resp, err := cl.GetV1PingWithResponse(ctx)
				if err != nil {
					return 0, err
				}
				return resp.StatusCode(), nil
			})
			arenaHealth := probe(func() (int, error) {
				cl, err := app.arena()
				if err != nil {
					return 0, err
				}
				resp, err := cl.PingControllerPingWithResponse(ctx)
				if err != nil {
					return 0, err
				}
				return resp.StatusCode(), nil
			})

			out["backends"] = map[string]any{
				sanr.Backend:  map[string]any{"accepted": sanrHealth.OK, "httpStatus": sanrHealth.HTTPStatus},
				arena.Backend: map[string]any{"accepted": arenaHealth.OK, "httpStatus": arenaHealth.HTTPStatus},
			}
			authorized := tokenConfigured && sanrHealth.OK && arenaHealth.OK
			out["authorized"] = authorized
			if tokenConfigured && !authorized {
				// A configured token that a backend rejects is an auth failure;
				// surface it via the stable exit-code contract so scripts branch.
				out["hint"] = "a configured token is rejected by at least one backend (see backends[*].httpStatus); if one backend accepts it and another returns 401, the token/account is invalid for the rejecting backend — do not loop on regeneration"
				app.exitCode = exitcode.Auth
			}

			return app.printer.EmitValue(out)
		},
	}
	c.Flags().BoolVar(&verify, "verify", false, "ping both backends to check they actually accept the configured token")
	return c
}

func newAuthLogoutCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the cached token from the active profile",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := config.SaveTokens(app.settings.ConfigPath, app.settings.ProfileName, ""); err != nil {
				return app.fail(err)
			}
			return app.printer.EmitValue(map[string]any{
				"loggedOut": true,
				"profile":   app.settings.ProfileName,
			})
		},
	}
}

func masked(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "***" + s[len(s)-2:]
}

// usageError marks an error as a usage problem (exit code 2).
func usageError(msg string) error {
	return &usageErr{msg: msg}
}

type usageErr struct{ msg string }

func (e *usageErr) Error() string { return e.msg }

// ExitCode lets exitcode.FromError classify usage errors as code 2.
func (e *usageErr) ExitCode() int { return exitcode.Usage }
