# Dashboard (React)

The AutoStrike dashboard is built with **React 18**, **TypeScript**, and **TailwindCSS**.

---

## Tech Stack

| Technology | Version | Usage |
|------------|---------|-------|
| React | 18.2 | UI Framework |
| TypeScript | 5.3 | Static typing |
| Vite | 5.0 | Build tool |
| TailwindCSS | 3.4 | Utility-first styling |
| TanStack Query | 5.17 | Data fetching & caching |
| React Router | 6.21 | SPA navigation |
| Chart.js | 4.4 | Charts (Doughnut) |
| Axios | 1.6 | HTTP client |
| react-hot-toast | 2.4 | Notifications |

---

## Structure

```
dashboard/
├── src/
│   ├── main.tsx              # React entry + providers
│   ├── App.tsx               # Route configuration
│   ├── index.css             # Global styles + Tailwind directives
│   ├── components/
│   │   ├── Layout.tsx        # Sidebar navigation layout + logout
│   │   ├── ProtectedRoute.tsx # Route protection wrapper
│   │   ├── MitreMatrix.tsx   # Interactive MITRE ATT&CK matrix
│   │   ├── RunExecutionModal.tsx  # Execution configuration modal
│   │   ├── LoadingState.tsx  # Loading spinner component
│   │   ├── EmptyState.tsx    # Empty state placeholder
│   │   └── ErrorBoundary.tsx # Error boundary wrapper
│   ├── contexts/
│   │   └── AuthContext.tsx   # Authentication state & hooks
│   ├── hooks/
│   │   └── useWebSocket.ts   # WebSocket connection hook
│   ├── pages/
│   │   ├── Dashboard.tsx     # Overview, KPIs, charts
│   │   ├── Agents.tsx        # Agent list and management
│   │   ├── Techniques.tsx    # MITRE ATT&CK catalog
│   │   ├── Matrix.tsx        # MITRE ATT&CK matrix page
│   │   ├── Scenarios.tsx     # Attack scenarios
│   │   ├── Executions.tsx    # Execution history
│   │   ├── ExecutionDetails.tsx  # Detailed execution results
│   │   ├── Settings.tsx      # Application configuration
│   │   └── Login.tsx         # Login page
│   ├── lib/
│   │   └── api.ts            # Axios client with interceptors
│   └── types/
│       └── index.ts          # TypeScript type definitions
├── public/
├── package.json
├── vite.config.ts            # Vite config + API proxy
├── tailwind.config.js        # Tailwind + custom colors
├── tsconfig.json
└── Dockerfile                # Multi-stage build + nginx
```

---

## Pages

### Dashboard (`/dashboard`)

Overview page with:
- **Agents Online**: Connected agent count
- **Security Score**: Latest execution score
- **Techniques Tested**: Number of techniques executed
- **Doughnut Chart**: Distribution of blocked/detected/successful
- **Recent Activity**: Last 5 executions

### Agents (`/agents`)

Agent management:
- Cards with status (online/offline badge)
- Information: hostname, username, platform
- Available executors (badges)
- Last seen with relative time

### Techniques (`/techniques`)

MITRE ATT&CK catalog:
- Table with ID, name, description
- Tactic color coding (14 colors)
- Platform badges
- Safe/Unsafe badge

### Matrix (`/matrix`)

Interactive MITRE ATT&CK matrix:
- 14 tactic columns
- Technique cells with safety indicators
- Platform filter dropdown
- Click to view technique details
- Coverage statistics

### Scenarios (`/scenarios`)

Attack scenarios:
- Cards with numbered phases
- Technique count per phase
- Category tags
- **Run** button → Opens RunExecutionModal

### Executions (`/executions`)

Execution history:
- Table with score, status, mode
- Details: blocked/detected/successful counts
- Stop button for running executions
- **Real-time WebSocket updates**
- Click row → ExecutionDetails page

### ExecutionDetails (`/executions/:id`)

Detailed results:
- Execution summary header
- Score breakdown (blocked/detected/successful/total)
- Results table with technique, agent, status, output
- Expandable output viewer
- Real-time polling while running

### Settings (`/settings`)

Configuration:
- Server URL
- Default safe mode toggle
- Agent settings (heartbeat, timeout)
- TLS certificate paths

---

## Components

### MitreMatrix

Interactive MITRE ATT&CK matrix visualization.

**Props:**
```typescript
interface MitreMatrixProps {
  techniques: Technique[];
  onTechniqueClick?: (technique: Technique) => void;
}
```

**Features:**
- CSS Grid with 14 tactic columns
- Platform filtering
- Safety indicators (green/red dots)
- Detail modal on click

### RunExecutionModal

Execution configuration modal.

**Props:**
```typescript
interface RunExecutionModalProps {
  scenario: Scenario;
  onConfirm: (agentPaws: string[], safeMode: boolean) => void;
  onCancel: () => void;
  isLoading: boolean;
}
```

**Features:**
- Agent multi-select (online agents only)
- Safe mode toggle
- Scenario info display
- Validation (requires at least one agent)

---

## Custom Hook: useWebSocket

Real-time updates via WebSocket.

```typescript
interface UseWebSocketOptions {
  onMessage?: (message: WebSocketMessage) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  reconnectInterval?: number;  // default: 3000ms
  maxRetries?: number;         // default: 5
}

const { isConnected, send, lastMessage } = useWebSocket({
  onMessage: (msg) => {
    if (msg.type === 'execution_completed') {
      queryClient.invalidateQueries(['executions']);
    }
  },
});
```

**Features:**
- Auto-reconnection with exponential backoff
- Connection state tracking
- JSON serialization

---

## CSS Classes (Tailwind)

```css
.btn-primary    /* Primary blue button */
.btn-danger     /* Danger red button */
.card           /* Card with shadow and border */
.input          /* Styled input field */
.badge          /* Inline badge */
.badge-success  /* Green badge (online, safe) */
.badge-danger   /* Red badge (offline, unsafe) */
.badge-warning  /* Orange badge (running) */
```

---

## Data Fetching

Using **TanStack Query** for fetching and caching:

```typescript
import { useQuery, useMutation } from '@tanstack/react-query';
import { api, executionApi } from '@/lib/api';

// Query
const { data: agents, isLoading } = useQuery({
  queryKey: ['agents'],
  queryFn: () => api.get('/agents').then(res => res.data),
});

// Mutation
const startMutation = useMutation({
  mutationFn: ({ scenarioId, agentPaws, safeMode }) =>
    executionApi.start(scenarioId, agentPaws, safeMode),
  onSuccess: () => {
    queryClient.invalidateQueries(['executions']);
    toast.success('Execution started');
  },
});
```

---

## API Client

The API client (`src/lib/api.ts`) provides:
- Configurable base URL (`/api/v1`)
- JWT token injection via interceptor
- 401 handling (removes invalid token)
- Typed API methods

```typescript
export const executionApi = {
  list: () => api.get('/executions'),
  get: (id: string) => api.get(`/executions/${id}`),
  getResults: (id: string) => api.get(`/executions/${id}/results`),
  start: (scenarioId, agentPaws, safeMode) => api.post('/executions', {...}),
  stop: (id: string) => api.post(`/executions/${id}/stop`),
};
```

---

## Development

```bash
cd dashboard
npm install       # Install dependencies
npm run dev       # Development server (port 3000)
npm run build     # Production build
npm run preview   # Preview build
npm run lint      # ESLint check
npm run type-check # TypeScript check
npm test          # Vitest tests (269 tests)
```

---

## Testing

269 tests across 21 test files:

- Component tests (Layout, MitreMatrix, RunExecutionModal)
- Page tests (Dashboard, Agents, Techniques, Scenarios, Executions, etc.)
- Hook tests (useWebSocket)
- API client tests

```bash
npm test -- --run        # Run once
npm test -- --coverage   # With coverage
```

---

## Docker

Multi-stage Dockerfile:
1. **Builder**: Node 20 Alpine for build
2. **Runtime**: Nginx Alpine for serving

```bash
docker build -t autostrike-dashboard .
docker run -p 3000:80 autostrike-dashboard
```

Nginx configuration:
- Proxy `/api/` to backend
- Proxy `/ws/` for WebSocket
- SPA fallback (`try_files`)
- Static asset caching
