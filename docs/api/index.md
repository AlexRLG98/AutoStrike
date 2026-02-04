# API

Documentation de l'API REST AutoStrike.

---

## Base URL

```
https://server:8443/api/v1
```

---

## Authentification

L'authentification JWT est **optionnelle** par défaut. Elle est activée uniquement si `JWT_SECRET` est défini dans `.env`.

### Sans authentification (développement)

```bash
curl https://localhost:8443/api/v1/agents
```

### Avec authentification (production)

Quand `JWT_SECRET` est défini, utilisez l'endpoint `/auth/login` pour obtenir un token :

```bash
# Login
curl -X POST https://localhost:8443/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'

# Utiliser le token
curl https://localhost:8443/api/v1/agents \
  -H "Authorization: Bearer <access_token>"
```

**Identifiants par défaut**: `admin / admin123`

---

## Endpoints d'authentification

| Méthode | Endpoint | Description |
|---------|----------|-------------|
| POST | `/auth/login` | Login (retourne access_token + refresh_token) |
| POST | `/auth/refresh` | Rafraîchir le token d'accès |
| POST | `/auth/logout` | Invalider les tokens |
| GET | `/auth/me` | Infos utilisateur courant |

---

## Endpoints principaux

| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/agents` | Liste des agents |
| GET | `/techniques` | Liste des techniques MITRE |
| GET | `/scenarios` | Liste des scénarios |
| POST | `/executions` | Lancer une exécution |
| GET | `/executions/:id` | Détails d'une exécution |

---

## WebSocket

| Path | Description |
|------|-------------|
| `/ws/agent` | Connexion agent |
| `/ws/dashboard` | Mises à jour temps réel |

---

## Documentation complète

Voir la [Référence API](reference.md) pour tous les endpoints, paramètres et exemples.
