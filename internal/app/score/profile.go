package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/sanr"
)

// profile is the authenticated user's own issuer record on the Sanr backend
// (/v1/issuer). This is distinct from `issuers` (the Arena public directory).
func newProfileCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Own issuer profile (Sanr): get, update, notifications, GDPR",
	}
	cmd.AddCommand(
		profileGetCmd(app),
		profileUpdateCmd(app),
		profileNotificationsStatusCmd(app),
		profileDisableNotificationsCmd(app),
		profileDisableNotificationsGroupCmd(app),
		profileGdprCmd(app),
	)
	return cmd
}

func profileGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get the authenticated user's profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1IssuerWithResponse(app.ctx(cmd))
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
}

func profileUpdateCmd(app *App) *cobra.Command {
	var b bodyFlags
	c := &cobra.Command{
		Use:   "update",
		Short: "Update the authenticated user's profile (write; body via --data/--data-file)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var body sanr.PutV1IssuerJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PutV1IssuerWithResponse(app.ctx(cmd), body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	b.bind(c.Flags())
	return c
}

func profileNotificationsStatusCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "notifications-status",
		Short: "Get notification settings status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1IssuerGetNotificationsStatusParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1IssuerGetNotificationsStatusWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	return c
}

func profileDisableNotificationsCmd(app *App) *cobra.Command {
	var b bodyFlags
	c := &cobra.Command{
		Use:   "disable-notifications",
		Short: "Disable notifications (write; body via --data/--data-file)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var body sanr.PostV1IssuerDisableNotificationsJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV1IssuerDisableNotificationsWithResponse(app.ctx(cmd), body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	b.bind(c.Flags())
	return c
}

func profileDisableNotificationsGroupCmd(app *App) *cobra.Command {
	var b bodyFlags
	c := &cobra.Command{
		Use:   "disable-notifications-group",
		Short: "Disable notifications for a group (write; JSON array body via --data/--data-file)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var body sanr.PostV1IssuerDisableNotificationsForGroupJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV1IssuerDisableNotificationsForGroupWithResponse(app.ctx(cmd), body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	b.bind(c.Flags())
	return c
}

func profileGdprCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "gdpr",
		Short: "Trigger a GDPR request for the authenticated user (write)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV1IssuerGdprWithResponse(app.ctx(cmd))
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
}
