package score

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/arena"
)

// contracts are served by the Arena backend (/v1/contracts).
func newContractsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contracts",
		Short: "Contracts (Arena): list, abi",
	}
	cmd.AddCommand(contractsListCmd(app), contractsAbiCmd(app))
	return cmd
}

func contractsListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all contracts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.ContractsControllerGetAllWithResponse(app.ctx(cmd))
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
}

func contractsAbiCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "abi <address>",
		Short: "Get the ABI for a contract address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.arena()
			if err != nil {
				return app.fail(err)
			}
			resp, err := client.ContractsControllerGetAbiWithResponse(app.ctx(cmd), args[0])
			if err != nil {
				return app.fail(err)
			}
			return app.emit(arena.Backend, resp.StatusCode(), resp.Body)
		},
	}
}
