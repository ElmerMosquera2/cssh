package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandHome expands leading "~/" or "~" to the user's home directory.
// Non-tilde paths are returned cleaned.
func ExpandHome(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("obtener home directory: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("obtener home directory: %w", err)
		}
		return filepath.Join(home, filepath.FromSlash(path[2:])), nil
	}
	return filepath.Clean(path), nil
}

// RelativizeHome converts paths inside the user's home directory to use "~/".
// Paths outside $HOME are returned as absolute paths.
// Convention: Always uses forward slashes in "~/..." for cross-platform portability.
func RelativizeHome(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		return "~/" + filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(path, "~/"), "~\\")), nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return path, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return absPath, nil
	}

	cleanAbs := filepath.Clean(absPath)
	cleanHome := filepath.Clean(home)

	if cleanAbs == cleanHome {
		return "~", nil
	}

	sep := string(filepath.Separator)
	if strings.HasPrefix(cleanAbs, cleanHome+sep) {
		rel, err := filepath.Rel(cleanHome, cleanAbs)
		if err != nil {
			return absPath, nil
		}
		return "~/" + filepath.ToSlash(rel), nil
	}

	return cleanAbs, nil
}
