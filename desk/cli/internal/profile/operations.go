package profile

import (
	"fmt"
	"strings"
)

// Resolve resolves a profile by name.
// If name is empty, it resolves the default profile.
// Returns an error if the profile does not exist or if no default profile is configured.
func (c *Config) Resolve(name string) (Profile, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		if c.Default == "" {
			return Profile{}, fmt.Errorf("no se especificó perfil y no hay perfil default")
		}
		target = c.Default
	}

	p, exists := c.Profiles[target]
	if !exists {
		return Profile{}, fmt.Errorf("perfil %q no encontrado", target)
	}
	p.Name = target
	return p, nil
}

// Add adds or updates a profile in the configuration.
// If force is false and the profile already exists, it returns an error.
// The key path is automatically relativized to ~/ if under $HOME.
func (c *Config) Add(p Profile, force bool) error {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return fmt.Errorf("el nombre del perfil no puede estar vacío")
	}

	key := strings.TrimSpace(p.Key)
	if key == "" {
		return fmt.Errorf("perfil %q no tiene clave asignada", name)
	}

	// Rule 10: Convertir path bajo $HOME a convención ~/
	if relKey, err := RelativizeHome(key); err == nil {
		p.Key = relKey
	} else {
		p.Key = key
	}

	if c.Profiles == nil {
		c.Profiles = make(map[string]Profile)
	}

	if !force {
		if _, exists := c.Profiles[name]; exists {
			return fmt.Errorf("el perfil %q ya existe (usa --force para sobrescribir)", name)
		}
	}

	p.Name = name
	p.Email = strings.TrimSpace(p.Email)
	p.GitName = strings.TrimSpace(p.GitName)

	c.Profiles[name] = p
	return nil
}

// Remove removes a profile from the configuration.
// If the removed profile was the default profile, wasDefault is returned as true and Default is reset.
// Returns an error if the profile does not exist.
func (c *Config) Remove(name string) (wasDefault bool, err error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return false, fmt.Errorf("nombre de perfil no especificado")
	}

	if _, exists := c.Profiles[target]; !exists {
		return false, fmt.Errorf("perfil %q no existe", target)
	}

	if c.Default == target {
		wasDefault = true
		c.Default = ""
	}

	delete(c.Profiles, target)
	return wasDefault, nil
}

// SetDefault marks an existing profile as default.
func (c *Config) SetDefault(name string) error {
	target := strings.TrimSpace(name)
	if target == "" {
		c.Default = ""
		return nil
	}

	if _, exists := c.Profiles[target]; !exists {
		return fmt.Errorf("perfil %q no existe", target)
	}

	c.Default = target
	return nil
}

// Package-level convenience wrappers operating on default ConfigPath()

// Resolve resolves a profile from the default configuration.
func Resolve(name string) (Profile, error) {
	cfg, err := Load()
	if err != nil {
		return Profile{}, err
	}
	return cfg.Resolve(name)
}

// Add adds a profile to the default configuration and saves it.
func Add(p Profile, force bool) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if err := cfg.Add(p, force); err != nil {
		return err
	}
	return cfg.Save()
}

// Remove removes a profile from the default configuration and saves it.
func Remove(name string) (wasDefault bool, err error) {
	cfg, err := Load()
	if err != nil {
		return false, err
	}
	wasDefault, err = cfg.Remove(name)
	if err != nil {
		return false, err
	}
	if err := cfg.Save(); err != nil {
		return wasDefault, err
	}
	return wasDefault, nil
}

// SetDefault sets the default profile in the default configuration and saves it.
func SetDefault(name string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if err := cfg.SetDefault(name); err != nil {
		return err
	}
	return cfg.Save()
}
