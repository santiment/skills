package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/sanr"
)

// competitions are served by the Sanr backend (/v1/competitions).
func newCompetitionsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "competitions",
		Short: "Competitions (Sanr): list, forecasts, issuers",
	}
	cmd.AddCommand(competitionsListCmd(app), competitionsForecastsCmd(app), competitionsIssuersCmd(app))
	return cmd
}

func competitionsListCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "list",
		Short: "List competitions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1CompetitionsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1CompetitionsWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func competitionsForecastsCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "forecasts <contractAddress> <competitionID>",
		Short: "List forecasts for a competition",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1CompetitionsContractAddressCompetitionIDForecastsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1CompetitionsContractAddressCompetitionIDForecastsWithResponse(
				app.ctx(cmd), args[0], args[1], &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func competitionsIssuersCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "issuers <contractAddress> <competitionID>",
		Short: "List issuers for a competition",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1CompetitionsContractAddressCompetitionIDIssuersParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1CompetitionsContractAddressCompetitionIDIssuersWithResponse(
				app.ctx(cmd), args[0], args[1], &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}
