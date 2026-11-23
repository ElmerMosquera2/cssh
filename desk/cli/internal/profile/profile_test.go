package profile

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func captureStderr(fn func()) string {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestConfigPathAndDir(t *testing.T) {
	// Rule 1 & 2: Path único, determinista y profiles.toml
	tmpDir := t.TempDir()
	t.Setenv("CSSH_CONFIG_DIR", tmpDir)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir error: %v", err)
	}
	if dir != tmpDir {
		t.Errorf("expected dir %s, got %s", tmpDir, dir)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath error: %v", err)
	}
	expectedPath := filepath.Join(tmpDir, "profiles.toml")
	if path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, path)
	}

	// Without CSSH_CONFIG_DIR
	t.Setenv("CSSH_CONFIG_DIR", "")
	userConfig, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir error: %v", err)
	}
	dir, err = ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir error: %v", err)
	}
	expectedDefaultDir := filepath.Join(userConfig, "cssh")
	if dir != expectedDefaultDir {
		t.Errorf("expected default dir %s, got %s", expectedDefaultDir, dir)
	}
}

func TestHomeExpansionAndRelativization(t *testing.T) {
	// Rule 10: Convención ~/
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir error: %v", err)
	}

	// Relativize under home
	target := filepath.Join(home, ".ssh", "id_ed25519")
	rel, err := RelativizeHome(target)
	if err != nil {
		t.Fatalf("RelativizeHome error: %v", err)
	}
	if rel != "~/.ssh/id_ed25519" {
		t.Errorf("expected ~/.ssh/id_ed25519, got %s", rel)
	}

	// Expand ~/
	expanded, err := ExpandHome("~/.ssh/id_ed25519")
	if err != nil {
		t.Fatalf("ExpandHome error: %v", err)
	}
	expectedExp := filepath.Join(home, ".ssh", "id_ed25519")
	if expanded != expectedExp {
		t.Errorf("expected %s, got %s", expectedExp, expanded)
	}

	// Path outside home should remain absolute
	var outside string
	if runtime.GOOS == "windows" {
		outside = `C:\some\other\path\key`
	} else {
		outside = "/var/secrets/key"
	}
	relOutside, err := RelativizeHome(outside)
	if err != nil {
		t.Fatalf("RelativizeHome outside error: %v", err)
	}
	if relOutside != filepath.Clean(outside) {
		t.Errorf("expected %s, got %s", filepath.Clean(outside), relOutside)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	// Rule 5: Lectura tolerante a archivo ausente
	nonExistent := filepath.Join(t.TempDir(), "missing.toml")
	cfg, err := LoadFrom(nonExistent)
	if err != nil {
		t.Fatalf("LoadFrom should not fail on non-existent file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if len(cfg.Profiles) != 0 {
		t.Errorf("expected empty profiles, got %d", len(cfg.Profiles))
	}
	if cfg.Default != "" {
		t.Errorf("expected empty default, got %q", cfg.Default)
	}
}

func TestLoadCorruptFile(t *testing.T) {
	// Rule 6: Lectura estricta a archivo corrupto
	corruptFile := filepath.Join(t.TempDir(), "corrupt.toml")
	if err := os.WriteFile(corruptFile, []byte("default = [corrupt toml syntax"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	_, err := LoadFrom(corruptFile)
	if err == nil {
		t.Fatal("expected error on corrupt TOML file, got nil")
	}
	if !strings.Contains(err.Error(), corruptFile) {
		t.Errorf("expected error message to contain path %q, got: %v", corruptFile, err)
	}
}

func TestKeyMandatory(t *testing.T) {
	// Rule 8: key es obligatorio
	file := filepath.Join(t.TempDir(), "profiles.toml")
	content := `
[profiles.invalido]
email = "test@example.com"
`
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	_, err := LoadFrom(file)
	if err == nil {
		t.Fatal("expected error for profile without key, got nil")
	}
	expectedErrMsg := `perfil "invalido" no tiene clave asignada`
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("expected error %q, got: %v", expectedErrMsg, err)
	}
}

func TestClosedSchemaAndBOM(t *testing.T) {
	// Rule 7 (esquema cerrado) & Rule 14 (UTF-8 BOM) & Rule 13 (hidratación de Name)
	file := filepath.Join(t.TempDir(), "profiles.toml")
	// With UTF-8 BOM: \xEF\xBB\xBF
	content := "\xef\xbb\xbf" + `
default = "personal"
extra_root_field = "ignored"

[profiles.personal]
key = "~/.ssh/id_ed25519_pers"
email = "pers@example.com"
name = "Personal Name"
hosts = ["github.com"]
unknown_field = "ignored"

[profiles.vps]
key = "/etc/ssh/vps_key"
`
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	cfg, err := LoadFrom(file)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if cfg.Default != "personal" {
		t.Errorf("expected default 'personal', got %q", cfg.Default)
	}
	if len(cfg.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg.Profiles))
	}

	pers, ok := cfg.Profiles["personal"]
	if !ok {
		t.Fatal("profile 'personal' not found")
	}
	if pers.Name != "personal" {
		t.Errorf("expected Name hydrated to 'personal', got %q", pers.Name)
	}
	if pers.Email != "pers@example.com" {
		t.Errorf("expected email 'pers@example.com', got %q", pers.Email)
	}
	if pers.GitName != "Personal Name" {
		t.Errorf("expected git name 'Personal Name', got %q", pers.GitName)
	}
	if len(pers.Hosts) != 1 || pers.Hosts[0] != "github.com" {
		t.Errorf("unexpected hosts: %v", pers.Hosts)
	}

	// Pure SSH profile without email or git name (Rule 9)
	vps, ok := cfg.Profiles["vps"]
	if !ok {
		t.Fatal("profile 'vps' not found")
	}
	if vps.Name != "vps" {
		t.Errorf("expected Name hydrated to 'vps', got %q", vps.Name)
	}
	if vps.Email != "" || vps.GitName != "" {
		t.Errorf("expected empty email and git name for vps, got email=%q name=%q", vps.Email, vps.GitName)
	}
}

func TestAtomicSaveAndPerms(t *testing.T) {
	// Rule 3 (Permisos 0700 dir, 0600 archivo) & Rule 4 (Escritura atómica)
	tmpDir := filepath.Join(t.TempDir(), "sub", "config")
	filePath := filepath.Join(tmpDir, "profiles.toml")

	cfg := &Config{
		Default: "work",
		Profiles: map[string]Profile{
			"work": {
				Key:     "~/.ssh/id_ed25519_work",
				Email:   "work@company.com",
				GitName: "Worker Bee",
			},
		},
	}

	if err := cfg.SaveTo(filePath); err != nil {
		t.Fatalf("SaveTo error: %v", err)
	}

	// Check directory perms on Unix
	if runtime.GOOS != "windows" {
		dirFi, err := os.Stat(tmpDir)
		if err != nil {
			t.Fatalf("Stat dir error: %v", err)
		}
		if dirFi.Mode().Perm() != 0700 {
			t.Errorf("expected dir perm 0700, got %#o", dirFi.Mode().Perm())
		}

		fileFi, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Stat file error: %v", err)
		}
		if fileFi.Mode().Perm() != 0600 {
			t.Errorf("expected file perm 0600, got %#o", fileFi.Mode().Perm())
		}
	}

	// Verify .tmp does not remain
	if _, err := os.Stat(filePath + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file .tmp was not cleaned up")
	}

	// Idempotence (Rule 15): save again, contents should be identical
	bytes1, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	if err := cfg.SaveTo(filePath); err != nil {
		t.Fatalf("SaveTo 2 error: %v", err)
	}

	bytes2, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile 2 error: %v", err)
	}

	if !bytes.Equal(bytes1, bytes2) {
		t.Errorf("Save is not idempotent. First:\n%s\nSecond:\n%s", string(bytes1), string(bytes2))
	}
}

func TestDefaultValidation(t *testing.T) {
	// Rule 12: default válido al guardar y warn al cargar
	file := filepath.Join(t.TempDir(), "profiles.toml")
	cfg := &Config{
		Default: "non_existent",
		Profiles: map[string]Profile{
			"valid": {Key: "~/.ssh/key"},
		},
	}

	// Save should fail because default profile doesn't exist
	if err := cfg.SaveTo(file); err == nil {
		t.Fatal("expected SaveTo to fail when default points to non-existent profile")
	}

	// Now write directly to file with an invalid default to test Load behavior
	rawTOML := `
default = "missing"

[profiles.valid]
key = "~/.ssh/key"
`
	if err := os.WriteFile(file, []byte(rawTOML), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	var loadedCfg *Config
	stderrOutput := captureStderr(func() {
		var err error
		loadedCfg, err = LoadFrom(file)
		if err != nil {
			t.Fatalf("LoadFrom error: %v", err)
		}
	})

	// Must reset Default to "" and emit warning to stderr
	if loadedCfg.Default != "" {
		t.Errorf("expected Default to be cleared to empty string, got %q", loadedCfg.Default)
	}
	if !strings.Contains(stderrOutput, `advertencia: perfil default "missing" no existe`) {
		t.Errorf("expected warning on stderr, got: %q", stderrOutput)
	}
}

func TestOperations(t *testing.T) {
	// CRUD: Add, Resolve, SetDefault, Remove
	cfg := &Config{
		Profiles: make(map[string]Profile),
	}

	home, _ := os.UserHomeDir()
	fullKey := filepath.Join(home, ".ssh", "id_test")

	// 1. Add
	err := cfg.Add(Profile{
		Name:    "test",
		Key:     fullKey,
		Email:   "test@domain.com",
		GitName: "Tester",
	}, false)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}

	// Verify key was relativized
	p := cfg.Profiles["test"]
	if p.Key != "~/.ssh/id_test" {
		t.Errorf("expected key to be relativized to ~/.ssh/id_test, got %s", p.Key)
	}

	// Add duplicate without force should fail
	err = cfg.Add(Profile{Name: "test", Key: fullKey}, false)
	if err == nil {
		t.Fatal("expected error adding existing profile without force")
	}

	// Add duplicate with force should succeed
	err = cfg.Add(Profile{Name: "test", Key: fullKey, GitName: "Tester Updated"}, true)
	if err != nil {
		t.Fatalf("expected add with force to succeed, got: %v", err)
	}
	if cfg.Profiles["test"].GitName != "Tester Updated" {
		t.Errorf("expected updated name, got %s", cfg.Profiles["test"].GitName)
	}

	// 2. SetDefault
	if err := cfg.SetDefault("unknown"); err == nil {
		t.Fatal("expected error setting non-existent profile as default")
	}
	if err := cfg.SetDefault("test"); err != nil {
		t.Fatalf("SetDefault error: %v", err)
	}
	if cfg.Default != "test" {
		t.Errorf("expected default 'test', got %q", cfg.Default)
	}

	// 3. Resolve
	resolved, err := cfg.Resolve("test")
	if err != nil {
		t.Fatalf("Resolve explicit error: %v", err)
	}
	if resolved.Name != "test" {
		t.Errorf("expected resolved Name 'test', got %s", resolved.Name)
	}

	// Resolve with empty string uses default
	resolvedDef, err := cfg.Resolve("")
	if err != nil {
		t.Fatalf("Resolve default error: %v", err)
	}
	if resolvedDef.Name != "test" {
		t.Errorf("expected resolved default 'test', got %s", resolvedDef.Name)
	}

	// 4. Remove
	wasDefault, err := cfg.Remove("test")
	if err != nil {
		t.Fatalf("Remove error: %v", err)
	}
	if !wasDefault {
		t.Error("expected wasDefault to be true")
	}
	if cfg.Default != "" {
		t.Errorf("expected Default to be reset to empty, got %q", cfg.Default)
	}
	if _, exists := cfg.Profiles["test"]; exists {
		t.Error("expected profile 'test' to be removed from map")
	}

	// Resolve when no default and empty name should fail
	_, err = cfg.Resolve("")
	if err == nil {
		t.Fatal("expected error resolving empty name when no default set")
	}
}

func TestEndToEndRealDiskWorkflow(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "cssh_config")
	t.Setenv("CSSH_CONFIG_DIR", configDir)

	// 1. Agregar perfil personal
	home, _ := os.UserHomeDir()
	err := Add(Profile{
		Name:    "personal",
		Key:     filepath.Join(home, ".ssh", "id_ed25519_pers"),
		Email:   "elmer@personal.dev",
		GitName: "Elmer Personal",
		Hosts:   []string{"github.com", "gitlab.com"},
	}, false)
	if err != nil {
		t.Fatalf("Add personal error: %v", err)
	}

	// 2. Agregar perfil empresa
	err = Add(Profile{
		Name:    "empresa",
		Key:     filepath.Join(home, ".ssh", "id_rsa_work"),
		Email:   "elmer@company.com",
		GitName: "Elmer Work",
		Hosts:   []string{"corp.git.com"},
	}, false)
	if err != nil {
		t.Fatalf("Add empresa error: %v", err)
	}

	// 3. Establecer default
	if err := SetDefault("personal"); err != nil {
		t.Fatalf("SetDefault error: %v", err)
	}

	// 4. Inspeccionar archivo en disco y permisos
	targetPath := filepath.Join(configDir, "profiles.toml")
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	t.Logf("\n--- Contenido real generado en %s ---\n%s---", targetPath, string(content))

	// 5. Resolver default
	res, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve default error: %v", err)
	}
	t.Logf("Perfil Default resuelto con éxito -> Name: %s, Key: %s, GitName: %s, Email: %s",
		res.Name, res.Key, res.GitName, res.Email)

	// 6. Eliminar default y verificar
	wasDefault, err := Remove("personal")
	if err != nil {
		t.Fatalf("Remove error: %v", err)
	}
	if !wasDefault {
		t.Errorf("se esperaba wasDefault=true")
	}

	contentAfter, _ := os.ReadFile(targetPath)
	t.Logf("\n--- Contenido real tras eliminar 'personal' ---\n%s---", string(contentAfter))
}
