package sshutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAndReadPublicKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519_test")

	// 1. Generate ed25519 key
	path, err := GenerateKey(KeyGenOptions{
		Type:    "ed25519",
		Path:    keyPath,
		Comment: "test@cssh.io",
		Force:   false,
	})
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	if path != keyPath {
		t.Errorf("expected path %s, got %s", keyPath, path)
	}

	// Verify files exist
	if !KeyExists(keyPath) {
		t.Errorf("KeyExists returned false for %s", keyPath)
	}
	if _, err := os.Stat(keyPath + ".pub"); err != nil {
		t.Errorf("public key file not created: %v", err)
	}

	// 2. Read public key
	pubKey, err := ReadPublicKey(keyPath)
	if err != nil {
		t.Fatalf("ReadPublicKey error: %v", err)
	}
	if !strings.HasPrefix(pubKey, "ssh-ed25519 ") {
		t.Errorf("expected public key to start with 'ssh-ed25519 ', got %q", pubKey)
	}
	if !strings.HasSuffix(pubKey, " test@cssh.io") {
		t.Errorf("expected public key to end with ' test@cssh.io', got %q", pubKey)
	}

	// 3. Duplicate without force should fail
	_, err = GenerateKey(KeyGenOptions{
		Type:  "ed25519",
		Path:  keyPath,
		Force: false,
	})
	if err == nil {
		t.Fatal("expected error generating existing key without force")
	}

	// 4. Duplicate with force should succeed
	_, err = GenerateKey(KeyGenOptions{
		Type:    "ed25519",
		Path:    keyPath,
		Comment: "updated@cssh.io",
		Force:   true,
	})
	if err != nil {
		t.Fatalf("expected GenerateKey with force to succeed, got: %v", err)
	}

	updatedPub, err := ReadPublicKey(keyPath)
	if err != nil {
		t.Fatalf("ReadPublicKey updated error: %v", err)
	}
	if !strings.HasSuffix(updatedPub, " updated@cssh.io") {
		t.Errorf("expected updated comment in public key, got %q", updatedPub)
	}
}

func TestUnsupportedKeyType(t *testing.T) {
	_, err := GenerateKey(KeyGenOptions{
		Type: "dsa",
		Path: filepath.Join(t.TempDir(), "key"),
	})
	if err == nil {
		t.Fatal("expected error for unsupported key type 'dsa'")
	}
}

func TestDefaultKeyPath(t *testing.T) {
	path, err := DefaultKeyPath("personal", "ed25519")
	if err != nil {
		t.Fatalf("DefaultKeyPath error: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".ssh", "id_ed25519_personal")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}
