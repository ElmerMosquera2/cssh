package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove <perfil>",
	Short: "Elimina un perfil del TOML (no borra la clave)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := strings.TrimSpace(args[0])

		cfg, err := profile.Load()
		if err != nil {
			return err
		}

		// Validar que el perfil exista antes de pedir confirmación
		if _, exists := cfg.Profiles[target]; !exists {
			return fmt.Errorf("perfil %q no existe", target)
		}

		out := cmd.OutOrStdout()

		if !removeForce {
			fmt.Fprintf(out, "¿Estás seguro de que deseas eliminar el perfil %q de profiles.toml? [s/N]: ", target)
			reader := bufio.NewReader(cmd.InOrStdin())
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("error leyendo confirmación: %w", err)
			}
			response = strings.ToLower(strings.TrimSpace(response))
			if response != "s" && response != "si" && response != "y" && response != "yes" {
				fmt.Fprintln(out, "Operación cancelada.")
				return nil
			}
		}

		wasDefault, err := cfg.Remove(target)
		if err != nil {
			return err
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		if wasDefault {
			fmt.Fprintf(out, "Perfil %q eliminado de profiles.toml (era el perfil default).\n", target)
		} else {
			fmt.Fprintf(out, "Perfil %q eliminado de profiles.toml.\n", target)
		}
		fmt.Fprintln(out, "Nota: las claves SSH en disco no han sido eliminadas.")
		return nil
	},
}

func init() {
	profileCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolVar(&removeForce, "force", false, "no pedir confirmación")
}
