---
tipo: "dominio"
id: "dominio-1"
version: 0.1
descripcion: "dentidad del sistema en lo que respecta a dónde y cómo CSSH persiste los perfiles. Delimita qué es válido escribir, leer y asumir sobre profiles.toml."
estado: "delimitacion"
---

## Reglas y restricciones

1. **Path único y determinista.** El archivo vive en exactamente un lugar: si `CSSH_CONFIG_DIR` está seteado, en `$CSSH_CONFIG_DIR/profiles.toml`; si no, en `os.UserConfigDir()/cssh/profiles.toml`. No hay búsqueda en múltiples rutas, no hay fallback silencioso.

2. **Nombre fijo del archivo.** Siempre `profiles.toml`. No configurable, no por perfil.

3. **Permisos.** Directorio `0700`, archivo `0600`. Se aplican al crear; no se re-aplican si ya existen (se respeta lo que el usuario haya cambiado). Si los permisos son más laxos al cargar, se advierte por stderr, no se corrige solo.

4. **Escritura atómica.** Siempre a `<path>.tmp` → `fsync` → `os.Rename`. En Windows, si `Rename` falla porque el destino existe, `os.Remove` del destino + `Rename`. Nunca escritura in-place. Ctrl+C a mitad de escritura no deja el TOML corrupto.

5. **Lectura tolerante a archivo ausente.** `Load()` sobre archivo inexistente devuelve `Config{Profiles: map{}}` sin error. El primer `Save()` crea el archivo. "No existe" no es error, es estado inicial.

6. **Lectura estricta a archivo corrupto.** TOML inválido → error con path y detalle. **No** se intenta reparar, no se sobreescribe, no se ignora. El usuario decide.

7. **Esquema cerrado.** El TOML solo contiene lo que el esquema define:

   ```toml
   default = "<perfil>"            # opcional, string

   [profiles.<nombre>]
     key   = "~/..."               # obligatorio, string no vacío
     email = "..."                 # opcional, string
     name  = "..."                 # opcional, string
     hosts = ["...", "..."]        # opcional, lista de strings
   ```

   Cualquier otra clave se ignora al leer y no se escribe al guardar.

8. **`key` es obligatorio.** Un perfil sin `key` no es un perfil válido. `Load()` falla con mensaje claro: `"perfil %q no tiene clave asignada"`.

9. **`email`, `name`, `hosts` son opcionales.** Un perfil SSH puro (VPS, servidor) no tiene email ni name. Es válido. La ausencia no es error.

10. **Convención `~/`.** Los paths bajo `$HOME` se guardan con `~/` para portabilidad entre máquinas y dotfiles. Paths fuera de `$HOME` se guardan absolutos. Se expande al usar, nunca al guardar.

11. **El TOML no contiene secretos.** Nunca se escribe la clave privada, ni passphrase, ni token. Solo la **ruta** a la clave. `profiles.toml` puede versionarse en dotfiles públicos sin riesgo.

12. **`default` válido al guardar.** Si `Default` no está vacío, debe apuntar a un perfil existente. Si no → error. Se valida al guardar, no al cargar (al cargar, si apunta a inexistente, se advierte y se pone `Default = ""`).

13. **Hidratación de `Name`.** El TOML no serializa el nombre del perfil (es la clave del mapa). Al cargar, cada `Profile.Name` se setea desde la clave del mapa. `toml:"-"`.

14. **BOM tolerado al leer.** Si el archivo empieza con BOM UTF-8 (editores Windows), se strip-ea antes de parsear. `BurntSushi/toml` no lo tolera solo.

15. **Idempotencia de `Save`.** Guardar dos veces el mismo `Config` produce el mismo archivo (mismo orden de claves si se puede, mismo contenido si no).

16. **Sin locking entre procesos en el MVP.** Dos `cssh` concurrentes sobre el mismo TOML pueden pisarse. Se documenta; no se resuelve. La escritura atómica evita corrupción, no pérdida de última escritura.

---

## Qué es

El **contrato de datos** de CSSH. Todo lo demás (`profile add`, `rclone`, `profile list/show/default/remove`) lee y escribe a través de este dominio. Es la **única fuente de verdad** de qué perfiles existen y qué contienen.

**Propósito:** que el usuario pueda **leer, editar y versionar** sus perfiles como un archivo de texto plano, sin depender de la herramienta para entender su estado. Si CSSH desaparece mañana, el TOML sigue siendo legible y parseable.

**Propiedades que garantiza:**

- **Legibilidad:** un humano entiende el archivo sin documentación.
- **Portabilidad:** `~/` permite mover el TOML entre máquinas y que siga funcionando.
- **Seguridad:** `0600` + cero secretos + paths bajo `$HOME` como convención.
- **Robustez:** escritura atómica + validación en carga y guardado.
- **Determinismo:** un único path, un único esquema, cero ambigüedad.

**Superficie pública del dominio** (lo que expone a los demás):

- `Load() (*Config, error)` / `(*Config) Save() error`
- `Resolve(name) (Profile, error)`
- `Add(p Profile, force bool) error`
- `Remove(name) (wasDefault bool, err error)`
- `SetDefault(name) error`
- `ExpandHome(path) (string, error)` / `RelativizeHome(path) (string, error)`
- `ConfigPath() (string, error)`

Todo lo demás (structs, paths internos, manejo de BOM, rename en Windows) es **implementación**, no dominio.

---

## Qué no cubre (libertades)

Este dominio **no** define, no restringe y no prescribe:

1. **Cómo se generan las claves SSH.** El dominio del TOML no sabe si la clave se generó con `ssh-keygen`, si vino de un YubiKey, o si fue copiada a mano. Solo registra la ruta.

2. **Cómo se valida la conexión SSH.** `ssh -T`, `nc`, o ninguna verificación — el TOML no opina.

3. **Cómo se ata el repo al perfil.** `core.sshCommand`, `includeIf`, `known_hosts` — fuera de este dominio.

4. **Cómo se resuelve el perfil en `rclone`.** La tabla de resolución (explícito gana, match único, ambigüedad, default) vive en otro dominio. El TOML solo provee los datos.

5. **Qué herramientas consumen los perfiles.** `git`, `scp`, `ssh`, `rsync` — el TOML es agnóstico del consumidor.

6. **Cómo se edita el TOML.** A mano, con `$EDITOR`, con `cssh profile edit` (fase 2) — el dominio garantiza que el archivo es editable, no cómo.

7. **Versionado del esquema.** Hoy es v1 implícito. Migraciones futuras (`v2`, `version = 2`) son de otro dominio.

8. **Cifrado del TOML.** El archivo es texto plano por diseño (no contiene secretos). Si algún día contiene algo sensible, eso es otro dominio.

9. **Concurrencia multi-proceso.** Sin locking en MVP. Documentado, no resuelto.

10. **Multi-usuario / multi-tenant.** Cada usuario tiene su `$XDG_CONFIG_HOME` / `%APPDATA%`. El dominio no define perfiles compartidos.

11. **UI de edición.** Sin TUI, sin GUI, sin editor embebido. El dominio solo lee/escribe bytes.

12. **Backup automático.** Sin rotación, sin `.bak`. La escritura atómica es el único mecanismo de integridad.
