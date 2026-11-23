package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
)

var defaultCmd = &cobra.Command{
	Use:   "default <perfil>",
	Short: "Marca un perfil como default",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := strings.TrimSpace(args[0])

		cfg, err := profile.Load()
		if err != nil {
			return err
		}

		if err := cfg.SetDefault(target); err != nil {
			return err
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Perfil default establecido a %q con éxito.\n", target)
		return nil
	},
}

func init() {
	profileCmd.AddCommand(defaultCmd)
}
