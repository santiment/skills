package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/arena"
)

// issuers are served by the Arena backend (/v1/issuers), including Hyperliquid
// trading data.
func newIssuersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issuers",
		Short: "Issuers directory (Arena): list, get, Hyperliquid trading data",
	}
	cmd.AddCommand(issuersListCmd(app), issuersGetCmd(app), issuersHyperliquidCmd(app))
	return cmd
}

func issuersListCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "list",
		Short: "List issuers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			var params arena.IssuersControllerFindAllParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.IssuersControllerFindAllWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleArena, 50)
	return c
}

func issuersGetCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "get <id>",
		Short: "Get an issuer by id (with computed position)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			var params arena.IssuersControllerFindOneParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.IssuersControllerFindOneWithResponse(app.ctx(cmd), args[0], &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	return c
}

func issuersHyperliquidCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hyperliquid",
		Short: "Hyperliquid trading data for an issuer: metrics, positions, trades, snapshots",
	}
	cmd.AddCommand(hlMetricsCmd(app), hlPositionsCmd(app), hlTradesCmd(app), hlSnapshotsCmd(app))
	return cmd
}

func hlMetricsCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "metrics <id>",
		Short: "Trading performance metrics for a Hyperliquid trader",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.HyperliquidControllerGetMetricsWithResponse(app.ctx(cmd), args[0])
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
}

func hlPositionsCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "positions <id>",
		Short: "List Hyperliquid positions (use --param status=open|closed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			var params arena.HyperliquidControllerGetPositionsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.HyperliquidControllerGetPositionsWithResponse(app.ctx(cmd), args[0], &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleArena, 50)
	return c
}

func hlTradesCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "trades <id>",
		Short: "List Hyperliquid trades (use --param asset=BTC)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			var params arena.HyperliquidControllerGetTradesParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.HyperliquidControllerGetTradesWithResponse(app.ctx(cmd), args[0], &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleArena, 50)
	return c
}

func hlSnapshotsCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "snapshots <id>",
		Short: "List Hyperliquid account balance snapshots",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			var params arena.HyperliquidControllerGetSnapshotsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.HyperliquidControllerGetSnapshotsWithResponse(app.ctx(cmd), args[0], &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleArena, 50)
	return c
}
