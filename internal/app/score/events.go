package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/arena"
)

// events are served by the Arena backend (/v1/events).
func newEventsCmd(app *App) *cobra.Command {
	var q queryFlags
	c := &cobra.Command{
		Use:   "events",
		Short: "List domain events (Arena)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			var params arena.EventsControllerFindAllParams
			if err := q.into(&params); err != nil {
				return app.fail(err)
			}
			resp, err := client.EventsControllerFindAllWithResponse(app.ctx(cmd), &params)
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
	q.bindList(c.Flags(), styleArena, 50)
	return c
}
