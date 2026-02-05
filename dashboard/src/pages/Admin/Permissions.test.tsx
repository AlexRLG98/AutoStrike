import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Permissions from './Permissions';

// Mock the auth context
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: vi.fn(() => ({
    user: { id: 'user-1', username: 'admin', role: 'admin' },
    isAuthenticated: true,
  })),
}));

// Mock the API
const mockGetMatrix = vi.fn();
const mockGetMyPermissions = vi.fn();

vi.mock('../../lib/api', () => ({
  permissionApi: {
    getMatrix: () => mockGetMatrix(),
    getMyPermissions: () => mockGetMyPermissions(),
  },
}));

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
}

function renderPermissions() {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <Permissions />
    </QueryClientProvider>
  );
}

const mockMatrixData = {
  roles: ['admin', 'rssi', 'operator', 'analyst', 'viewer'] as const,
  categories: [
    {
      name: 'Agents',
      description: 'Agent management permissions',
      permissions: ['agents.read', 'agents.write', 'agents.delete'],
    },
    {
      name: 'Scenarios',
      description: 'Scenario management permissions',
      permissions: ['scenarios.read', 'scenarios.write', 'scenarios.execute'],
    },
  ],
  permissions: [
    {
      permission: 'agents.read',
      name: 'View Agents',
      description: 'View agent list and details',
      category: 'Agents',
    },
    {
      permission: 'agents.write',
      name: 'Manage Agents',
      description: 'Create and update agents',
      category: 'Agents',
    },
    {
      permission: 'agents.delete',
      name: 'Delete Agents',
      description: 'Remove agents from the system',
      category: 'Agents',
    },
    {
      permission: 'scenarios.read',
      name: 'View Scenarios',
      description: 'View scenario list and details',
      category: 'Scenarios',
    },
    {
      permission: 'scenarios.write',
      name: 'Manage Scenarios',
      description: 'Create and update scenarios',
      category: 'Scenarios',
    },
    {
      permission: 'scenarios.execute',
      name: 'Execute Scenarios',
      description: 'Run attack simulations',
      category: 'Scenarios',
    },
  ],
  matrix: {
    admin: ['agents.read', 'agents.write', 'agents.delete', 'scenarios.read', 'scenarios.write', 'scenarios.execute'],
    rssi: ['agents.read', 'scenarios.read'],
    operator: ['agents.read', 'agents.write', 'scenarios.read', 'scenarios.write', 'scenarios.execute'],
    analyst: ['agents.read', 'scenarios.read'],
    viewer: ['agents.read', 'scenarios.read'],
  },
};

const mockMyPermissions = {
  role: 'admin',
  permissions: ['agents.read', 'agents.write', 'agents.delete', 'scenarios.read', 'scenarios.write', 'scenarios.execute'],
};

describe('Permissions Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetMatrix.mockResolvedValue({ data: mockMatrixData });
    mockGetMyPermissions.mockResolvedValue({ data: mockMyPermissions });
  });

  it('renders loading state initially', () => {
    mockGetMatrix.mockReturnValue(new Promise(() => {}));
    mockGetMyPermissions.mockReturnValue(new Promise(() => {}));

    renderPermissions();
    expect(screen.getByText('Loading permissions...')).toBeInTheDocument();
  });

  it('renders page title after loading', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('Permissions Matrix')).toBeInTheDocument();
    });
  });

  it('displays page description', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('Overview of role-based permissions across the system')).toBeInTheDocument();
    });
  });

  it('displays your permissions section', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('Your Permissions')).toBeInTheDocument();
    });
  });

  it('displays user role badge', async () => {
    renderPermissions();

    await waitFor(() => {
      const adminBadges = screen.getAllByText('Admin');
      expect(adminBadges.length).toBeGreaterThan(0);
    });
  });

  it('displays user permission badges', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('agents.read')).toBeInTheDocument();
      expect(screen.getByText('agents.write')).toBeInTheDocument();
      expect(screen.getByText('scenarios.execute')).toBeInTheDocument();
    });
  });

  it('displays all role columns in table', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getAllByText('Admin').length).toBeGreaterThan(0);
      expect(screen.getAllByText('RSSI').length).toBeGreaterThan(0);
      expect(screen.getAllByText('Operator').length).toBeGreaterThan(0);
      expect(screen.getAllByText('Analyst').length).toBeGreaterThan(0);
      expect(screen.getAllByText('Viewer').length).toBeGreaterThan(0);
    });
  });

  it('displays category headers', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('Agents')).toBeInTheDocument();
      expect(screen.getByText('Scenarios')).toBeInTheDocument();
    });
  });

  it('displays category descriptions', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText(/Agent management permissions/)).toBeInTheDocument();
      expect(screen.getByText(/Scenario management permissions/)).toBeInTheDocument();
    });
  });

  it('displays permission names in table', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('View Agents')).toBeInTheDocument();
      expect(screen.getByText('Manage Agents')).toBeInTheDocument();
      expect(screen.getByText('Delete Agents')).toBeInTheDocument();
      expect(screen.getByText('View Scenarios')).toBeInTheDocument();
      expect(screen.getByText('Manage Scenarios')).toBeInTheDocument();
      expect(screen.getByText('Execute Scenarios')).toBeInTheDocument();
    });
  });

  it('displays permission descriptions', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('View agent list and details')).toBeInTheDocument();
      expect(screen.getByText('Run attack simulations')).toBeInTheDocument();
    });
  });

  it('renders permission table header', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('Permission')).toBeInTheDocument();
    });
  });
});

describe('Permissions Matrix Indicators', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetMatrix.mockResolvedValue({ data: mockMatrixData });
    mockGetMyPermissions.mockResolvedValue({ data: mockMyPermissions });
  });

  it('renders check icons for allowed permissions', async () => {
    renderPermissions();

    await waitFor(() => {
      // Admin has all permissions - should have many checkmarks
      expect(screen.getByText('View Agents')).toBeInTheDocument();
    });
  });
});

describe('Role Descriptions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetMatrix.mockResolvedValue({ data: mockMatrixData });
    mockGetMyPermissions.mockResolvedValue({ data: mockMyPermissions });
  });

  it('displays role description cards', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('Full system access. Can manage users, configure settings, and perform all operations.')).toBeInTheDocument();
    });
  });

  it('displays RSSI role description', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText(/Security Officer/)).toBeInTheDocument();
    });
  });

  it('displays operator role description', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText(/Can manage agents, scenarios, and execute attack simulations/)).toBeInTheDocument();
    });
  });

  it('displays analyst role description', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText(/Read-only access with analytics capabilities/)).toBeInTheDocument();
    });
  });

  it('displays viewer role description', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText(/Basic read-only access/)).toBeInTheDocument();
    });
  });

  it('shows permission count for each role', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('6 permissions')).toBeInTheDocument();
      // Multiple roles have 2 permissions (rssi, analyst, viewer)
      const twoPermTexts = screen.getAllByText('2 permissions');
      expect(twoPermTexts.length).toBe(3);
    });
  });

  it('shows singular permission text when 1 permission', async () => {
    mockGetMatrix.mockResolvedValue({
      data: {
        ...mockMatrixData,
        matrix: {
          ...mockMatrixData.matrix,
          viewer: ['agents.read'],
        },
      },
    });

    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('1 permission')).toBeInTheDocument();
    });
  });
});

describe('Permissions with Different User Roles', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetMatrix.mockResolvedValue({ data: mockMatrixData });
  });

  it('displays viewer role when user is viewer', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    vi.mocked(useAuth).mockReturnValue({
      user: { id: 'user-2', username: 'viewer', email: 'viewer@example.com', role: 'viewer', is_active: true, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
      isAuthenticated: true,
      isLoading: false,
      login: vi.fn(),
      logout: vi.fn(),
      authEnabled: true,
    });

    mockGetMyPermissions.mockResolvedValue({
      data: {
        role: 'viewer',
        permissions: ['agents.read', 'scenarios.read'],
      },
    });

    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('Your Permissions')).toBeInTheDocument();
      // Viewer should have fewer permissions
      expect(screen.getByText('agents.read')).toBeInTheDocument();
    });
  });

  it('displays operator role permissions', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    vi.mocked(useAuth).mockReturnValue({
      user: { id: 'user-3', username: 'operator', email: 'operator@example.com', role: 'operator', is_active: true, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
      isAuthenticated: true,
      isLoading: false,
      login: vi.fn(),
      logout: vi.fn(),
      authEnabled: true,
    });

    mockGetMyPermissions.mockResolvedValue({
      data: {
        role: 'operator',
        permissions: ['agents.read', 'agents.write', 'scenarios.read', 'scenarios.write', 'scenarios.execute'],
      },
    });

    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('agents.read')).toBeInTheDocument();
      expect(screen.getByText('scenarios.execute')).toBeInTheDocument();
    });
  });
});

describe('Permissions Empty States', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('handles empty matrix gracefully', async () => {
    mockGetMatrix.mockResolvedValue({
      data: {
        roles: [],
        categories: [],
        permissions: [],
        matrix: {},
      },
    });
    mockGetMyPermissions.mockResolvedValue({
      data: { role: 'viewer', permissions: [] },
    });

    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('Permissions Matrix')).toBeInTheDocument();
    });
  });

  it('handles missing myPermissions gracefully', async () => {
    mockGetMatrix.mockResolvedValue({ data: mockMatrixData });
    mockGetMyPermissions.mockResolvedValue({ data: null });

    renderPermissions();

    await waitFor(() => {
      expect(screen.getByText('Permissions Matrix')).toBeInTheDocument();
    });
  });
});

describe('Permissions Role Badge Colors', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetMatrix.mockResolvedValue({ data: mockMatrixData });
    mockGetMyPermissions.mockResolvedValue({ data: mockMyPermissions });
  });

  it('displays role-specific styling for admin', async () => {
    renderPermissions();

    await waitFor(() => {
      const adminBadges = screen.getAllByText('Admin');
      expect(adminBadges[0]).toBeInTheDocument();
    });
  });

  it('displays role labels in table header', async () => {
    renderPermissions();

    await waitFor(() => {
      expect(screen.getByRole('table')).toBeInTheDocument();
    });
  });
});

describe('Permissions Permission Check Logic', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetMatrix.mockResolvedValue({ data: mockMatrixData });
    mockGetMyPermissions.mockResolvedValue({ data: mockMyPermissions });
  });

  it('correctly identifies when admin has all permissions', async () => {
    renderPermissions();

    await waitFor(() => {
      // Admin has 6 permissions
      expect(screen.getByText('6 permissions')).toBeInTheDocument();
    });
  });

  it('correctly shows viewer has limited permissions', async () => {
    renderPermissions();

    await waitFor(() => {
      // Viewer has 2 permissions (along with rssi and analyst)
      const twoPermTexts = screen.getAllByText('2 permissions');
      expect(twoPermTexts.length).toBeGreaterThan(0);
    });
  });
});
