package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/sanr"
)

// predictions are served by the Sanr backend (/v2/predictions).
func newPredictionsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "predictions",
		Short: "User predictions (Sanr): list, get, create, update, close",
	}
	cmd.AddCommand(
		predictionsListCmd(app),
		predictionsGetCmd(app),
		predictionsCreateCmd(app),
		predictionsUpdateCmd(app),
		predictionsCloseCmd(app),
		predictionsMerkleCmd(app),
		predictionsPerfChartCmd(app),
	)
	return cmd
}

func predictionsListCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "list",
		Short: "List predictions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV2PredictionsParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV2PredictionsWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleSanr, 50)
	return c
}

func predictionsGetCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a single prediction by numeric id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pathInt(args[0], "id")
			if err != nil {
				return app.fail(err)
			}
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV2PredictionsIdParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV2PredictionsIdWithResponse(app.ctx(cmd), id, &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	return c
}

func predictionsCreateCmd(app *App) *cobra.Command {
	var b bodyFlags
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a prediction (write; body via --data/--data-file)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var body sanr.PostV2PredictionsJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV2PredictionsWithResponse(app.ctx(cmd), body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	b.bind(c.Flags())
	return c
}

func predictionsUpdateCmd(app *App) *cobra.Command {
	var b bodyFlags
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a prediction by numeric id (write)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pathInt(args[0], "id")
			if err != nil {
				return app.fail(err)
			}
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var body sanr.PutV2PredictionsIdJSONRequestBody
			if err := b.into(cmd, &body); err != nil {
				return app.fail(err)
			}
			resp, err := client.PutV2PredictionsIdWithResponse(app.ctx(cmd), id, body)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	b.bind(c.Flags())
	return c
}

func predictionsCloseCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "close <id>",
		Short: "Close a prediction by numeric id (write)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := pathInt(args[0], "id")
			if err != nil {
				return app.fail(err)
			}
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.PostV2PredictionsIdCloseWithResponse(app.ctx(cmd), id)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
}

func predictionsMerkleCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "merkle",
		Short: "Get Merkle proof data for predictions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV2PredictionsMerkleParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.GetV2PredictionsMerkleWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindParams(c.Flags())
	return c
}

func predictionsPerfChartCmd(app *App) *cobra.Command {
	var address string
	c := &cobra.Command{
		Use:   "performance-chart",
		Short: "Get the performance chart for a wallet address",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if address == "" {
				return app.fail(usageError("--address is required"))
			}
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			params := sanr.GetV2PredictionsPerformanceChartParams{Address: address}
			resp, err := client.GetV2PredictionsPerformanceChartWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	c.Flags().StringVar(&address, "address", "", "EVM wallet address (required)")
	return c
}
