package cmd

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista perfiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := profile.Load()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if len(cfg.Profiles) == 0 {
			fmt.Fprintln(out, "No hay perfiles configurados.")
			fmt.Fprintln(out, "Usa 'cssh profile add <perfil>' para crear uno.")
			return nil
		}

		names := make([]string, 0, len(cfg.Profiles))
		for name := range cfg.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)

		w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PERFIL\tDEFAULT\tCLAVE\tEMAIL\tGIT NAME\tHOSTS")

		for _, name := range names {
			p := cfg.Profiles[name]
			isDefault := ""
			if cfg.Default == name {
				isDefault = "*"
			}

			email := p.Email
			if email == "" {
				email = "-"
			}

			gitName := p.GitName
			if gitName == "" {
				gitName = "-"
			}

			hosts := "-"
			if len(p.Hosts) > 0 {
				hosts = strings.Join(p.Hosts, ", ")
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				name, isDefault, p.Key, email, gitName, hosts)
		}
		return w.Flush()
	},
}

func init() {
	profileCmd.AddCommand(listCmd)
}
