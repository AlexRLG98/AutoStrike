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
