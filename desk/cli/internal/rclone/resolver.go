package rclone

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
)

// ResolveProfile resolves which profile to use for cloning according to the resolution table:
// 1. Explicit profile wins: If explicitProfile is provided, it must exist.
// 2. Single host match: If not explicit and exactly one profile claims the URL host, use it.
// 3. Ambiguity error: If multiple profiles claim the URL host, fail asking for explicit profile.
// 4. Default fallback: If no profile claims the URL host, fallback to default profile.
func ResolveProfile(cfg *profile.Config, explicitProfile string, rawURL string) (profile.Profile, error) {
	explicit := strings.TrimSpace(explicitProfile)

	// 1. Explícito gana
	if explicit != "" {
		p, exists := cfg.Profiles[explicit]
		if !exists {
			return profile.Profile{}, fmt.Errorf("perfil explícito %q no encontrado en la configuración", explicit)
		}
		p.Name = explicit
		return p, nil
	}

	urlInfo, err := ParseGitURL(rawURL)
	if err != nil {
		return profile.Profile{}, err
	}

	host := strings.ToLower(strings.TrimSpace(urlInfo.Host))

	var matchingProfiles []string
	if host != "" {
		for name, p := range cfg.Profiles {
			for _, h := range p.Hosts {
				if strings.ToLower(strings.TrimSpace(h)) == host {
					matchingProfiles = append(matchingProfiles, name)
					break
				}
			}
		}
	}

	sort.Strings(matchingProfiles)

	// 2. Match único
	if len(matchingProfiles) == 1 {
		matchedName := matchingProfiles[0]
		p := cfg.Profiles[matchedName]
		p.Name = matchedName
		return p, nil
	}

	// 3. Ambigüedad
	if len(matchingProfiles) > 1 {
		return profile.Profile{}, fmt.Errorf(
			"múltiples perfiles reclaman el host %q (%s); especifica el perfil explícitamente: cssh rclone <perfil> <url>",
			host, strings.Join(matchingProfiles, ", "),
		)
	}

	// 4. Fallback a Default
	if cfg.Default != "" {
		p, exists := cfg.Profiles[cfg.Default]
		if exists {
			p.Name = cfg.Default
			return p, nil
		}
	}

	if host != "" {
		return profile.Profile{}, fmt.Errorf(
			"no se pudo determinar el perfil para %q: ningún perfil reclama el host %q y no hay perfil default",
			rawURL, host,
		)
	}

	return profile.Profile{}, fmt.Errorf("no se especificó perfil y no hay perfil default configurado")
}
