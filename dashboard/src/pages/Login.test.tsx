import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Login from './Login';
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
  healthApi: {
    check: vi.fn(),
  },
}));

import { healthApi } from '../lib/api';

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Mock useNavigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

function renderLogin() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <Login />
      </AuthProvider>
    </MemoryRouter>
  );
}

describe('Login Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.getItem.mockReturnValue(null);
    // Default: auth is enabled
    vi.mocked(healthApi.check).mockResolvedValue({
      data: { status: 'ok', auth_enabled: true },
    } as never);
  });

  it('renders login form', () => {
    renderLogin();
    expect(screen.getByText('AutoStrike')).toBeInTheDocument();
    expect(screen.getByText('Sign in to your account')).toBeInTheDocument();
    expect(screen.getByLabelText('Username')).toBeInTheDocument();
    expect(screen.getByLabelText('Password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign in' })).toBeInTheDocument();
  });

  it('renders contact admin hint', () => {
    renderLogin();
    expect(screen.getByText('Contact your administrator for access credentials')).toBeInTheDocument();
  });

  it('renders platform subtitle', () => {
    renderLogin();
    expect(screen.getByText('Breach and Attack Simulation Platform')).toBeInTheDocument();
  });

  it('allows entering username and password', () => {
    renderLogin();
    const usernameInput = screen.getByLabelText('Username');
    const passwordInput = screen.getByLabelText('Password');

    fireEvent.change(usernameInput, { target: { value: 'testuser' } });
    fireEvent.change(passwordInput, { target: { value: 'testpass' } });

    expect(usernameInput).toHaveValue('testuser');
    expect(passwordInput).toHaveValue('testpass');
  });

  it('submits login form successfully', async () => {
    vi.mocked(authApi.login).mockResolvedValue({
      data: {
        access_token: 'test-token',
        refresh_token: 'test-refresh',
        expires_in: 900,
      },
    } as never);
    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'admin',
      },
    } as never);

    renderLogin();
    const usernameInput = screen.getByLabelText('Username');
    const passwordInput = screen.getByLabelText('Password');
    const submitButton = screen.getByRole('button', { name: 'Sign in' });

    fireEvent.change(usernameInput, { target: { value: 'testuser' } });
    fireEvent.change(passwordInput, { target: { value: 'testpass' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(authApi.login).toHaveBeenCalledWith({
        username: 'testuser',
        password: 'testpass',
      });
    });
  });

  it('shows error message on login failure', async () => {
    vi.mocked(authApi.login).mockRejectedValue(new Error('Invalid credentials'));

    renderLogin();
    const usernameInput = screen.getByLabelText('Username');
    const passwordInput = screen.getByLabelText('Password');
    const submitButton = screen.getByRole('button', { name: 'Sign in' });

    fireEvent.change(usernameInput, { target: { value: 'wronguser' } });
    fireEvent.change(passwordInput, { target: { value: 'wrongpass' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText('Invalid username or password')).toBeInTheDocument();
    });
  });

  it('shows loading state during login', async () => {
    vi.mocked(authApi.login).mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 100))
    );

    renderLogin();
    const usernameInput = screen.getByLabelText('Username');
    const passwordInput = screen.getByLabelText('Password');
    const submitButton = screen.getByRole('button', { name: 'Sign in' });

    fireEvent.change(usernameInput, { target: { value: 'testuser' } });
    fireEvent.change(passwordInput, { target: { value: 'testpass' } });
    fireEvent.click(submitButton);

    expect(await screen.findByText('Signing in...')).toBeInTheDocument();
  });

  it('disables submit button during loading', async () => {
    vi.mocked(authApi.login).mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 100))
    );

    renderLogin();
    const usernameInput = screen.getByLabelText('Username');
    const passwordInput = screen.getByLabelText('Password');
    const submitButton = screen.getByRole('button', { name: 'Sign in' });

    fireEvent.change(usernameInput, { target: { value: 'testuser' } });
    fireEvent.change(passwordInput, { target: { value: 'testpass' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(submitButton).toBeDisabled();
    });
  });

  it('has required attribute on username input', () => {
    renderLogin();
    expect(screen.getByLabelText('Username')).toBeRequired();
  });

  it('has required attribute on password input', () => {
    renderLogin();
    expect(screen.getByLabelText('Password')).toBeRequired();
  });

  it('has correct input types', () => {
    renderLogin();
    expect(screen.getByLabelText('Username')).toHaveAttribute('type', 'text');
    expect(screen.getByLabelText('Password')).toHaveAttribute('type', 'password');
  });

  it('redirects to dashboard when auth is disabled', async () => {
    vi.mocked(healthApi.check).mockResolvedValue({
      data: { status: 'ok', auth_enabled: false },
    } as never);

    renderLogin();

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/dashboard', { replace: true });
    });
  });

  it('does not redirect when auth is enabled', async () => {
    vi.mocked(healthApi.check).mockResolvedValue({
      data: { status: 'ok', auth_enabled: true },
    } as never);

    renderLogin();

    // Wait for auth check to complete
    await waitFor(() => {
      expect(healthApi.check).toHaveBeenCalled();
    });

    // Give it some time to potentially redirect
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Should not have navigated to dashboard
    expect(mockNavigate).not.toHaveBeenCalledWith('/dashboard', { replace: true });
  });

  it('redirects to dashboard when user is already authenticated', async () => {
    // Mock user already logged in with valid token
    localStorageMock.getItem.mockImplementation((key: string) => {
      if (key === 'token') return 'valid-token';
      if (key === 'refreshToken') return 'valid-refresh';
      return null;
    });

    vi.mocked(healthApi.check).mockResolvedValue({
      data: { status: 'ok', auth_enabled: true },
    } as never);

    vi.mocked(authApi.me).mockResolvedValue({
      data: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'admin',
      },
    } as never);

    renderLogin();

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/dashboard', { replace: true });
    });
  });
});
