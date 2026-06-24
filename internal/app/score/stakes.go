package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/arena"
)

// stakes are served by the Arena backend (/v1/stakes).
func newStakesCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "stakes",
		Short: "List stakes (Arena)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			var params arena.StakesControllerFindAllParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.StakesControllerFindAllWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleArena, 50)
	return c
}
