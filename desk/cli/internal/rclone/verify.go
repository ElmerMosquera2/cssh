package rclone

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
)

// VerifySSH performs a fail-fast SSH check against the remote host using the specified key.
func VerifySSH(keyPath, user, host string) error {
	absKey, err := profile.ExpandHome(keyPath)
	if err != nil {
		return err
	}

	target := host
	if user != "" {
		target = user + "@" + host
	}

	cmd := exec.Command("ssh", "-T",
		"-i", absKey,
		"-o", "IdentitiesOnly=yes",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
		target,
	)

	out, err := cmd.CombinedOutput()
	outputStr := string(out)

	if err == nil {
		return nil
	}

	// Known git forge greeting messages that might exit with non-zero when no shell is granted:
	// GitHub: "Hi <user>! You've successfully authenticated, but GitHub does not provide shell access."
	// GitLab: "Welcome to GitLab, @<user>!"
	// Bitbucket: "logged in as <user>."
	lowerOut := strings.ToLower(outputStr)
	if strings.Contains(lowerOut, "successfully authenticated") ||
		strings.Contains(lowerOut, "welcome to gitlab") ||
		strings.Contains(lowerOut, "logged in as") {
		return nil
	}

	return fmt.Errorf("verificación SSH falló para %s: %s", target, strings.TrimSpace(outputStr))
}
