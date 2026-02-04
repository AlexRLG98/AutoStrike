import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Layout from './Layout';
import { AuthProvider } from '../contexts/AuthContext';
import { authApi } from '../lib/api';

// Mock the API
vi.mock('../lib/api', () => ({
  authApi: {
    login: vi.fn(),
    me: vi.fn(),
    logout: vi.fn(),
    refresh: vi.fn(),
  },
}));

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

function renderLayout(path = '/dashboard', user = { username: 'TestUser', email: 'test@example.com' }) {
  // Set up the mock to return the user
  localStorageMock.getItem.mockImplementation((key: string) => {
    if (key === 'token') return 'test-token';
    return null;
  });

  vi.mocked(authApi.me).mockResolvedValue({
    data: {
      id: 'user-1',
      username: user.username,
      email: user.email,
      role: 'admin',
    },
  } as never);

  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthProvider>
        <Layout>
          <div data-testid="child-content">Test Content</div>
        </Layout>
      </AuthProvider>
    </MemoryRouter>
  );
}

describe('Layout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders AutoStrike brand name', async () => {
    renderLayout();
    await waitFor(() => {
      expect(screen.getByText('AutoStrike')).toBeInTheDocument();
    });
  });

  it('renders BAS Platform subtitle', async () => {
    renderLayout();
    await waitFor(() => {
      expect(screen.getByText('BAS Platform')).toBeInTheDocument();
    });
  });

  it('renders all navigation links', async () => {
    renderLayout();

    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument();
    });
    expect(screen.getByText('Agents')).toBeInTheDocument();
    expect(screen.getByText('Techniques')).toBeInTheDocument();
    expect(screen.getByText('Scenarios')).toBeInTheDocument();
    expect(screen.getByText('Executions')).toBeInTheDocument();
    expect(screen.getByText('Settings')).toBeInTheDocument();
  });

  it('renders children content', async () => {
    renderLayout();
    await waitFor(() => {
      expect(screen.getByTestId('child-content')).toBeInTheDocument();
    });
    expect(screen.getByText('Test Content')).toBeInTheDocument();
  });

  it('displays user info from context', async () => {
    renderLayout('/dashboard', { username: 'JohnDoe', email: 'john@example.com' });
    await waitFor(() => {
      expect(screen.getByText('JohnDoe')).toBeInTheDocument();
    });
    expect(screen.getByText('john@example.com')).toBeInTheDocument();
  });

  it('displays first letter of username in avatar', async () => {
    renderLayout('/dashboard', { username: 'JohnDoe', email: 'john@example.com' });
    await waitFor(() => {
      expect(screen.getByText('J')).toBeInTheDocument();
    });
  });

  it('renders logout button', async () => {
    renderLayout();
    await waitFor(() => {
      expect(screen.getByTitle('Logout')).toBeInTheDocument();
    });
  });

  it('calls logout when logout button is clicked', async () => {
    vi.mocked(authApi.logout).mockResolvedValue({} as never);

    renderLayout();

    await waitFor(() => {
      expect(screen.getByTitle('Logout')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTitle('Logout'));

    await waitFor(() => {
      expect(authApi.logout).toHaveBeenCalled();
    });
  });
});

describe('Layout Navigation Active State', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'TestUser',
        email: 'test@example.com',
        role: 'admin',
      },
    } as never);
  });

  it('highlights active route', async () => {
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <AuthProvider>
          <Layout>
            <div>Content</div>
          </Layout>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      const dashboardLink = screen.getByText('Dashboard').closest('a');
      expect(dashboardLink).toHaveClass('bg-primary-600');
    });
  });

  it('does not highlight inactive routes', async () => {
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <AuthProvider>
          <Layout>
            <div>Content</div>
          </Layout>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      const agentsLink = screen.getByText('Agents').closest('a');
      expect(agentsLink).not.toHaveClass('bg-primary-600');
      expect(agentsLink).toHaveClass('text-gray-300');
    });
  });
});

describe('Layout with default user', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.getItem.mockReturnValue(null);
  });

  it('displays default values when no user in context', async () => {
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <AuthProvider>
          <Layout>
            <div>Content</div>
          </Layout>
        </AuthProvider>
      </MemoryRouter>
    );

    // When not authenticated, the default values should show
    await waitFor(() => {
      expect(screen.getByText('Admin')).toBeInTheDocument();
    });
    expect(screen.getByText('admin@autostrike.local')).toBeInTheDocument();
  });
});
