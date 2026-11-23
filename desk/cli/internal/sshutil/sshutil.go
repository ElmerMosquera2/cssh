package sshutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
)

// SupportedKeyTypes lists valid SSH key types.
var SupportedKeyTypes = map[string]bool{
	"ed25519": true,
	"rsa":     true,
	"ecdsa":   true,
}

// KeyGenOptions contains options for generating an SSH key.
type KeyGenOptions struct {
	Type    string
	Path    string
	Comment string
	Force   bool
}

// DefaultKeyPath returns the conventional key path ~/.ssh/id_<type>_<profile>.
func DefaultKeyPath(profileName, keyType string) (string, error) {
	if keyType == "" {
		keyType = "ed25519"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("obtener home directory: %w", err)
	}
	fileName := fmt.Sprintf("id_%s_%s", keyType, profileName)
	return filepath.Join(home, ".ssh", fileName), nil
}

// GenerateKey generates an SSH keypair using ssh-keygen.
func GenerateKey(opts KeyGenOptions) (string, error) {
	keyType := strings.ToLower(strings.TrimSpace(opts.Type))
	if keyType == "" {
		keyType = "ed25519"
	}
	if !SupportedKeyTypes[keyType] {
		return "", fmt.Errorf("tipo de clave no soportado: %q (válidos: ed25519, rsa, ecdsa)", keyType)
	}

	absPath, err := profile.ExpandHome(opts.Path)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("crear directorio para clave %s: %w", dir, err)
	}

	// Check if key already exists
	if _, err := os.Stat(absPath); err == nil {
		if !opts.Force {
			return "", fmt.Errorf("el archivo de clave ya existe en %s (usa --force para sobrescribir)", absPath)
		}
		_ = os.Remove(absPath)
		_ = os.Remove(absPath + ".pub")
	}

	args := []string{
		"-q",
		"-t", keyType,
		"-f", absPath,
		"-N", "", // empty passphrase
	}
	if keyType == "rsa" {
		args = append(args, "-b", "4096")
	}
	if opts.Comment != "" {
		args = append(args, "-C", opts.Comment)
	}

	cmd := exec.Command("ssh-keygen", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("error al ejecutar ssh-keygen: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return absPath, nil
}

// ReadPublicKey reads the public key corresponding to a private key path.
// It checks <path>.pub, and if missing, attempts to extract it using ssh-keygen -y.
func ReadPublicKey(privateKeyPath string) (string, error) {
	absPath, err := profile.ExpandHome(privateKeyPath)
	if err != nil {
		return "", err
	}

	pubPath := absPath + ".pub"
	if data, err := os.ReadFile(pubPath); err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// Fallback to ssh-keygen -y -f <path>
	cmd := exec.Command("ssh-keygen", "-y", "-f", absPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("no se pudo leer la clave pública: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// KeyExists checks if a key exists on disk.
func KeyExists(path string) bool {
	absPath, err := profile.ExpandHome(path)
	if err != nil {
		return false
	}
	fi, err := os.Stat(absPath)
	return err == nil && !fi.IsDir()
}
