package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/sanr"
)

// portfolio is served by the Sanr backend (/v1/portfolio). Several subcommands
// are write operations (place/cancel orders, deposit, withdrawal).
func newPortfolioCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "Portfolio (Sanr): balances, orders, positions, deposits, withdrawals",
	}
	cmd.AddCommand(
		portfolioBalancesCmd(app),
		portfolioHistoricalCmd(app),
		portfolioDistributionCmd(app),
		portfolioDistributionSaveCmd(app),
		portfolioPositionsCmd(app),
		portfolioOrdersCmd(app),
		portfolioOrderPlaceCmd(app),
		portfolioOrderCancelCmd(app),
		portfolioDepositCmd(app),
		portfolioWithdrawalCmd(app),
		portfolioRecalcCmd(app),
	)
	return cmd
}

func portfolioBalancesCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "balances",
		Short: "Get current portfolio balances",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PortfolioBalancesParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1PortfolioBalancesWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func portfolioHistoricalCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "historical-balances",
		Short: "Get historical portfolio balances",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PortfolioHistoricalBalancesParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1PortfolioHistoricalBalancesWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func portfolioDistributionCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "distribution",
		Short: "Get assets distribution snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PortfolioAssetsDistributionParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1PortfolioAssetsDistributionWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func portfolioDistributionSaveCmd(app *App) *cobra.Command {
	var q queryFlags
	var b bodyFlags
	c := &cobra.Command{
		Use:   "distribution-save",
		Short: "Save an assets distribution snapshot (write)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.PostV1PortfolioAssetsDistributionParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			var body sanr.PostV1PortfolioAssetsDistributionJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV1PortfolioAssetsDistributionWithResponse(app.ctx(cmd), &params, body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	b.bind(c.Flags())
	return c
}

func portfolioPositionsCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "positions",
		Short: "List open positions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PortfolioPositionParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1PortfolioPositionWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func portfolioOrdersCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "orders",
		Short: "List portfolio orders",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PortfolioOrderParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1PortfolioOrderWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func portfolioOrderPlaceCmd(app *App) *cobra.Command {
	var q queryFlags
	var b bodyFlags
	c := &cobra.Command{
		Use:   "order-place",
		Short: "Place a new order (write)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.PostV1PortfolioOrderParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			var body sanr.PostV1PortfolioOrderJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV1PortfolioOrderWithResponse(app.ctx(cmd), &params, body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	b.bind(c.Flags())
	return c
}

func portfolioOrderCancelCmd(app *App) *cobra.Command {
	var q queryFlags
	var b bodyFlags
	c := &cobra.Command{
		Use:   "order-cancel",
		Short: "Cancel an existing order (write)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.DeleteV1PortfolioOrderParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			var body sanr.DeleteV1PortfolioOrderJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.DeleteV1PortfolioOrderWithResponse(app.ctx(cmd), &params, body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	b.bind(c.Flags())
	return c
}

func portfolioDepositCmd(app *App) *cobra.Command {
	var q queryFlags
	var b bodyFlags
	c := &cobra.Command{
		Use:   "deposit",
		Short: "Record a deposit into the portfolio (write)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.PostV1PortfolioDepositParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			var body sanr.PostV1PortfolioDepositJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV1PortfolioDepositWithResponse(app.ctx(cmd), &params, body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	b.bind(c.Flags())
	return c
}

func portfolioWithdrawalCmd(app *App) *cobra.Command {
	var q queryFlags
	var b bodyFlags
	c := &cobra.Command{
		Use:   "withdrawal",
		Short: "Record a withdrawal from the portfolio (write)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.PostV1PortfolioWithdrawalParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			var body sanr.PostV1PortfolioWithdrawalJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV1PortfolioWithdrawalWithResponse(app.ctx(cmd), &params, body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	b.bind(c.Flags())
	return c
}

func portfolioRecalcCmd(app *App) *cobra.Command {
	var q queryFlags
	var b bodyFlags
	c := &cobra.Command{
		Use:   "recalc-historical",
		Short: "Recalculate historical balances using provided price data (write)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.PostV1PortfolioGetHistoricalBalancesByPricesParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			var body sanr.PostV1PortfolioGetHistoricalBalancesByPricesJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV1PortfolioGetHistoricalBalancesByPricesWithResponse(app.ctx(cmd), &params, body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	b.bind(c.Flags())
	return c
}
