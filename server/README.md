# AutoStrike Server

Backend Go du projet AutoStrike - Plateforme BAS (Breach and Attack Simulation).

## Architecture

Le serveur suit une **architecture hexagonale** (Ports & Adapters) :

```
server/
├── cmd/autostrike/          # Point d'entrée
│   └── main.go
├── config/                  # Configuration
│   └── config.yaml
├── internal/
│   ├── domain/              # Couche métier (indépendante)
│   │   ├── entity/          # Entités: Agent, Technique, Scenario, Result
│   │   ├── repository/      # Interfaces (ports sortants)
│   │   ├── service/         # Services domaine: Orchestrator, Validator, ScoreCalculator
│   │   └── valueobject/     # Objets valeur
│   ├── application/         # Cas d'utilisation
│   │   ├── agent_service.go
│   │   ├── execution_service.go
│   │   ├── scenario_service.go
│   │   └── technique_service.go
│   └── infrastructure/      # Adaptateurs externes
│       ├── api/rest/        # Serveur REST (Gin)
│       ├── http/
│       │   ├── handlers/    # Handlers HTTP
│       │   └── middleware/  # Auth JWT, Logging
│       ├── persistence/
│       │   └── sqlite/      # Implémentation SQLite
│       └── websocket/       # Communication agents
└── go.mod
```

## Prérequis

- Go 1.21+
- SQLite3
- OpenSSL (pour les certificats)

## Installation

```bash
# Installer les dépendances
go mod download

# Générer les certificats TLS
cd .. && ./scripts/generate-certs.sh ./certs

# Créer le dossier data
mkdir -p data
```

## Configuration

Éditer `config/config.yaml` :

```yaml
server:
  host: "0.0.0.0"
  port: 8443

database:
  path: "./data/autostrike.db"

security:
  jwt_secret: "${JWT_SECRET}"      # Variable d'environnement
  agent_secret: "${AGENT_SECRET}"
  tls:
    enabled: true
    cert_file: "./certs/server.crt"
    key_file: "./certs/server.key"
    ca_file: "./certs/ca.crt"
    mtls: true
```

## Lancement

```bash
# Mode développement
go run ./cmd/autostrike

# Mode production
go build -o autostrike ./cmd/autostrike
./autostrike
```

## API REST

Base URL: `https://localhost:8443/api/v1`

### Authentication (routes publiques)
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| POST | `/auth/login` | Login avec username/password |
| POST | `/auth/refresh` | Rafraîchir le token d'accès |
| POST | `/auth/logout` | Invalider les tokens |
| GET | `/auth/me` | Infos utilisateur courant (requiert token) |

### Agents
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/agents` | Liste tous les agents |
| GET | `/agents/:paw` | Détails d'un agent |
| POST | `/agents` | Enregistrer un agent |
| DELETE | `/agents/:paw` | Supprimer un agent |
| POST | `/agents/:paw/heartbeat` | Heartbeat agent |

### Techniques
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/techniques` | Liste toutes les techniques |
| GET | `/techniques/:id` | Détails d'une technique |
| GET | `/techniques/tactic/:tactic` | Techniques par tactique MITRE |
| GET | `/techniques/platform/:platform` | Techniques par plateforme |
| GET | `/techniques/coverage` | Statistiques de couverture |
| POST | `/techniques/import` | Importer depuis YAML |

### Scénarios
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/scenarios` | Liste des scénarios |
| GET | `/scenarios/:id` | Détails d'un scénario |
| GET | `/scenarios/tag/:tag` | Scénarios par tag |
| POST | `/scenarios` | Créer un scénario |
| PUT | `/scenarios/:id` | Modifier un scénario |
| DELETE | `/scenarios/:id` | Supprimer un scénario |

### Exécutions
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/executions` | Liste des exécutions récentes |
| GET | `/executions/:id` | Détails d'une exécution |
| GET | `/executions/:id/results` | Résultats d'une exécution |
| POST | `/executions` | Démarrer une exécution |
| POST | `/executions/:id/stop` | Arrêter une exécution |
| POST | `/executions/:id/complete` | Terminer une exécution |

### Schedules (Planification)
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/schedules` | Liste des planifications |
| GET | `/schedules/:id` | Détails d'une planification |
| GET | `/schedules/:id/runs` | Historique des exécutions |
| POST | `/schedules` | Créer une planification |
| PUT | `/schedules/:id` | Modifier une planification |
| DELETE | `/schedules/:id` | Supprimer une planification |
| POST | `/schedules/:id/pause` | Mettre en pause |
| POST | `/schedules/:id/resume` | Reprendre |

### Notifications
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/notifications` | Liste des notifications |
| GET | `/notifications/settings` | Paramètres de notification |
| PUT | `/notifications/settings` | Modifier les paramètres |
| POST | `/notifications/:id/read` | Marquer comme lue |
| POST | `/notifications/read-all` | Tout marquer comme lu |

### Analytics
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/analytics/trend` | Tendance du score |
| GET | `/analytics/compare` | Comparaison de périodes |
| GET | `/analytics/summary` | Résumé des exécutions |

### Admin (Utilisateurs)
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/admin/users` | Liste des utilisateurs |
| GET | `/admin/users/:id` | Détails d'un utilisateur |
| POST | `/admin/users` | Créer un utilisateur |
| PUT | `/admin/users/:id` | Modifier un utilisateur |
| DELETE | `/admin/users/:id` | Supprimer un utilisateur |
| POST | `/admin/users/:id/activate` | Activer un utilisateur |
| POST | `/admin/users/:id/deactivate` | Désactiver un utilisateur |

### Permissions
| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/permissions/matrix` | Matrice des permissions |
| GET | `/permissions/me` | Mes permissions |

## WebSocket

Endpoint: `wss://localhost:8443/ws/agent`

### Messages Agent → Server
```json
{"type": "register", "payload": {"paw": "...", "hostname": "...", "platform": "..."}}
{"type": "heartbeat", "payload": {"paw": "..."}}
{"type": "task_result", "payload": {"task_id": "...", "success": true, "output": "..."}}
```

### Messages Server → Agent
```json
{"type": "task", "payload": {"id": "...", "technique_id": "T1082", "command": "...", "timeout": 300}}
{"type": "ping", "payload": {}}
```

## Score de Sécurité

Formule : `(blocked × 100 + detected × 50) / (total × 100) × 100`

| Statut | Points |
|--------|--------|
| Blocked | 100 |
| Detected | 50 |
| Successful | 0 |

## Tests

```bash
go test -v ./...
```

## Docker

```bash
docker build -t autostrike-server .
docker run -p 8443:8443 -v $(pwd)/certs:/app/certs autostrike-server
```
