package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/sanr"
)

// pairs are served by the Sanr backend (/v1/pairs).
func newPairsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pairs",
		Short: "Trading pairs (Sanr): list, aggregated, categories, favorites",
	}
	cmd.AddCommand(pairsListCmd(app), pairsAggregatedCmd(app), pairsCategoriesCmd(app), pairsFavoritesCmd(app))
	return cmd
}

func pairsListCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "list",
		Short: "List trading pairs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PairsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1PairsWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func pairsAggregatedCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "aggregated",
		Short: "List pairs aggregated by signals",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PairsAggregatedBySignalsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1PairsAggregatedBySignalsWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func pairsCategoriesCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "categories",
		Short: "List pair categories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PairsCategoriesParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1PairsCategoriesWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func pairsFavoritesCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "favorites",
		Short: "List favorite pairs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PairsFavoritesParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1PairsFavoritesWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}
