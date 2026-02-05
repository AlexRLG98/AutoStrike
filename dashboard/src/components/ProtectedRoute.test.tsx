import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { ProtectedRoute } from './ProtectedRoute';
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

function renderWithRouter(
  initialEntries: string[] = ['/protected'],
  hasToken = false
) {
  localStorageMock.getItem.mockImplementation((key: string) => {
    if (key === 'token' && hasToken) return 'test-token';
    return null;
  });

  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<div>Login Page</div>} />
          <Route
            path="/protected"
            element={
              <ProtectedRoute>
                <div>Protected Content</div>
              </ProtectedRoute>
            }
          />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  );
}

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.getItem.mockReturnValue(null);
  });

  it('shows loading spinner while checking auth', () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 1000))
    );

    renderWithRouter(['/protected'], true);

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('redirects to login when not authenticated', async () => {
    localStorageMock.getItem.mockReturnValue(null);

    renderWithRouter(['/protected'], false);

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument();
    });
  });

  it('renders protected content when authenticated', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'admin',
      },
    } as never);

    renderWithRouter(['/protected'], true);

    await waitFor(() => {
      expect(screen.getByText('Protected Content')).toBeInTheDocument();
    });
  });

  it('redirects to login when token is invalid', async () => {
    localStorageMock.getItem.mockReturnValue('invalid-token');
    vi.mocked(authApi.me).mockRejectedValue(new Error('Unauthorized'));

    renderWithRouter(['/protected'], true);

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument();
    });
  });
});

describe('ProtectedRoute with children', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders children correctly when authenticated', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'admin',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/protected']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/protected"
              element={
                <ProtectedRoute>
                  <div data-testid="custom-child">Custom Child Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByTestId('custom-child')).toBeInTheDocument();
      expect(screen.getByText('Custom Child Content')).toBeInTheDocument();
    });
  });
});

describe('ProtectedRoute with requiredRole', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('allows access when user has exact required role', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'operator',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/admin']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/admin"
              element={
                <ProtectedRoute requiredRole="operator">
                  <div>Operator Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Operator Content')).toBeInTheDocument();
    });
  });

  it('allows access when user has higher role than required', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'admin',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/content']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/content"
              element={
                <ProtectedRoute requiredRole="viewer">
                  <div>Protected Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Protected Content')).toBeInTheDocument();
    });
  });

  it('denies access when user has lower role than required', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'viewer',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/admin']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/admin"
              element={
                <ProtectedRoute requiredRole="admin">
                  <div>Admin Only Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('403')).toBeInTheDocument();
      expect(screen.getByText('Access Denied')).toBeInTheDocument();
    });
  });

  it('shows return to dashboard link on access denied', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'viewer',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/admin']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/admin"
              element={
                <ProtectedRoute requiredRole="admin">
                  <div>Admin Only Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Return to Dashboard')).toBeInTheDocument();
    });
  });
});

describe('ProtectedRoute with allowedRoles', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('allows access when user role is in allowedRoles list', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'analyst',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/reports']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/reports"
              element={
                <ProtectedRoute allowedRoles={['admin', 'analyst', 'rssi']}>
                  <div>Reports Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Reports Content')).toBeInTheDocument();
    });
  });

  it('denies access when user role is not in allowedRoles list', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'viewer',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/reports']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/reports"
              element={
                <ProtectedRoute allowedRoles={['admin', 'analyst']}>
                  <div>Reports Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('403')).toBeInTheDocument();
      expect(screen.getByText('Access Denied')).toBeInTheDocument();
    });
  });

  it('shows permission denied message', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'viewer',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/admin']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/admin"
              element={
                <ProtectedRoute allowedRoles={['admin']}>
                  <div>Admin Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(
        screen.getByText("You don't have permission to access this page.")
      ).toBeInTheDocument();
    });
  });
});

describe('ProtectedRoute role hierarchy', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('admin has access to all role levels', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'admin',
        email: 'admin@example.com',
        role: 'admin',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/viewer-page']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/viewer-page"
              element={
                <ProtectedRoute requiredRole="viewer">
                  <div>Viewer Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Viewer Content')).toBeInTheDocument();
    });
  });

  it('rssi has access to analyst-level content', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'rssi',
        email: 'rssi@example.com',
        role: 'rssi',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/analyst-page']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/analyst-page"
              element={
                <ProtectedRoute requiredRole="analyst">
                  <div>Analyst Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Analyst Content')).toBeInTheDocument();
    });
  });

  it('operator cannot access rssi-level content', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'operator',
        email: 'operator@example.com',
        role: 'operator',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/rssi-page']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/rssi-page"
              element={
                <ProtectedRoute requiredRole="rssi">
                  <div>RSSI Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('403')).toBeInTheDocument();
      expect(screen.getByText('Access Denied')).toBeInTheDocument();
    });
  });
});

describe('ProtectedRoute without role requirements', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('allows any authenticated user when no role specified', async () => {
    localStorageMock.getItem.mockReturnValue('test-token');
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'viewer',
        email: 'viewer@example.com',
        role: 'viewer',
      },
    } as never);

    render(
      <MemoryRouter initialEntries={['/general']}>
        <AuthProvider>
          <Routes>
            <Route
              path="/general"
              element={
                <ProtectedRoute>
                  <div>General Content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('General Content')).toBeInTheDocument();
    });
  });
});
