package cmd

import (
	"github.com/spf13/cobra"
	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
	"github.com/ElmerMosquera2/cssh/desk/cli/internal/rclone"
)

var (
	rcloneInto     string
	rcloneNoVerify bool
	rcloneDryRun   bool
	rcloneGitArgs  []string
)

var rcloneCmd = &cobra.Command{
	Use:   "rclone [<perfil>] <url-ssh>",
	Short: "Clona un repo SSH usando la clave de un perfil",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var explicitProfile, gitURL string
		if len(args) == 1 {
			gitURL = args[0]
		} else {
			explicitProfile = args[0]
			gitURL = args[1]
		}

		cfg, err := profile.Load()
		if err != nil {
			return err
		}

		p, err := rclone.ResolveProfile(cfg, explicitProfile, gitURL)
		if err != nil {
			return err
		}

		return rclone.Execute(rclone.CloneOptions{
			Profile:  p,
			URL:      gitURL,
			Into:     rcloneInto,
			NoVerify: rcloneNoVerify,
			DryRun:   rcloneDryRun,
			GitArgs:  rcloneGitArgs,
			Stdout:   cmd.OutOrStdout(),
			Stderr:   cmd.ErrOrStderr(),
		})
	},
}

func init() {
	rootCmd.AddCommand(rcloneCmd)
	rcloneCmd.Flags().StringVar(&rcloneInto, "into", "", "directorio destino")
	rcloneCmd.Flags().BoolVar(&rcloneNoVerify, "no-verify", false, "saltar validación ssh -T")
	rcloneCmd.Flags().BoolVar(&rcloneDryRun, "dry-run", false, "mostrar comandos sin ejecutar")
	rcloneCmd.Flags().StringArrayVar(&rcloneGitArgs, "git-arg", nil, "argumento extra para git clone")
}
