package score

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// describeFlag is the machine-readable description of a single flag.
type describeFlag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage"`
}

// describeCommand is the machine-readable description of one command node.
type describeCommand struct {
	Path        string            `json:"path"`
	Short       string            `json:"short"`
	Flags       []describeFlag    `json:"flags,omitempty"`
	Subcommands []describeCommand `json:"subcommands,omitempty"`
}

// describeDoc is the full catalog emitted by `score describe`.
type describeDoc struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	ExitCodes   map[string]string `json:"exitCodes"`
	GlobalFlags []describeFlag    `json:"globalFlags"`
	Commands    []describeCommand `json:"commands"`
}

func newDescribeCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "describe",
		Short: "Emit a machine-readable catalog of commands, flags, and exit codes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := cmd.Root()
			doc := describeDoc{
				Name:        root.Name(),
				Version:     Version,
				Description: root.Short,
				ExitCodes:   exitCodeCatalog(),
				GlobalFlags: collectFlags(root.PersistentFlags()),
			}
			for _, c := range root.Commands() {
				if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
					continue
				}
				doc.Commands = append(doc.Commands, describeCmd(c))
			}
			if err := app.printer.EmitValue(doc); err != nil {
				return app.fail(err)
			}
			return nil
		},
	}
}

func describeCmd(c *cobra.Command) describeCommand {
	d := describeCommand{
		Path:  c.CommandPath(),
		Short: c.Short,
		Flags: collectFlags(c.LocalFlags()),
	}
	for _, sub := range c.Commands() {
		if sub.Hidden || sub.Name() == "help" {
			continue
		}
		d.Subcommands = append(d.Subcommands, describeCmd(sub))
	}
	return d
}

func collectFlags(set *pflag.FlagSet) []describeFlag {
	var out []describeFlag
	set.VisitAll(func(f *pflag.Flag) {
		out = append(out, describeFlag{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Type:      f.Value.Type(),
			Default:   f.DefValue,
			Usage:     f.Usage,
		})
	})
	return out
}

func exitCodeCatalog() map[string]string {
	return map[string]string{
		"0": "success",
		"1": "unclassified runtime error",
		"2": "bad flags or arguments",
		"3": "authentication/authorization failure (401/403)",
		"4": "resource not found (404)",
		"5": "rate limited (429)",
		"6": "upstream server error (5xx)",
		"7": "network failure or timeout",
	}
}
