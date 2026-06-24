package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/sanr"
)

// prices are served by the Sanr backend (/v1/prices).
func newPricesCmd(app *App) *cobra.Command {
	var symbols, root, timestamp string
	c := &cobra.Command{
		Use:   "prices",
		Short: "Get prices (Sanr); optionally reconstruct a historical snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.sanr()
			if err != nil {
				return app.fail(err)
			}
			var params sanr.GetV1PricesParams
			if symbols != "" {
				params.Symbols = &symbols
			}
			if root != "" {
				params.Root = &root
			}
			if timestamp != "" {
				params.Timestamp = &timestamp
			}
			resp, err := client.GetV1PricesWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(sanr.Backend, resp.StatusCode(), resp.Body)
		},
	}
	c.Flags().StringVar(&symbols, "symbols", "", "comma-separated symbols to include")
	c.Flags().StringVar(&root, "root", "", "exact legacy price root")
	c.Flags().StringVar(&timestamp, "timestamp", "", "timestamp to reconstruct a historical snapshot")
	return c
}
