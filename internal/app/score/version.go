package score

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version is the CLI version. Overridable at build time with
// -ldflags "-X santiment.net/san-skills/internal/app/score.Version=vX.Y.Z".
var Version = "0.1.0"

func userAgent() string {
	return "score/" + Version
}

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the score CLI version",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			info := map[string]string{"version": Version}
			if bi, ok := debug.ReadBuildInfo(); ok {
				info["go"] = bi.GoVersion
			}
			if err := app.printer.EmitValue(info); err != nil {
				return app.fail(err)
			}
			return nil
		},
	}
}
