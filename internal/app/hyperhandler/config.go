package hyperhandler

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/hyperhandler/wallet"
	"santiment.net/san-skills/internal/platform/exitcode"
)

// readSecretStdin reads a single secret (key/mnemonic) from stdin and trims
// surrounding whitespace. Reading from stdin keeps the secret out of argv (and
// thus out of `ps`), which is why no --key flag exists.
func readSecretStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func newConfigCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Manage keys and inspect configuration",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	c.AddCommand(
		newConfigSetKeyCmd(app),
		newConfigRemoveKeyCmd(app),
		newConfigShowAddressCmd(app),
		newConfigCheckCmd(app),
	)
	return c
}

func newConfigSetKeyCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "set-key",
		Short: "Store a private key in the OS keyring (key read from stdin). STATE-CHANGING.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			key, err := readSecretStdin()
			if err != nil {
				return app.fail(err)
			}
			if key == "" {
				return app.fail(coded(exitcode.Usage, "no key on stdin; pipe the private key, e.g. `printf %s \"$KEY\" | hyperhandler config set-key`"))
			}
			mgr := wallet.NewWalletManager()
			if err := mgr.SaveToKeyring(network, key); err != nil {
				return app.fail(err)
			}
			addr, _ := wallet.DeriveAddress(key)
			return app.emitValue(map[string]any{"status": "saved", "network": network, "address": addr})
		},
	}
}

func newConfigRemoveKeyCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "remove-key",
		Short: "Remove a private key from the OS keyring",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			removed := wallet.NewWalletManager().RemoveFromKeyring(network)
			return app.emitValue(map[string]any{"network": network, "removed": removed})
		},
	}
}

func newConfigShowAddressCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show-address",
		Short: "Show the Ethereum address of the configured key",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			addr, err := wallet.NewWalletManager().GetAddress(network)
			if err != nil {
				return app.fail(err)
			}
			if addr == "" {
				return app.fail(coded(exitcode.Auth, "no key configured for "+network))
			}
			return app.emitValue(map[string]any{"network": network, "address": addr})
		},
	}
}

func newConfigCheckCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Report config path, active network, and key-provider availability",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			providers := wallet.NewWalletManager().CheckProviders(network)
			return app.emitValue(map[string]any{
				"config_path": app.cfg.Path(),
				"network":     network,
				"providers":   providers,
			})
		},
	}
}
