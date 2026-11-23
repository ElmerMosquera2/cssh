package profile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	// DefaultConfigFileName is the fixed configuration file name.
	DefaultConfigFileName = "profiles.toml"
	// DirPerms are the permissions required for the config directory.
	DirPerms = 0700
	// FilePerms are the permissions required for the config file.
	FilePerms = 0600
)

// ConfigDir returns the configuration directory path.
// Rule 1: Si CSSH_CONFIG_DIR está seteado, usa ese directorio.
// Si no, usa os.UserConfigDir()/cssh.
func ConfigDir() (string, error) {
	if env := os.Getenv("CSSH_CONFIG_DIR"); env != "" {
		return env, nil
	}
	userConfig, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("obtener UserConfigDir: %w", err)
	}
	return filepath.Join(userConfig, "cssh"), nil
}

// ConfigPath returns the full path to profiles.toml.
// Rule 1 & 2: Path único, determinista y nombre fijo profiles.toml.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultConfigFileName), nil
}

// Load loads the configuration from the standard ConfigPath().
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom loads the configuration from a specific file path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Rule 5: Lectura tolerante a archivo ausente.
		return &Config{
			Profiles: make(map[string]Profile),
			path:     path,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("leer archivo %s: %w", path, err)
	}

	// Rule 3: Validar permisos si el archivo existe (en sistemas no-Windows)
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(path); err == nil {
			perm := fi.Mode().Perm()
			if perm&0077 != 0 {
				fmt.Fprintf(os.Stderr, "advertencia: permisos de %s son más laxos que 0600 (%#o)\n", path, perm)
			}
		}
		dir := filepath.Dir(path)
		if fi, err := os.Stat(dir); err == nil {
			perm := fi.Mode().Perm()
			if perm&0077 != 0 {
				fmt.Fprintf(os.Stderr, "advertencia: permisos de directorio %s son más laxos que 0700 (%#o)\n", dir, perm)
			}
		}
	}

	// Rule 14: Strip UTF-8 BOM if present
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	// Raw struct para deserialización
	type rawConfig struct {
		Default  string             `toml:"default"`
		Profiles map[string]Profile `toml:"profiles"`
	}

	var raw rawConfig
	// Rule 6: Lectura estricta a archivo corrupto
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("error al parsear %s: %w", path, err)
	}

	if raw.Profiles == nil {
		raw.Profiles = make(map[string]Profile)
	}

	cfg := &Config{
		Default:  raw.Default,
		Profiles: raw.Profiles,
		path:     path,
	}

	// Rule 8 & Rule 13:
	// - Rule 8: key es obligatorio (falla si algún perfil no tiene key)
	// - Rule 13: Hidratación de Name desde la clave del mapa
	for name, p := range cfg.Profiles {
		if strings.TrimSpace(p.Key) == "" {
			return nil, fmt.Errorf("perfil %q no tiene clave asignada", name)
		}
		p.Name = name
		cfg.Profiles[name] = p
	}

	// Rule 12: Si default apunta a perfil inexistente, advertir por stderr y limpiar
	if cfg.Default != "" {
		if _, exists := cfg.Profiles[cfg.Default]; !exists {
			fmt.Fprintf(os.Stderr, "advertencia: perfil default %q no existe en %s\n", cfg.Default, path)
			cfg.Default = ""
		}
	}

	return cfg, nil
}

// Save saves the configuration to the path it was loaded from or to ConfigPath().
func (c *Config) Save() error {
	path := c.path
	if path == "" {
		var err error
		path, err = ConfigPath()
		if err != nil {
			return err
		}
		c.path = path
	}
	return c.SaveTo(path)
}

// SaveTo saves the configuration to a specific file path atomically.
func (c *Config) SaveTo(path string) error {
	// Rule 12: Validar default al guardar. Si no está vacío, debe existir.
	if c.Default != "" {
		if _, exists := c.Profiles[c.Default]; !exists {
			return fmt.Errorf("perfil default %q no existe", c.Default)
		}
	}

	// Validar que todos los perfiles tengan key
	for name, p := range c.Profiles {
		if strings.TrimSpace(p.Key) == "" {
			return fmt.Errorf("perfil %q no tiene clave asignada", name)
		}
	}

	dir := filepath.Dir(path)
	// Rule 3: Permisos directorio 0700 al crear
	if err := os.MkdirAll(dir, DirPerms); err != nil {
		return fmt.Errorf("crear directorio de configuración %s: %w", dir, err)
	}

	// Determinar modo de archivo: 0600 al crear, o preservar si ya existía
	fileMode := os.FileMode(FilePerms)
	if fi, err := os.Stat(path); err == nil {
		fileMode = fi.Mode().Perm()
	}

	// Rule 4: Escritura atómica vía <path>.tmp -> fsync -> Rename
	tmpPath := path + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("crear archivo temporal %s: %w", tmpPath, err)
	}

	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	// Rule 7 & Rule 15: Esquema cerrado e Idempotencia de guardado
	enc := toml.NewEncoder(tmpFile)
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("serializar TOML: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sincronizar archivo temporal (fsync): %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("cerrar archivo temporal: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		if runtime.GOOS == "windows" {
			_ = os.Remove(path)
			if err := os.Rename(tmpPath, path); err != nil {
				return fmt.Errorf("reemplazar archivo de destino en windows: %w", err)
			}
		} else {
			return fmt.Errorf("reemplazar archivo de destino: %w", err)
		}
	}

	success = true
	c.path = path
	return nil
}
