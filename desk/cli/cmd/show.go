package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
	"github.com/ElmerMosquera2/cssh/desk/cli/internal/sshutil"
)

var showCmd = &cobra.Command{
	Use:   "show <perfil>",
	Short: "Muestra el detalle de un perfil",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := strings.TrimSpace(args[0])

		cfg, err := profile.Load()
		if err != nil {
			return err
		}

		p, err := cfg.Resolve(target)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		fmt.Fprintf(out, "Perfil:        %s", p.Name)
		if cfg.Default == p.Name {
			fmt.Fprint(out, " (default)")
		}
		fmt.Fprintln(out)

		keyStatus := "no encontrada en disco"
		if sshutil.KeyExists(p.Key) {
			keyStatus = "existe en disco"
		}
		fmt.Fprintf(out, "Clave privada: %s (%s)\n", p.Key, keyStatus)

		if pubKey, err := sshutil.ReadPublicKey(p.Key); err == nil && pubKey != "" {
			fmt.Fprintf(out, "Clave pública: %s\n", pubKey)
		} else {
			fmt.Fprintf(out, "Clave pública: (no disponible)\n")
		}

		email := p.Email
		if email == "" {
			email = "(no configurado)"
		}
		fmt.Fprintf(out, "Email:         %s\n", email)

		gitName := p.GitName
		if gitName == "" {
			gitName = "(no configurado)"
		}
		fmt.Fprintf(out, "Git user.name: %s\n", gitName)

		hosts := "(ninguno)"
		if len(p.Hosts) > 0 {
			hosts = strings.Join(p.Hosts, ", ")
		}
		fmt.Fprintf(out, "Hosts:         %s\n", hosts)

		return nil
	},
}

func init() {
	profileCmd.AddCommand(showCmd)
}
