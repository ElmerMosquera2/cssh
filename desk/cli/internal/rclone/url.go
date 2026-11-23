package rclone

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// GitURLInfo holds parsed components of a Git SSH URL.
type GitURLInfo struct {
	Raw      string
	User     string
	Host     string
	Port     string
	Path     string
	RepoName string
}

// scpRegex matches SCP-like git URLs: [user@]host:path/to/repo.git
var scpRegex = regexp.MustCompile(`^(?:([^@:]+)@)?([^:]+):(.+)$`)

// ParseGitURL parses SSH Git URLs in both SCP-like syntax (git@github.com:owner/repo.git)
// and URI syntax (ssh://git@github.com:22/owner/repo.git).
func ParseGitURL(rawURL string) (*GitURLInfo, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil, fmt.Errorf("la URL del repositorio no puede estar vacía")
	}

	info := &GitURLInfo{Raw: trimmed}

	// 1. URI syntax: ssh://...
	if strings.HasPrefix(trimmed, "ssh://") {
		u, err := url.Parse(trimmed)
		if err != nil {
			return nil, fmt.Errorf("URL SSH inválida: %w", err)
		}
		if u.User != nil {
			info.User = u.User.Username()
		}
		info.Host = u.Hostname()
		info.Port = u.Port()
		info.Path = strings.TrimPrefix(u.Path, "/")
	} else if matches := scpRegex.FindStringSubmatch(trimmed); len(matches) == 4 {
		// 2. SCP-like syntax: [user@]host:path
		info.User = matches[1]
		if info.User == "" {
			info.User = "git"
		}
		info.Host = matches[2]
		info.Path = matches[3]
	} else {
		// If it's another protocol (e.g. https:// or local path), try parsing or reject if SSH is expected
		if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
			u, err := url.Parse(trimmed)
			if err == nil {
				info.Host = u.Hostname()
				info.Path = strings.TrimPrefix(u.Path, "/")
			}
		} else {
			info.Path = trimmed
		}
	}

	// Calculate RepoName from Path
	cleanPath := strings.TrimSuffix(strings.TrimSuffix(info.Path, "/"), ".git")
	base := filepath.Base(cleanPath)
	if base == "." || base == "/" || base == "" {
		info.RepoName = "repo"
	} else {
		info.RepoName = base
	}

	return info, nil
}
