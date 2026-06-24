package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/sanr"
)

// leaderboards are served by the Sanr backend (/v1/leaderboards).
func newLeaderboardsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "leaderboards",
		Short: "Leaderboards (Sanr): forecasts, players, and per-player stats",
	}
	cmd.AddCommand(
		lbForecastsCmd(app),
		lbPlayersCmd(app),
		lbPlayerCmd(app),
		lbPlayerForecastsCmd(app),
		lbPlayerCompetitionPointsCmd(app),
		lbPlayerPerformancePointsCmd(app),
		lbPlayerSignalsBySymbolCmd(app),
	)
	return cmd
}

func lbForecastsCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "forecasts",
		Short: "Forecasts leaderboard",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1LeaderboardsForecastsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1LeaderboardsForecastsWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func lbPlayersCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "players",
		Short: "Players leaderboard",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1LeaderboardsPlayersParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1LeaderboardsPlayersWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func lbPlayerCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "player <username>",
		Short: "Get a single player by username",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1LeaderboardsPlayersUsernameWithResponse(app.ctx(cmd), args[0])
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
}

func lbPlayerForecastsCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "player-forecasts <username>",
		Short: "Forecasts for a single player",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1LeaderboardsPlayersUsernameForecastsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1LeaderboardsPlayersUsernameForecastsWithResponse(app.ctx(cmd), args[0], &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func lbPlayerCompetitionPointsCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "player-competition-points <username>",
		Short: "Competition points for a single player",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1LeaderboardsPlayersUsernameCompetitionPointsWithResponse(app.ctx(cmd), args[0])
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
}

func lbPlayerPerformancePointsCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "player-performance-points <username>",
		Short: "Performance points for a single player",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1LeaderboardsPlayersUsernamePerformancePointsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1LeaderboardsPlayersUsernamePerformancePointsWithResponse(app.ctx(cmd), args[0], &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func lbPlayerSignalsBySymbolCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "player-signals-by-symbol <username>",
		Short: "Signal counts grouped by symbol for a single player",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1LeaderboardsPlayersUsernameSignalsAmountGroupedBySymbolParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV1LeaderboardsPlayersUsernameSignalsAmountGroupedBySymbolWithResponse(app.ctx(cmd), args[0], &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}
