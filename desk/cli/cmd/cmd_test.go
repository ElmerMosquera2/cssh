package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func executeCommandWithInput(input string, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	in := strings.NewReader(input)
	rootCmd.SetIn(in)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestProfileCLIWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CSSH_CONFIG_DIR", tmpDir)

	// 1. List when empty
	out, err := executeCommand("profile", "list")
	if err != nil {
		t.Fatalf("profile list error: %v", err)
	}
	if !strings.Contains(out, "No hay perfiles configurados") {
		t.Errorf("expected empty message, got: %s", out)
	}

	// 2. Add profile with key generation
	keyPath := filepath.Join(tmpDir, "id_ed25519_pers")
	out, err = executeCommand("profile", "add", "personal",
		"--key", keyPath,
		"--email", "elmer@pers.dev",
		"--name", "Elmer Personal",
		"--host", "github.com",
		"--host", "gitlab.com",
	)
	if err != nil {
		t.Fatalf("profile add error: %v", err)
	}
	if !strings.Contains(out, "Clave SSH generada exitosamente") {
		t.Errorf("expected success message, got: %s", out)
	}
	if !strings.Contains(out, "Perfil \"personal\" configurado automáticamente como default") {
		t.Errorf("expected default auto-assignment, got: %s", out)
	}

	// 3. Add second profile with --no-gen-key
	mockWorkKey := filepath.Join(tmpDir, "id_rsa_work")
	_ = os.WriteFile(mockWorkKey, []byte("fake-key"), 0600)

	out, err = executeCommand("profile", "add", "empresa",
		"--key", mockWorkKey,
		"--email", "elmer@work.com",
		"--name", "Elmer Work",
		"--host", "corp.internal",
		"--no-gen-key",
	)
	if err != nil {
		t.Fatalf("profile add second error: %v", err)
	}
	if !strings.Contains(out, "Perfil \"empresa\" guardado con éxito") {
		t.Errorf("expected success message, got: %s", out)
	}

	// 4. List profiles
	out, err = executeCommand("profile", "list")
	if err != nil {
		t.Fatalf("profile list error: %v", err)
	}
	if !strings.Contains(out, "personal") || !strings.Contains(out, "empresa") {
		t.Errorf("expected both profiles in list, got: %s", out)
	}
	if !strings.Contains(out, "*") {
		t.Errorf("expected default indicator '*' in list, got: %s", out)
	}

	// 5. Show profile
	out, err = executeCommand("profile", "show", "personal")
	if err != nil {
		t.Fatalf("profile show error: %v", err)
	}
	if !strings.Contains(out, "Perfil:        personal (default)") {
		t.Errorf("expected show default header, got: %s", out)
	}
	if !strings.Contains(out, "Git user.name: Elmer Personal") {
		t.Errorf("expected git user name, got: %s", out)
	}
	if !strings.Contains(out, "github.com, gitlab.com") {
		t.Errorf("expected hosts, got: %s", out)
	}

	// 6. Set default
	out, err = executeCommand("profile", "default", "empresa")
	if err != nil {
		t.Fatalf("profile default error: %v", err)
	}
	if !strings.Contains(out, "Perfil default establecido a \"empresa\"") {
		t.Errorf("expected default confirmation, got: %s", out)
	}

	// Verify show now shows empresa as default
	out, _ = executeCommand("profile", "show", "empresa")
	if !strings.Contains(out, "Perfil:        empresa (default)") {
		t.Errorf("expected empresa to be default, got: %s", out)
	}

	// 7. Remove with cancellation
	out, err = executeCommandWithInput("n\n", "profile", "remove", "personal")
	if err != nil {
		t.Fatalf("profile remove cancel error: %v", err)
	}
	if !strings.Contains(out, "Operación cancelada") {
		t.Errorf("expected cancellation message, got: %s", out)
	}

	// 8. Remove with confirmation "s"
	out, err = executeCommandWithInput("s\n", "profile", "remove", "personal")
	if err != nil {
		t.Fatalf("profile remove confirm error: %v", err)
	}
	if !strings.Contains(out, "Perfil \"personal\" eliminado de profiles.toml") {
		t.Errorf("expected removal message, got: %s", out)
	}

	// 9. Remove default with --force
	out, err = executeCommand("profile", "remove", "empresa", "--force")
	if err != nil {
		t.Fatalf("profile remove force error: %v", err)
	}
	if !strings.Contains(out, "(era el perfil default)") {
		t.Errorf("expected wasDefault indication, got: %s", out)
	}
}

func TestRcloneCLI(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CSSH_CONFIG_DIR", tmpDir)

	keyPath := filepath.Join(tmpDir, "id_ed25519_pers")
	_ = os.WriteFile(keyPath, []byte("dummy-key"), 0600)

	// Crear perfil para pruebas de rclone
	_, err := executeCommand("profile", "add", "github-pers",
		"--key", keyPath,
		"--email", "pers@github.com",
		"--name", "Personal Dev",
		"--host", "github.com",
		"--no-gen-key",
	)
	if err != nil {
		t.Fatalf("profile add error: %v", err)
	}

	// 1. rclone con perfil implícito por match único de host en modo --dry-run
	out, err := executeCommand("rclone", "git@github.com:ElmerMosquera2/cssh.git", "--dry-run")
	if err != nil {
		t.Fatalf("rclone dry-run error: %v", err)
	}
	if !strings.Contains(out, "[dry-run] Perfil resuelto: github-pers") {
		t.Errorf("expected resolved profile github-pers, got: %s", out)
	}
	if !strings.Contains(out, "git -C cssh config core.sshCommand") {
		t.Errorf("expected binding in dry run, got: %s", out)
	}

	// 2. rclone con perfil explícito y flag --into
	out, err = executeCommand("rclone", "github-pers", "git@github.com:ElmerMosquera2/cssh.git", "--into", "custom-dir", "--dry-run")
	if err != nil {
		t.Fatalf("rclone explicit dry-run error: %v", err)
	}
	if !strings.Contains(out, "[dry-run] Directorio:      custom-dir") {
		t.Errorf("expected custom-dir, got: %s", out)
	}

	// 3. rclone con perfil inexistente
	_, err = executeCommand("rclone", "no-existe", "git@github.com:user/repo.git", "--dry-run")
	if err == nil {
		t.Fatal("expected error with non-existent explicit profile, got nil")
	}
}
