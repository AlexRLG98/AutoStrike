import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';
import { authApi, healthApi } from '../lib/api';

// Mock the API
vi.mock('../lib/api', () => ({
  authApi: {
    login: vi.fn(),
    me: vi.fn(),
    logout: vi.fn(),
    refresh: vi.fn(),
  },
  healthApi: {
    check: vi.fn(),
  },
}));

// Helper component to test the hook
function TestComponent() {
  const { user, isAuthenticated, isLoading, authEnabled, login, logout } = useAuth();

  return (
    <div>
      <div data-testid="loading">{isLoading ? 'loading' : 'not-loading'}</div>
      <div data-testid="authenticated">
        {isAuthenticated ? 'authenticated' : 'not-authenticated'}
      </div>
      <div data-testid="auth-enabled">
        {authEnabled ? 'auth-enabled' : 'auth-disabled'}
      </div>
      <div data-testid="user">{user ? user.username : 'no-user'}</div>
      <button
        onClick={() => login({ username: 'test', password: 'pass' })}
        data-testid="login-btn"
      >
        Login
      </button>
      <button onClick={() => logout()} data-testid="logout-btn">
        Logout
      </button>
    </div>
  );
}

describe('AuthContext', () => {
  const originalLocalStorage = global.localStorage;
  let store: Record<string, string> = {};

  beforeEach(() => {
    store = {};
    const mockLocalStorage = {
      getItem: vi.fn((key: string) => store[key] || null),
      setItem: vi.fn((key: string, value: string) => {
        store[key] = value;
      }),
      removeItem: vi.fn((key: string) => {
        delete store[key];
      }),
      clear: vi.fn(() => {
        store = {};
      }),
      length: 0,
      key: vi.fn(),
    };
    Object.defineProperty(global, 'localStorage', {
      value: mockLocalStorage,
      writable: true,
    });
    vi.clearAllMocks();
    // Default: auth is enabled
    vi.mocked(healthApi.check).mockResolvedValue({
      data: { status: 'ok', auth_enabled: true },
    } as never);
  });

  afterEach(() => {
    Object.defineProperty(global, 'localStorage', {
      value: originalLocalStorage,
      writable: true,
    });
  });

  it('provides initial unauthenticated state without token', async () => {
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('not-loading');
    });
    expect(screen.getByTestId('authenticated')).toHaveTextContent(
      'not-authenticated'
    );
    expect(screen.getByTestId('user')).toHaveTextContent('no-user');
  });

  it('checks existing token on mount', async () => {
    store['token'] = 'existing-token';
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'existinguser',
        email: 'existing@example.com',
        role: 'admin',
      },
    } as never);

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent(
        'authenticated'
      );
    });
    expect(screen.getByTestId('user')).toHaveTextContent('existinguser');
    expect(authApi.me).toHaveBeenCalled();
  });

  it('clears token when validation fails on mount and no refresh token', async () => {
    store['token'] = 'invalid-token';
    vi.mocked(authApi.me).mockRejectedValue(new Error('Unauthorized'));

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('not-loading');
    });
    expect(screen.getByTestId('authenticated')).toHaveTextContent(
      'not-authenticated'
    );
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken');
  });

  it('attempts token refresh when validation fails and refresh token exists', async () => {
    store['token'] = 'expired-token';
    store['refreshToken'] = 'valid-refresh-token';

    // First call to me() fails
    vi.mocked(authApi.me).mockRejectedValueOnce(new Error('Unauthorized'));
    // Refresh succeeds
    vi.mocked(authApi.refresh).mockResolvedValueOnce({
      data: {
        access_token: 'new-access-token',
        refresh_token: 'new-refresh-token',
        expires_in: 900,
      },
    } as never);
    // Second call to me() succeeds with new token
    vi.mocked(authApi.me).mockResolvedValueOnce({
      data: {
        id: 'user-1',
        username: 'refresheduser',
        email: 'refreshed@example.com',
        role: 'admin',
      },
    } as never);

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('not-loading');
    });
    expect(screen.getByTestId('authenticated')).toHaveTextContent(
      'authenticated'
    );
    expect(screen.getByTestId('user')).toHaveTextContent('refresheduser');
    expect(authApi.refresh).toHaveBeenCalledWith('valid-refresh-token');
    expect(localStorage.setItem).toHaveBeenCalledWith('token', 'new-access-token');
    expect(localStorage.setItem).toHaveBeenCalledWith('refreshToken', 'new-refresh-token');
  });

  it('clears tokens when refresh fails', async () => {
    store['token'] = 'expired-token';
    store['refreshToken'] = 'expired-refresh-token';

    // First call to me() fails
    vi.mocked(authApi.me).mockRejectedValue(new Error('Unauthorized'));
    // Refresh also fails
    vi.mocked(authApi.refresh).mockRejectedValue(new Error('Refresh failed'));

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('not-loading');
    });
    expect(screen.getByTestId('authenticated')).toHaveTextContent(
      'not-authenticated'
    );
    expect(authApi.refresh).toHaveBeenCalledWith('expired-refresh-token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken');
  });

  it('clears tokens when refresh succeeds but second me() fails', async () => {
    store['token'] = 'expired-token';
    store['refreshToken'] = 'valid-refresh-token';

    // All me() calls fail
    vi.mocked(authApi.me).mockRejectedValue(new Error('Unauthorized'));
    // Refresh succeeds
    vi.mocked(authApi.refresh).mockResolvedValueOnce({
      data: {
        access_token: 'new-access-token',
        refresh_token: 'new-refresh-token',
        expires_in: 900,
      },
    } as never);

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('not-loading');
    });
    expect(screen.getByTestId('authenticated')).toHaveTextContent(
      'not-authenticated'
    );
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken');
  });

  it('handles login successfully', async () => {
    vi.mocked(authApi.login).mockResolvedValue({
      data: {
        access_token: 'new-token',
        refresh_token: 'new-refresh',
        expires_in: 900,
      },
    } as never);
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'newuser',
        email: 'new@example.com',
        role: 'operator',
      },
    } as never);

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('not-loading');
    });

    await act(async () => {
      screen.getByTestId('login-btn').click();
    });

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent(
        'authenticated'
      );
    });
    expect(screen.getByTestId('user')).toHaveTextContent('newuser');
    expect(localStorage.setItem).toHaveBeenCalledWith('token', 'new-token');
    expect(localStorage.setItem).toHaveBeenCalledWith(
      'refreshToken',
      'new-refresh'
    );
  });

  it('handles logout correctly', async () => {
    store['token'] = 'existing-token';
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'existinguser',
        email: 'existing@example.com',
        role: 'admin',
      },
    } as never);
    vi.mocked(authApi.logout).mockResolvedValue({} as never);

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent(
        'authenticated'
      );
    });

    await act(async () => {
      screen.getByTestId('logout-btn').click();
    });

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent(
        'not-authenticated'
      );
    });
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken');
  });

  it('handles logout when API fails', async () => {
    store['token'] = 'existing-token';
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'existinguser',
        email: 'existing@example.com',
        role: 'admin',
      },
    } as never);
    vi.mocked(authApi.logout).mockRejectedValue(new Error('Network error'));

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent(
        'authenticated'
      );
    });

    await act(async () => {
      screen.getByTestId('logout-btn').click();
    });

    // Should still clear local state even if API fails
    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent(
        'not-authenticated'
      );
    });
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
  });

  it('bypasses auth when server has auth disabled', async () => {
    vi.mocked(healthApi.check).mockResolvedValue({
      data: { status: 'ok', auth_enabled: false },
    } as never);

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('not-loading');
    });
    // Should be authenticated even without token when auth is disabled
    expect(screen.getByTestId('authenticated')).toHaveTextContent('authenticated');
    expect(screen.getByTestId('auth-enabled')).toHaveTextContent('auth-disabled');
    expect(screen.getByTestId('user')).toHaveTextContent('no-user');
    // Should not attempt to validate tokens
    expect(authApi.me).not.toHaveBeenCalled();
  });

  it('assumes auth enabled when health check fails', async () => {
    vi.mocked(healthApi.check).mockRejectedValue(new Error('Network error'));

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('not-loading');
    });
    // Should assume auth is enabled for safety
    expect(screen.getByTestId('auth-enabled')).toHaveTextContent('auth-enabled');
    expect(screen.getByTestId('authenticated')).toHaveTextContent('not-authenticated');
  });
});

describe('useAuth hook', () => {
  it('throws error when used outside AuthProvider', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => {
      render(<TestComponent />);
    }).toThrow('useAuth must be used within an AuthProvider');

    consoleError.mockRestore();
  });
});
