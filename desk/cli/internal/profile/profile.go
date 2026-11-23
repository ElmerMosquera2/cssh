package profile

// Profile represents an SSH profile stored in profiles.toml.
type Profile struct {
	Name    string   `toml:"-"`              // Identificador del perfil (clave del mapa en TOML, hidratado al cargar)
	Key     string   `toml:"key"`            // Ruta a la clave SSH (obligatorio, guardado con ~/)
	Email   string   `toml:"email,omitempty"`// Opcional: configurado como user.email en git
	GitName string   `toml:"name,omitempty"` // Opcional: configurado como user.name en git
	Hosts   []string `toml:"hosts,omitempty"`// Opcional: lista de hosts asociados
}

// Config represents the top-level configuration in profiles.toml.
type Config struct {
	Default  string             `toml:"default,omitempty"`
	Profiles map[string]Profile `toml:"profiles,omitempty"`

	// path guarda la ruta del archivo del cual fue cargado o hacia el cual se guardará
	path string
}
