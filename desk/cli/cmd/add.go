package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
	"github.com/ElmerMosquera2/cssh/desk/cli/internal/sshutil"
)

var (
	addKey      string
	addEmail    string
	addName     string
	addHosts    []string
	addType     string
	addComment  string
	addNoGenKey bool
	addForce    bool
)

var addCmd = &cobra.Command{
	Use:   "add <perfil>",
	Short: "Crea un perfil (genera o registra su clave SSH)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := strings.TrimSpace(args[0])
		if profileName == "" {
			return fmt.Errorf("el nombre del perfil no puede estar vacío")
		}

		keyPath := strings.TrimSpace(addKey)
		if keyPath == "" {
			var err error
			keyPath, err = sshutil.DefaultKeyPath(profileName, addType)
			if err != nil {
				return err
			}
		}

		out := cmd.OutOrStdout()

		// Si addNoGenKey es falso, generamos la clave
		if !addNoGenKey {
			comment := addComment
			if comment == "" {
				if addEmail != "" {
					comment = addEmail
				} else {
					comment = fmt.Sprintf("%s@cssh", profileName)
				}
			}

			if sshutil.KeyExists(keyPath) && !addForce {
				return fmt.Errorf("la clave SSH ya existe en %s (usa --force para sobrescribir o --no-gen-key para solo registrar)", keyPath)
			}

			_, err := sshutil.GenerateKey(sshutil.KeyGenOptions{
				Type:    addType,
				Path:    keyPath,
				Comment: comment,
				Force:   addForce,
			})
			if err != nil {
				return fmt.Errorf("generar clave SSH: %w", err)
			}
			fmt.Fprintf(out, "Clave SSH generada exitosamente en %s\n", keyPath)
		} else {
			if !sshutil.KeyExists(keyPath) {
				fmt.Fprintf(out, "Advertencia: la clave indicada no existe actualmente en %s\n", keyPath)
			}
		}

		// Leer clave pública para mostrársela al usuario
		pubKey, err := sshutil.ReadPublicKey(keyPath)
		if err == nil && pubKey != "" {
			fmt.Fprintf(out, "\nClave pública (.pub):\n%s\n\n", pubKey)
			fmt.Fprintln(out, "Recuerda copiar y pegar la clave pública en tu proveedor Git (GitHub, GitLab, etc.).")
		}

		// Persistir en profiles.toml
		cfg, err := profile.Load()
		if err != nil {
			return err
		}

		p := profile.Profile{
			Name:    profileName,
			Key:     keyPath,
			Email:   addEmail,
			GitName: addName,
			Hosts:   addHosts,
		}

		if err := cfg.Add(p, addForce); err != nil {
			return err
		}

		// Si es el primer perfil y no hay default, asignarlo como default
		if cfg.Default == "" {
			cfg.Default = profileName
			fmt.Fprintf(out, "Perfil %q configurado automáticamente como default.\n", profileName)
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Fprintf(out, "Perfil %q guardado con éxito en profiles.toml\n", profileName)
		return nil
	},
}

func init() {
	profileCmd.AddCommand(addCmd)
	addCmd.Flags().StringVar(&addKey, "key", "", "ruta de la clave SSH")
	addCmd.Flags().StringVar(&addEmail, "email", "", "email para user.email")
	addCmd.Flags().StringVar(&addName, "name", "", "nombre para user.name")
	addCmd.Flags().StringArrayVar(&addHosts, "host", nil, "host reclamado por el perfil (repetible)")
	addCmd.Flags().StringVar(&addType, "type", "ed25519", "tipo de clave: ed25519|rsa|ecdsa")
	addCmd.Flags().StringVar(&addComment, "comment", "", "comentario de la clave")
	addCmd.Flags().BoolVar(&addNoGenKey, "no-gen-key", false, "solo registrar, no generar")
	addCmd.Flags().BoolVar(&addForce, "force", false, "sobrescribir perfil existente")
}
