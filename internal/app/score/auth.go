package score

import (
	"strings"

	"github.com/spf13/cobra"

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
	return &cobra.Command{
		Use:   "status",
		Short: "Show the resolved profile, base URLs, and whether a token is configured",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			s := app.settings
			return app.printer.EmitValue(map[string]any{
				"profile":      s.ProfileName,
				"config":       s.ConfigPath,
				"sanrBaseURL":  s.SanrBaseURL,
				"arenaBaseURL": s.ArenaBaseURL,
				"token":        masked(s.SanrToken),
				"authorized":   s.SanrToken != "" || s.ArenaAPIKey != "",
			})
		},
	}
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
