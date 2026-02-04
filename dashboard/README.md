# AutoStrike Dashboard

Interface web React pour la gestion et le monitoring de la plateforme AutoStrike.

## Architecture

```
dashboard/
├── src/
│   ├── main.tsx              # Point d'entrée React + AuthProvider
│   ├── App.tsx               # Routes principales
│   ├── index.css             # Styles globaux + Tailwind
│   ├── components/
│   │   ├── Layout.tsx        # Layout avec sidebar + logout
│   │   └── ProtectedRoute.tsx # Protection des routes
│   ├── contexts/
│   │   └── AuthContext.tsx   # État d'authentification
│   ├── pages/
│   │   ├── Dashboard.tsx     # Vue d'ensemble, graphiques
│   │   ├── Agents.tsx        # Gestion des agents
│   │   ├── Techniques.tsx    # Catalogue MITRE ATT&CK
│   │   ├── Scenarios.tsx     # Scénarios d'attaque
│   │   ├── Executions.tsx    # Historique des exécutions
│   │   ├── Settings.tsx      # Configuration
│   │   └── Login.tsx         # Page de connexion
│   └── lib/
│       └── api.ts            # Client Axios + authApi
├── package.json
├── vite.config.ts
├── tailwind.config.js
└── tsconfig.json
```

## Stack Technique

| Technologie | Version | Usage |
|-------------|---------|-------|
| React | 18.2 | Framework UI |
| TypeScript | 5.3 | Typage statique |
| Vite | 5.0 | Build tool |
| TailwindCSS | 3.4 | Styling |
| TanStack Query | 5.17 | Data fetching |
| React Router | 6.21 | Routing |
| Chart.js | 4.4 | Graphiques |
| Axios | 1.6 | HTTP client |

## Prérequis

- Node.js 18+
- npm ou yarn

## Installation

```bash
# Installer les dépendances
npm install

# Lancer en développement
npm run dev

# Build production
npm run build
```

## Scripts Disponibles

| Script | Description |
|--------|-------------|
| `npm run dev` | Serveur de développement (port 3000) |
| `npm run build` | Build de production |
| `npm run preview` | Preview du build |
| `npm run lint` | Vérification ESLint |
| `npm run test` | Tests Vitest |

## Configuration

### Proxy API (développement)

Le fichier `vite.config.ts` configure le proxy vers le backend :

```typescript
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'https://localhost:8443',
      changeOrigin: true,
      secure: false,
    },
    '/ws': {
      target: 'wss://localhost:8443',
      ws: true,
      secure: false,
    },
  },
}
```

### Alias d'import

```typescript
// Utiliser @/ pour importer depuis src/
import { api } from '@/lib/api';
```

## Pages

### Dashboard (`/dashboard`)
- Statistiques agents online/total
- Score de sécurité global
- Graphique donut des résultats (blocked/detected/successful)
- Activité récente

### Agents (`/agents`)
- Liste des agents avec statut
- Informations: hostname, platform, executors
- Last seen avec formatage relatif

### Techniques (`/techniques`)
- Tableau des techniques MITRE ATT&CK
- Filtrage par tactique et plateforme
- Badge safe/unsafe
- Import depuis fichiers YAML

### Scenarios (`/scenarios`)
- Cartes de scénarios avec phases
- Tags et nombre de techniques
- Bouton Run

### Executions (`/executions`)
- Historique des exécutions
- Score, statut, mode (safe/full)
- Détails des résultats par technique

### Settings (`/settings`)
- Configuration serveur
- Mode safe par défaut
- Paramètres agents (heartbeat, timeout)
- Chemins certificats TLS

## Composants UI

### Classes CSS Personnalisées

```css
.btn-primary    /* Bouton bleu */
.btn-danger     /* Bouton rouge */
.card           /* Carte avec ombre */
.input          /* Champ de saisie */
.badge          /* Badge inline */
.badge-success  /* Badge vert */
.badge-danger   /* Badge rouge */
.badge-warning  /* Badge orange */
```

### Couleurs (Tailwind)

```javascript
colors: {
  primary: { 50-950 },   // Bleu
  danger: { 50, 500, 600 },
  success: { 50, 500, 600 },
  warning: { 50, 500, 600 },
}
```

## API Client

Le client API (`src/lib/api.ts`) :
- Base URL: `/api`
- Intercepteur pour token JWT
- Redirection vers `/login` sur 401

```typescript
import { api } from '@/lib/api';

// GET
const agents = await api.get('/agents');

// POST
await api.post('/executions', { scenario_id, agent_paws, safe_mode });
```

## Data Fetching

Utilisation de TanStack Query :

```typescript
const { data: agents, isLoading } = useQuery({
  queryKey: ['agents'],
  queryFn: () => api.get('/agents').then(res => res.data),
});
```

## Docker

```bash
docker build -t autostrike-dashboard .
docker run -p 3000:80 autostrike-dashboard
```

Le conteneur utilise Nginx pour servir les fichiers statiques avec :
- Proxy vers le backend (`/api/`, `/ws/`)
- Fallback SPA (`try_files`)
- Cache des assets statiques

## Tests

```bash
npm run test
```

## Linting

```bash
npm run lint
```

Configuration ESLint stricte avec `max-warnings: 0`.
