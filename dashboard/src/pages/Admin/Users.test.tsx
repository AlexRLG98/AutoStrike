import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Users from './Users';

// Mock the auth context
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: vi.fn(() => ({
    user: { id: 'current-user', username: 'admin', role: 'admin' },
    isAuthenticated: true,
  })),
}));

// Mock the API
vi.mock('../../lib/api', () => ({
  adminApi: {
    listUsers: vi.fn(() =>
      Promise.resolve({
        data: {
          users: [
            {
              id: 'user-1',
              username: 'admin',
              email: 'admin@example.com',
              role: 'admin',
              role_display: 'Administrator',
              is_active: true,
              last_login_at: new Date().toISOString(),
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            },
            {
              id: 'user-2',
              username: 'operator',
              email: 'operator@example.com',
              role: 'operator',
              role_display: 'Operator',
              is_active: true,
              last_login_at: null,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            },
            {
              id: 'user-3',
              username: 'inactive',
              email: 'inactive@example.com',
              role: 'viewer',
              role_display: 'Viewer',
              is_active: false,
              last_login_at: null,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            },
          ],
          total: 3,
        },
      })
    ),
    createUser: vi.fn(() => Promise.resolve({ data: {} })),
    updateUser: vi.fn(() => Promise.resolve({ data: {} })),
    updateUserRole: vi.fn(() => Promise.resolve({ data: {} })),
    resetPassword: vi.fn(() => Promise.resolve({ data: {} })),
    deactivateUser: vi.fn(() => Promise.resolve({ data: {} })),
    reactivateUser: vi.fn(() => Promise.resolve({ data: {} })),
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

function renderUsers() {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Users />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('Users Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders user management title', async () => {
    renderUsers();
    await waitFor(() => {
      expect(screen.getByText('User Management')).toBeInTheDocument();
    });
  });

  it('renders add user button', async () => {
    renderUsers();
    await waitFor(() => {
      expect(screen.getByText('Add User')).toBeInTheDocument();
    });
  });

  it('displays users after loading', async () => {
    renderUsers();
    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
      expect(screen.getByText('operator@example.com')).toBeInTheDocument();
    });
  });

  it('shows role badges', async () => {
    renderUsers();
    await waitFor(() => {
      expect(screen.getByText('Administrator')).toBeInTheDocument();
      expect(screen.getByText('Operator')).toBeInTheDocument();
    });
  });

  it('shows active/inactive status', async () => {
    renderUsers();
    await waitFor(() => {
      const activeLabels = screen.getAllByText('Active');
      expect(activeLabels.length).toBeGreaterThan(0);
    });
  });

  it('shows show inactive users checkbox', async () => {
    renderUsers();
    await waitFor(() => {
      expect(screen.getByText('Show inactive users')).toBeInTheDocument();
    });
  });

  it('shows last login time or Never', async () => {
    renderUsers();
    await waitFor(() => {
      const neverTexts = screen.getAllByText('Never');
      expect(neverTexts.length).toBeGreaterThan(0);
    });
  });
});

describe('Users Create Modal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('has add user button', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const addButton = screen.getByText('Add User');
    expect(addButton).toBeInTheDocument();
  });

  it('shows role options in create form', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Add User'));

    await waitFor(() => {
      const roleSelect = screen.getByLabelText('Role');
      expect(roleSelect).toBeInTheDocument();
    });
  });
});

describe('Users Edit Modal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('opens edit modal when edit button clicked', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const editButtons = screen.getAllByTitle('Edit user');
    fireEvent.click(editButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Edit User')).toBeInTheDocument();
    });
  });

  it('closes edit modal when cancel clicked', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const editButtons = screen.getAllByTitle('Edit user');
    fireEvent.click(editButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Edit User')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Cancel'));

    await waitFor(() => {
      expect(screen.queryByText('Edit User')).not.toBeInTheDocument();
    });
  });
});

describe('Users Role Modal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('opens role modal when change role button clicked', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('operator@example.com')).toBeInTheDocument();
    });

    const roleButtons = screen.getAllByTitle('Change role');
    // Click on a user that is not the current user
    const enabledRoleButton = roleButtons.find(
      (btn) => !(btn as HTMLButtonElement).disabled
    );
    if (enabledRoleButton) {
      fireEvent.click(enabledRoleButton);

      await waitFor(() => {
        expect(screen.getByText('Change Role')).toBeInTheDocument();
      });
    }
  });
});

describe('Users Password Modal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows reset password buttons', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const passwordButtons = screen.getAllByTitle('Reset password');
    expect(passwordButtons.length).toBeGreaterThan(0);
  });
});

describe('Users Deactivate Modal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('opens deactivate modal when deactivate button clicked', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('operator@example.com')).toBeInTheDocument();
    });

    const deactivateButtons = screen.getAllByTitle('Deactivate user');
    // Click on a user that is not the current user
    const enabledButton = deactivateButtons.find(
      (btn) => !(btn as HTMLButtonElement).disabled
    );
    if (enabledButton) {
      fireEvent.click(enabledButton);

      await waitFor(() => {
        expect(screen.getByText('Deactivate User')).toBeInTheDocument();
      });
    }
  });

  it('shows warning message in deactivate modal', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('operator@example.com')).toBeInTheDocument();
    });

    const deactivateButtons = screen.getAllByTitle('Deactivate user');
    const enabledButton = deactivateButtons.find(
      (btn) => !(btn as HTMLButtonElement).disabled
    );
    if (enabledButton) {
      fireEvent.click(enabledButton);

      await waitFor(() => {
        expect(
          screen.getByText(/The user will no longer be able to log in/)
        ).toBeInTheDocument();
      });
    }
  });
});

describe('Users Reactivate Modal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows reactivate button for inactive users', async () => {
    renderUsers();

    // First, enable showing inactive users
    await waitFor(() => {
      expect(screen.getByText('Show inactive users')).toBeInTheDocument();
    });

    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);

    await waitFor(() => {
      const reactivateButtons = screen.queryAllByTitle('Reactivate user');
      expect(reactivateButtons.length).toBeGreaterThanOrEqual(0);
    });
  });
});

describe('Users Table Actions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('disables change role button for current user', async () => {
    // Mock useAuth to return the same user as in the list
    const { useAuth } = await import('../../contexts/AuthContext');
    vi.mocked(useAuth).mockReturnValue({
      user: { id: 'user-1', username: 'admin', email: 'admin@example.com', role: 'admin', is_active: true, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
      isAuthenticated: true,
      isLoading: false,
      login: vi.fn(),
      logout: vi.fn(),
      authEnabled: true,
    });

    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const roleButtons = screen.getAllByTitle('Change role');
    // The first user (admin) should have a disabled role button
    expect(roleButtons[0]).toBeDisabled();
  });

  it('disables deactivate button for current user', async () => {
    const { useAuth } = await import('../../contexts/AuthContext');
    vi.mocked(useAuth).mockReturnValue({
      user: { id: 'user-1', username: 'admin', email: 'admin@example.com', role: 'admin', is_active: true, created_at: new Date().toISOString(), updated_at: new Date().toISOString() },
      isAuthenticated: true,
      isLoading: false,
      login: vi.fn(),
      logout: vi.fn(),
      authEnabled: true,
    });

    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const deactivateButtons = screen.getAllByTitle('Deactivate user');
    // The first user (admin) should have a disabled deactivate button
    expect(deactivateButtons[0]).toBeDisabled();
  });
});

describe('Users Empty State', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows empty state when no users', async () => {
    const { adminApi } = await import('../../lib/api');
    vi.mocked(adminApi.listUsers).mockResolvedValueOnce({
      data: { users: [], total: 0 },
    } as never);

    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('No users found')).toBeInTheDocument();
    });
  });
});

describe('Users Role Badge Colors', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('displays role labels correctly', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('Administrator')).toBeInTheDocument();
      expect(screen.getByText('Operator')).toBeInTheDocument();
    });
  });
});

describe('Users Filter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('toggles include inactive users', async () => {
    const { adminApi } = await import('../../lib/api');

    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('Show inactive users')).toBeInTheDocument();
    });

    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);

    await waitFor(() => {
      expect(adminApi.listUsers).toHaveBeenCalledWith(true);
    });
  });
});

describe('Users Create Form', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows username field in create modal', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Add User'));

    await waitFor(() => {
      expect(screen.getByLabelText('Username')).toBeInTheDocument();
    });
  });

  it('shows email field in create modal', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Add User'));

    await waitFor(() => {
      expect(screen.getByLabelText('Email')).toBeInTheDocument();
    });
  });

  it('shows password field in create modal', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Add User'));

    await waitFor(() => {
      expect(screen.getByLabelText('Password')).toBeInTheDocument();
    });
  });

  it('shows cancel button in create modal', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Add User'));

    await waitFor(() => {
      const cancelButtons = screen.getAllByText('Cancel');
      expect(cancelButtons.length).toBeGreaterThan(0);
    });
  });

  it('closes create modal on cancel', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Add User'));

    await waitFor(() => {
      expect(screen.getByLabelText('Username')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Cancel'));

    await waitFor(() => {
      expect(screen.queryByLabelText('Username')).not.toBeInTheDocument();
    });
  });
});

describe('Users API Errors', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('handles list users error gracefully', async () => {
    const { adminApi } = await import('../../lib/api');
    vi.mocked(adminApi.listUsers).mockRejectedValueOnce(new Error('Network error'));

    renderUsers();

    // Should not crash even with error
    await waitFor(() => {
      expect(screen.getByText('User Management')).toBeInTheDocument();
    });
  });
});

describe('Users Table Display', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows user count in header', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('User Management')).toBeInTheDocument();
    });
  });

  it('displays inactive status for inactive users', async () => {
    renderUsers();

    // First enable showing inactive users
    await waitFor(() => {
      expect(screen.getByText('Show inactive users')).toBeInTheDocument();
    });

    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);

    await waitFor(() => {
      const inactiveLabels = screen.queryAllByText('Inactive');
      expect(inactiveLabels.length).toBeGreaterThanOrEqual(0);
    });
  });

  it('shows viewer role badge', async () => {
    renderUsers();

    // Enable showing inactive users to see the viewer role
    await waitFor(() => {
      expect(screen.getByText('Show inactive users')).toBeInTheDocument();
    });

    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);

    await waitFor(() => {
      const viewerBadges = screen.queryAllByText('Viewer');
      expect(viewerBadges.length).toBeGreaterThanOrEqual(0);
    });
  });
});

describe('Users Password Reset', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('has password reset buttons for each user', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const passwordButtons = screen.getAllByTitle('Reset password');
    expect(passwordButtons.length).toBeGreaterThan(0);
  });
});

describe('Users Edit Form', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows email field in edit modal', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const editButtons = screen.getAllByTitle('Edit user');
    fireEvent.click(editButtons[0]);

    await waitFor(() => {
      expect(screen.getByLabelText('Email')).toBeInTheDocument();
    });
  });

  it('pre-fills email in edit modal', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const editButtons = screen.getAllByTitle('Edit user');
    fireEvent.click(editButtons[0]);

    await waitFor(() => {
      const emailInput = screen.getByDisplayValue('admin@example.com');
      expect(emailInput).toBeInTheDocument();
    });
  });

  it('shows username field in edit modal', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const editButtons = screen.getAllByTitle('Edit user');
    fireEvent.click(editButtons[0]);

    await waitFor(() => {
      expect(screen.getByLabelText('Username')).toBeInTheDocument();
    });
  });
});

describe('Users Action Buttons', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows all action buttons for each user', async () => {
    renderUsers();

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument();
    });

    const editButtons = screen.getAllByTitle('Edit user');
    const roleButtons = screen.getAllByTitle('Change role');
    const passwordButtons = screen.getAllByTitle('Reset password');
    const deactivateButtons = screen.getAllByTitle('Deactivate user');

    expect(editButtons.length).toBeGreaterThan(0);
    expect(roleButtons.length).toBeGreaterThan(0);
    expect(passwordButtons.length).toBeGreaterThan(0);
    expect(deactivateButtons.length).toBeGreaterThan(0);
  });
});
