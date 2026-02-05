# AutoStrike

**Plateforme de Breach and Attack Simulation (BAS) basée sur MITRE ATT&CK**

> Projet EIP EPITECH - Promotion 2028

---

## Description

AutoStrike est une plateforme open-source de validation continue des défenses de sécurité par simulation d'attaques. Elle permet aux équipes SOC et RSSI de tester automatiquement leurs capacités de détection en utilisant le framework MITRE ATT&CK.

## Fonctionnalités

- **15 techniques MITRE ATT&CK** - Discovery, Execution, Persistence, Defense Evasion
- **Matrice MITRE interactive** - Visualisation de la couverture de détection
- **Scénarios d'attaque** - Exécution automatisée de techniques
- **Agents multi-plateformes** - Windows, Linux et macOS
- **Dashboard temps réel** - Mises à jour WebSocket instantanées
- **Safe Mode** - Toutes les techniques sont non-destructives
- **Score de sécurité** - Évaluation automatique des défenses

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Dashboard (React + TypeScript)                │
│                         https://server:8443                      │
└─────────────────────────────────────────────────────────────────┘
                                │
                    HTTPS + WebSocket (port 8443)
                                │
┌─────────────────────────────────────────────────────────────────┐
│                     Control Server (Go + Gin)                    │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐    │
│   │  REST API   │  │  WebSocket  │  │  Static Dashboard   │    │
│   │  /api/v1/*  │  │  /ws/*      │  │  /                  │    │
│   └─────────────┘  └─────────────┘  └─────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                          WebSocket (TLS)
                                │
            ┌───────────────────┼───────────────────┐
            ▼                   ▼                   ▼
      ┌──────────┐        ┌──────────┐        ┌──────────┐
      │  Agent   │        │  Agent   │        │  Agent   │
      │ Windows  │        │  Linux   │        │  macOS   │
      │  (Rust)  │        │  (Rust)  │        │  (Rust)  │
      └──────────┘        └──────────┘        └──────────┘
```

## Stack Technique

| Composant | Technologie |
|-----------|-------------|
| Frontend | React 18, TypeScript, TailwindCSS, TanStack Query |
| Backend | Go 1.21+, Gin, gorilla/websocket, SQLite |
| Agent | Rust 1.75+, tokio, tokio-tungstenite |
| Communication | WebSocket (TLS), REST API |

## Démarrage Rapide

```bash
# Cloner le projet
git clone https://github.com/AutoStrike-EIP/AutoStrike_POC.git
cd AutoStrike_POC

# Installer les dépendances et build
make install

# Lancer AutoStrike (API + Dashboard sur port 8443)
make run

# Dans un autre terminal, connecter un agent
make agent
```

Ouvrir **https://localhost:8443** (accepter le certificat auto-signé)

### Commandes utiles

| Commande | Description |
|----------|-------------|
| `make install` | Installer dépendances + build complet |
| `make run` | Démarrer le serveur (API + Dashboard) |
| `make agent` | Connecter un agent local |
| `make stop` | Arrêter les services |
| `make test` | Lancer tous les tests |
| `make logs` | Voir les logs serveur |
| `make certs` | Générer certificats TLS |

## Techniques MITRE ATT&CK

15 techniques implémentées :

| Tactique | Techniques |
|----------|------------|
| **Discovery** (9) | T1082, T1083, T1057, T1016, T1049, T1087, T1069, T1018, T1007 |
| **Execution** (3) | T1059.001, T1059.003, T1059.004 |
| **Persistence** (2) | T1053.005, T1547.001 |
| **Defense Evasion** (1) | T1070.004 |

## Tests

```bash
# Tous les tests
make test

# Par composant
cd server && go test ./...      # 85%+ coverage
cd agent && cargo test          # 67 tests
cd dashboard && npm test        # 513 tests
```

## Documentation

Voir la [documentation complète](https://autostrike-eip.github.io/AutoStrike/).

## Équipe

Projet EIP EPITECH - AutoStrike-EIP

## Licence

Ce projet est sous licence MIT - voir le fichier [LICENSE](LICENSE) pour plus de détails.
