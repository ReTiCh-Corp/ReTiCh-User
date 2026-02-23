# ReTiCh User Service

Service de gestion des utilisateurs pour la plateforme ReTiCh. Gère les profils, les contacts et les préférences.

## Fonctionnalités

- Gestion des profils utilisateurs
- Paramètres et préférences
- Liste de contacts / amis
- Demandes d'amis
- Blocage d'utilisateurs
- Statut en ligne (online/offline/away/busy)

## Prérequis

- Go 1.22+
- PostgreSQL 16+
- Redis (pour le cache)
- Docker (optionnel)

## Démarrage rapide

### Avec Docker (recommandé)

```bash
# Depuis le repo ReTiCh-Infrastucture
make up
make migrate-user
```

### Sans Docker

```bash
# Installer les dépendances
go mod download

# Configurer la base de données
export DATABASE_URL="postgres://retich:retich_secret@localhost:5433/retich_users?sslmode=disable"

# Lancer les migrations
migrate -path migrations -database "$DATABASE_URL" up

# Lancer le serveur
go run cmd/server/main.go
```

### Développement avec hot-reload

```bash
# Installer Air
go install github.com/air-verse/air@latest

# Lancer avec hot-reload
air -c .air.toml
```

## Configuration

Variables d'environnement:

| Variable | Description | Défaut |
|----------|-------------|--------|
| `PORT` | Port du serveur | `8083` |
| `DATABASE_URL` | URL PostgreSQL | - |
| `REDIS_URL` | URL Redis | `redis:6379` |
| `LOG_LEVEL` | Niveau de log | `info` |

## Endpoints

| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |
| GET | `/users/:id` | Obtenir un profil |
| PUT | `/users/:id` | Modifier son profil |
| GET | `/users/:id/settings` | Obtenir ses paramètres |
| PUT | `/users/:id/settings` | Modifier ses paramètres |
| GET | `/users/:id/contacts` | Liste des contacts |
| POST | `/users/:id/contacts` | Ajouter un contact |
| DELETE | `/users/:id/contacts/:contactId` | Supprimer un contact |
| GET | `/users/:id/friend-requests` | Demandes d'amis reçues |
| POST | `/friend-requests` | Envoyer une demande |
| PUT | `/friend-requests/:id` | Accepter/Refuser |
| POST | `/users/:id/block` | Bloquer un utilisateur |
| DELETE | `/users/:id/block/:blockedId` | Débloquer |

## Base de données

### Schéma

```
profiles
├── id (UUID, PK - même que auth.users.id)
├── username (UNIQUE)
├── display_name
├── avatar_url
├── bio
├── status (online/offline/away/busy/invisible)
├── custom_status
├── last_seen_at
└── timestamps

user_settings
├── user_id (PK, FK → profiles)
├── theme (light/dark/system)
├── language
├── timezone
├── notifications_enabled
├── sound_enabled
├── desktop_notifications
├── email_notifications
├── show_online_status
└── show_read_receipts

contacts
├── user_id (FK → profiles)
├── contact_id (FK → profiles)
├── nickname
├── is_favorite
└── is_blocked

friend_requests
├── sender_id (FK → profiles)
├── receiver_id (FK → profiles)
├── status (pending/accepted/rejected/cancelled)
└── message

blocked_users
├── user_id (FK → profiles)
├── blocked_user_id (FK → profiles)
└── reason
```

### Migrations

```bash
# Appliquer les migrations
migrate -path migrations -database "$DATABASE_URL" up

# Rollback
migrate -path migrations -database "$DATABASE_URL" down 1
```

## Structure du projet

```
ReTiCh-User/
├── cmd/
│   └── server/
│       └── main.go         # Point d'entrée
├── internal/               # Code interne
├── migrations/
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── Dockerfile              # Image production
├── Dockerfile.dev          # Image développement
├── .air.toml               # Config hot-reload
├── go.mod
└── go.sum
```

## Tests

```bash
# Lancer les tests
go test ./...

# Avec couverture
go test -cover ./...
```

## Licence

MIT
