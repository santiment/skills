package score

import (
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/sanr"
	"santiment.net/san-skills/internal/platform/config"
	"santiment.net/san-skills/internal/platform/exitcode"
)

func newAuthCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Santiment Score authentication (token cache, login)",
		Long: "The CLI authenticates with a single long-lived Sanr JWT (~5y). The same\n" +
			"token also authenticates Arena (sent as x-api-key), so one credential\n" +
			"covers both backends. Provide it via --token / SANR_TOKEN / `auth set-token`\n" +
			"(set ARENA_API_KEY to the same value, or rely on --api-key). `auth login`\n" +
			"performs the wallet-signature exchange and best-effort caches the token\n" +
			"returned via Set-Cookie.",
	}
	cmd.AddCommand(
		newAuthSetTokenCmd(app),
		newAuthStatusCmd(app),
		newAuthLoginCmd(app),
	)
	return cmd
}

func newAuthSetTokenCmd(app *App) *cobra.Command {
	var token string
	c := &cobra.Command{
		Use:   "set-token",
		Short: "Cache the Santiment Score API token (works for both backends) into the active profile",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if token == "" {
				return app.fail(usageError("--token is required"))
			}
			if err := config.SaveTokens(app.settings.ConfigPath, app.settings.ProfileName, token); err != nil {
				return app.fail(err)
			}
			return app.printer.EmitValue(map[string]any{
				"saved":   true,
				"profile": app.settings.ProfileName,
				"config":  app.settings.ConfigPath,
			})
		},
	}
	c.Flags().StringVar(&token, "token", "", "Santiment Score API token (JWT) to cache (required)")
	return c
}

func newAuthStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the resolved profile, base URLs, and which credentials are present",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			s := app.settings
			return app.printer.EmitValue(map[string]any{
				"profile":         s.ProfileName,
				"config":          s.ConfigPath,
				"sanrBaseURL":     s.SanrBaseURL,
				"arenaBaseURL":    s.ArenaBaseURL,
				"sanrToken":       masked(s.SanrToken),
				"arenaApiKey":     masked(s.ArenaAPIKey),
				"sanrAuthorized":  s.SanrToken != "",
				"arenaAuthorized": s.ArenaAPIKey != "",
			})
		},
	}
}

func newAuthLoginCmd(app *App) *cobra.Command {
	var original, signed string
	var accessExpiresIn int
	c := &cobra.Command{
		Use:   "login",
		Short: "Exchange a signed wallet message for a session, caching any returned token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if original == "" || signed == "" {
				return app.fail(usageError("--original-message and --signed-message are required"))
			}
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			body := sanr.PostV1AuthJSONRequestBody{OriginalMessage: original, SignedMessage: signed}
			if accessExpiresIn > 0 {
				body.AccessTokenExpiresIn = &accessExpiresIn
			}
			resp, err := client.PostV1AuthWithResponse(app.ctx(cmd), body)
			if err != nil {
				return app.fail(err)
			}
			if resp.StatusCode() >= 400 {
				return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
			}
			if tok := tokenFromCookies(resp.HTTPResponse); tok != "" {
				if err := config.SaveTokens(app.settings.ConfigPath, app.settings.ProfileName, tok); err != nil {
					return app.fail(err)
				}
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	c.Flags().StringVar(&original, "original-message", "", "the original message that was signed (required)")
	c.Flags().StringVar(&signed, "signed-message", "", "the wallet signature of the message (required)")
	c.Flags().IntVar(&accessExpiresIn, "access-expires", 0, "access token lifetime in seconds")
	return c
}

// tokenFromCookies best-effort extracts an access token from Set-Cookie headers,
// matching common cookie names.
func tokenFromCookies(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	for _, ck := range resp.Cookies() {
		name := strings.ToLower(ck.Name)
		if strings.Contains(name, "refresh") {
			continue
		}
		if strings.Contains(name, "access") || name == "token" ||
			strings.Contains(name, "jwt") || strings.Contains(name, "auth") {
			return ck.Value
		}
	}
	return ""
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
