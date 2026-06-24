package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/sanr"
)

// markets are served by the Sanr backend (/v2/markets).
func newMarketsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "markets",
		Short: "Signal markets (Sanr): list, filters, status",
	}
	cmd.AddCommand(marketsListCmd(app), marketsFiltersCmd(app), marketsStatusCmd(app))
	return cmd
}

func marketsListCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "list",
		Short: "List signal markets (supports --param status=, asset=, search=, ...)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV2MarketsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV2MarketsWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func marketsFiltersCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "filters",
		Short: "Get available market filters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV2MarketsFiltersWithResponse(app.ctx(cmd))
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
}

func marketsStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Get market status summary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV2MarketsStatusWithResponse(app.ctx(cmd))
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
}
