package rclone

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
)

// CloneOptions configures the clone and bind operation.
type CloneOptions struct {
	Profile  profile.Profile
	URL      string
	Into     string
	NoVerify bool
	DryRun   bool
	GitArgs  []string
	Stdout   io.Writer
	Stderr   io.Writer
}

// Execute performs the complete rclone workflow (verification, clone, git config binding).
func Execute(opts CloneOptions) error {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	urlInfo, err := ParseGitURL(opts.URL)
	if err != nil {
		return err
	}

	destDir := opts.Into
	if destDir == "" {
		destDir = urlInfo.RepoName
	}

	absKey, err := profile.ExpandHome(opts.Profile.Key)
	if err != nil {
		return fmt.Errorf("expandir ruta de clave SSH: %w", err)
	}

	sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", absKey)

	if opts.DryRun {
		fmt.Fprintf(opts.Stdout, "[dry-run] Perfil resuelto: %s\n", opts.Profile.Name)
		fmt.Fprintf(opts.Stdout, "[dry-run] Clave SSH:       %s (%s)\n", opts.Profile.Key, absKey)
		fmt.Fprintf(opts.Stdout, "[dry-run] Directorio:      %s\n", destDir)
		if !opts.NoVerify && urlInfo.Host != "" {
			target := urlInfo.Host
			if urlInfo.User != "" {
				target = urlInfo.User + "@" + urlInfo.Host
			}
			fmt.Fprintf(opts.Stdout, "[dry-run] 1. ssh -T -i %s -o IdentitiesOnly=yes %s\n", absKey, target)
		}
		gitCloneArgs := append([]string{"clone"}, opts.GitArgs...)
		gitCloneArgs = append(gitCloneArgs, opts.URL, destDir)
		fmt.Fprintf(opts.Stdout, "[dry-run] 2. GIT_SSH_COMMAND=%q git %s\n", sshCmd, strings.Join(gitCloneArgs, " "))
		fmt.Fprintf(opts.Stdout, "[dry-run] 3. git -C %s config core.sshCommand %q\n", destDir, sshCmd)
		if opts.Profile.GitName != "" {
			fmt.Fprintf(opts.Stdout, "[dry-run] 4. git -C %s config user.name %q\n", destDir, opts.Profile.GitName)
		}
		if opts.Profile.Email != "" {
			fmt.Fprintf(opts.Stdout, "[dry-run] 5. git -C %s config user.email %q\n", destDir, opts.Profile.Email)
		}
		return nil
	}

	// 1. Fail-fast SSH check (si no se omitió y hay host)
	if !opts.NoVerify && urlInfo.Host != "" {
		fmt.Fprintf(opts.Stdout, "Verificando conexión SSH a %s...\n", urlInfo.Host)
		if err := VerifySSH(absKey, urlInfo.User, urlInfo.Host); err != nil {
			return fmt.Errorf("fail-fast SSH falló: %w\n(Usa --no-verify si deseas omitir esta verificación)", err)
		}
		fmt.Fprintf(opts.Stdout, "✓ Conexión SSH verificada con éxito.\n")
	}

	// 2. Ejecutar git clone
	fmt.Fprintf(opts.Stdout, "Clonando %s en %s usando perfil %q...\n", opts.URL, destDir, opts.Profile.Name)
	cloneArgs := append([]string{"clone"}, opts.GitArgs...)
	cloneArgs = append(cloneArgs, opts.URL, destDir)

	cloneCmd := exec.Command("git", cloneArgs...)
	cloneCmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
	cloneCmd.Stdout = opts.Stdout
	cloneCmd.Stderr = opts.Stderr

	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("error al clonar repositorio: %w", err)
	}

	// 3. Atar el repo localmente al perfil configurando .git/config
	setSSHCmd := exec.Command("git", "-C", destDir, "config", "core.sshCommand", sshCmd)
	if out, err := setSSHCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("configurar core.sshCommand: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	if opts.Profile.GitName != "" {
		setNameCmd := exec.Command("git", "-C", destDir, "config", "user.name", opts.Profile.GitName)
		if out, err := setNameCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("configurar user.name: %s (%w)", strings.TrimSpace(string(out)), err)
		}
	}

	if opts.Profile.Email != "" {
		setEmailCmd := exec.Command("git", "-C", destDir, "config", "user.email", opts.Profile.Email)
		if out, err := setEmailCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("configurar user.email: %s (%w)", strings.TrimSpace(string(out)), err)
		}
	}

	fmt.Fprintf(opts.Stdout, "\n✓ Repositorio clonado exitosamente en %q y atado al perfil %q.\n", destDir, opts.Profile.Name)
	fmt.Fprintf(opts.Stdout, "  core.sshCommand = %q\n", sshCmd)
	if opts.Profile.GitName != "" {
		fmt.Fprintf(opts.Stdout, "  user.name       = %q\n", opts.Profile.GitName)
	}
	if opts.Profile.Email != "" {
		fmt.Fprintf(opts.Stdout, "  user.email      = %q\n", opts.Profile.Email)
	}

	return nil
}
