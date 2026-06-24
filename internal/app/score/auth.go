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
		Short: "Manage Sanr authentication (token cache, login, refresh)",
		Long: "Sanr uses wallet-signature authentication, so the primary path for agents\n" +
			"is to provide a pre-obtained JWT via --token / SANR_TOKEN / `auth set-token`.\n" +
			"`auth login` performs the signature exchange and best-effort caches a token\n" +
			"returned via Set-Cookie; `auth refresh` renews using a cached refresh token.",
	}
	cmd.AddCommand(
		newAuthSetTokenCmd(app),
		newAuthStatusCmd(app),
		newAuthLoginCmd(app),
		newAuthRefreshCmd(app),
	)
	return cmd
}

func newAuthSetTokenCmd(app *App) *cobra.Command {
	var token, refresh string
	c := &cobra.Command{
		Use:   "set-token",
		Short: "Cache a Sanr JWT (and optional refresh token) into the active profile",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if token == "" {
				return app.fail(usageError("--token is required"))
			}
			if err := config.SaveTokens(app.settings.ConfigPath, app.settings.ProfileName, token, refresh); err != nil {
				return app.fail(err)
			}
			return app.printer.EmitValue(map[string]any{
				"saved":   true,
				"profile": app.settings.ProfileName,
				"config":  app.settings.ConfigPath,
			})
		},
	}
	c.Flags().StringVar(&token, "token", "", "Sanr JWT to cache (required)")
	c.Flags().StringVar(&refresh, "refresh", "", "Sanr refresh token to cache")
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
				"sanrRefresh":     masked(s.SanrRefreshToken),
				"arenaApiKey":     masked(s.ArenaAPIKey),
				"sanrAuthorized":  s.SanrToken != "",
				"arenaAuthorized": s.ArenaAPIKey != "",
			})
		},
	}
}

func newAuthLoginCmd(app *App) *cobra.Command {
	var original, signed string
	var accessExpiresIn, refreshExpiresIn int
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
			if refreshExpiresIn > 0 {
				body.RefreshTokenExpiresIn = &refreshExpiresIn
			}
			resp, err := client.PostV1AuthWithResponse(app.ctx(cmd), body)
			if err != nil {
				return app.fail(err)
			}
			if resp.StatusCode() >= 400 {
				return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
			}
			if tok, rt := tokensFromCookies(resp.HTTPResponse); tok != "" {
				if err := config.SaveTokens(app.settings.ConfigPath, app.settings.ProfileName, tok, rt); err != nil {
					return app.fail(err)
				}
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	c.Flags().StringVar(&original, "original-message", "", "the original message that was signed (required)")
	c.Flags().StringVar(&signed, "signed-message", "", "the wallet signature of the message (required)")
	c.Flags().IntVar(&accessExpiresIn, "access-expires", 0, "access token lifetime in seconds")
	c.Flags().IntVar(&refreshExpiresIn, "refresh-expires", 0, "refresh token lifetime in seconds")
	return c
}

func newAuthRefreshCmd(app *App) *cobra.Command {
	var expiresIn int
	c := &cobra.Command{
		Use:   "refresh",
		Short: "Renew the session using the cached Sanr refresh token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if app.settings.SanrRefreshToken == "" {
				return app.fail(usageError("no cached refresh token; run `score auth login` or `auth set-token --refresh`"))
			}
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			body := sanr.PostV1AuthRefreshJSONRequestBody{RefreshToken: app.settings.SanrRefreshToken}
			if expiresIn > 0 {
				body.ExpiresIn = &expiresIn
			}
			resp, err := client.PostV1AuthRefreshWithResponse(app.ctx(cmd), body)
			if err != nil {
				return app.fail(err)
			}
			if resp.StatusCode() < 400 {
				if tok, rt := tokensFromCookies(resp.HTTPResponse); tok != "" {
					if err := config.SaveTokens(app.settings.ConfigPath, app.settings.ProfileName, tok, rt); err != nil {
						return app.fail(err)
					}
				}
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	c.Flags().IntVar(&expiresIn, "expires", 0, "new token lifetime in seconds")
	return c
}

// tokensFromCookies best-effort extracts an access token and refresh token from
// Set-Cookie headers, matching common cookie names.
func tokensFromCookies(resp *http.Response) (access, refresh string) {
	if resp == nil {
		return "", ""
	}
	for _, ck := range resp.Cookies() {
		name := strings.ToLower(ck.Name)
		switch {
		case strings.Contains(name, "refresh"):
			refresh = ck.Value
		case strings.Contains(name, "access"), name == "token", strings.Contains(name, "jwt"), strings.Contains(name, "auth"):
			if access == "" {
				access = ck.Value
			}
		}
	}
	return access, refresh
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
