---
tipo: "vision"
id: "VA-1"
version: 0.1
descripcion: "MVP - CRUD de perfiles SSH más un clonador de repos que ata cada repo a su perfil"
naturaleza: "proyecto"
fecha: 22/11/2026
estado: "borrador"
---

# CSSH

## QUE ES?

CSSH en su MVP es un CRUD de perfiles SSH más un clonador de repos que ata cada repo a su perfil.

CSSH es un gestor de perfiles SSH y una sintaxis corta para usar esos perfiles en cualquier herramienta que hable SSH — con el caso Git como el más importante, porque ahí la elección de perfil queda grabada en el propio repositorio.

## COMO SE USA?


| Comando | Qué hace |
|---|---|
| `cssh profile add <perfil> [flags]` | Crea perfil (genera/registra clave) |
| `cssh profile list` | Lista perfiles |
| `cssh profile show <perfil>` | Detalle de un perfil |
| `cssh profile default <perfil>` | Marca perfil como default |
| `cssh profile remove <perfil> [--force]` | Elimina perfil del TOML |
| `cssh rclone [<perfil>] <url> [flags]` | Clona repo atado a un perfil |

Seis comandos. Ese es el MVP completo en modo no interactivo.


```mermaid

sequenceDiagram
    actor U as Usuario
    participant C as cssh
    participant FS as ~/.ssh + ~/.config
    participant G as GitHub
    participant R as Orquidea/

    U->>C: 1. profile add personal
    C->>FS: genera id_ed25519_personal + profiles.toml
    U->>G: 2. pega .pub en Settings → SSH keys
    U->>C: 3. rclone personal git@github.com:.../Orquidea.git
    C->>G: ssh -T git@github.com (fail-fast)
    G-->>C: OK
    C->>G: git clone (GIT_SSH_COMMAND)
    C->>R: git config core.sshCommand + user.*
    U->>R: 4. git pull → clave personal

```

## QUE NO ES?

- No es "solo para repos". (MVP)
    - No toca ~/.ssh/config ni ~/.gitconfig.
- No es un reemplazo de SSH.
- No es un llavero de claves suelto.
- No es un gestor de ssh-agent.



