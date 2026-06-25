package hyperhandler

import (
	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/hyperhandler/wallet"
	"santiment.net/san-skills/internal/platform/exitcode"
)

func newWalletCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "wallet",
		Short: "HD wallet operations (BIP-39 seed phrases)",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	c.AddCommand(
		newWalletGenerateCmd(app),
		newWalletImportCmd(app),
		newWalletListCmd(app),
		newWalletUseCmd(app),
		newWalletDeleteCmd(app),
	)
	return c
}

func newWalletGenerateCmd(app *App) *cobra.Command {
	var (
		words int
		save  bool
	)
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate a new BIP-39 seed phrase (12 or 24 words)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			mnemonic, err := wallet.GenerateMnemonic(words)
			if err != nil {
				return app.fail(err)
			}
			dk, err := wallet.DeriveHDKey(mnemonic, "", 0)
			if err != nil {
				return app.fail(err)
			}
			saved := false
			if save {
				network := app.resolveNetwork()
				if err := wallet.NewHDWalletProvider().SaveMnemonic(network, mnemonic); err != nil {
					return app.fail(err)
				}
				saved = true
			}
			return app.emitValue(map[string]any{
				"mnemonic": mnemonic,
				"words":    words,
				"address":  dk.Address,
				"path":     dk.Path,
				"saved":    saved,
			})
		},
	}
	f := c.Flags()
	f.IntVar(&words, "words", 12, "number of seed words (12 or 24)")
	f.BoolVar(&save, "save", false, "save the mnemonic to the OS keyring for the network")
	return c
}

func newWalletImportCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "import",
		Short: "Import a BIP-39 seed phrase (read from stdin) into the keyring. STATE-CHANGING.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			mnemonic, err := readSecretStdin()
			if err != nil {
				return app.fail(err)
			}
			if !wallet.ValidateMnemonic(mnemonic) {
				return app.fail(coded(exitcode.Usage, "invalid BIP-39 mnemonic on stdin"))
			}
			if err := wallet.NewHDWalletProvider().SaveMnemonic(network, mnemonic); err != nil {
				return app.fail(err)
			}
			dk, err := wallet.DeriveHDKey(mnemonic, "", 0)
			if err != nil {
				return app.fail(err)
			}
			return app.emitValue(map[string]any{"status": "imported", "network": network, "address": dk.Address})
		},
	}
	return c
}

func newWalletListCmd(app *App) *cobra.Command {
	var count int
	c := &cobra.Command{
		Use:   "list",
		Short: "List derived addresses for the configured seed phrase",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			entries, err := wallet.NewHDWalletProvider().ListAddresses(network, count, 0)
			if err != nil {
				return app.fail(err)
			}
			return app.emitValue(map[string]any{"network": network, "addresses": entries})
		},
	}
	c.Flags().IntVar(&count, "count", 5, "number of addresses to derive")
	return c
}

func newWalletUseCmd(app *App) *cobra.Command {
	var index int
	c := &cobra.Command{
		Use:   "use",
		Short: "Export the private key for a derivation index (SENSITIVE: prints the key)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			res, ok := wallet.NewHDWalletProvider().GetKeyAt(network, index)
			if !ok {
				return app.fail(coded(exitcode.NotFound, "no seed phrase configured for "+network))
			}
			return app.emitValue(map[string]any{
				"network":     network,
				"index":       index,
				"address":     res.Address,
				"private_key": res.Key,
			})
		},
	}
	c.Flags().IntVar(&index, "index", 0, "derivation index")
	return c
}

func newWalletDeleteCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete the seed phrase from the keyring for the network",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			network := app.resolveNetwork()
			deleted := wallet.NewHDWalletProvider().DeleteMnemonic(network)
			return app.emitValue(map[string]any{"network": network, "deleted": deleted})
		},
	}
	return c
}
