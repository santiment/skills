package hyperhandler

import "github.com/spf13/cobra"

// Version is the build version, overridable via:
// -ldflags "-X santiment.net/san-skills/internal/app/hyperhandler.Version=v1.2.3".
var Version = "dev"

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return app.emitValue(map[string]string{"name": "hyperhandler", "version": Version})
		},
	}
}
