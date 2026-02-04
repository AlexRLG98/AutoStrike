import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useMemo,
  ReactNode,
} from 'react';
import { authApi, healthApi, LoginCredentials, User } from '../lib/api';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  authEnabled: boolean;
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  readonly children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [authEnabled, setAuthEnabled] = useState(true); // Default to true for safety

  const isAuthenticated = user !== null || !authEnabled;

  // Try to refresh tokens when access token is invalid
  const tryRefreshToken = useCallback(async (): Promise<boolean> => {
    const refreshToken = localStorage.getItem('refreshToken');
    if (!refreshToken) {
      return false;
    }

    try {
      const response = await authApi.refresh(refreshToken);
      const { access_token, refresh_token } = response.data;
      localStorage.setItem('token', access_token);
      localStorage.setItem('refreshToken', refresh_token);
      return true;
    } catch {
      // Refresh token also invalid/expired
      return false;
    }
  }, []);

  // Check auth status on mount
  useEffect(() => {
    const checkAuth = async () => {
      // First, check if auth is enabled on the server
      try {
        const healthResponse = await healthApi.check();
        const serverAuthEnabled = healthResponse.data.auth_enabled;
        setAuthEnabled(serverAuthEnabled);

        if (!serverAuthEnabled) {
          // Auth disabled on server - no need to check tokens
          setIsLoading(false);
          return;
        }
      } catch {
        // If health check fails, assume auth is enabled for safety
        setAuthEnabled(true);
      }

      // Auth is enabled - check for existing token
      const token = localStorage.getItem('token');
      if (!token) {
        setIsLoading(false);
        return;
      }

      try {
        const response = await authApi.me();
        setUser(response.data);
      } catch {
        // Token invalid or expired, try to refresh
        const refreshed = await tryRefreshToken();
        if (refreshed) {
          // Retry with new token
          try {
            const response = await authApi.me();
            setUser(response.data);
          } catch {
            // Still failed, clear tokens
            localStorage.removeItem('token');
            localStorage.removeItem('refreshToken');
          }
        } else {
          // Refresh failed, clear tokens
          localStorage.removeItem('token');
          localStorage.removeItem('refreshToken');
        }
      } finally {
        setIsLoading(false);
      }
    };

    checkAuth();
  }, [tryRefreshToken]);

  const login = useCallback(async (credentials: LoginCredentials) => {
    const response = await authApi.login(credentials);
    const { access_token, refresh_token } = response.data;

    localStorage.setItem('token', access_token);
    localStorage.setItem('refreshToken', refresh_token);

    // Fetch user info
    const userResponse = await authApi.me();
    setUser(userResponse.data);
  }, []);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      // Ignore errors during logout
    } finally {
      localStorage.removeItem('token');
      localStorage.removeItem('refreshToken');
      localStorage.removeItem('username');
      localStorage.removeItem('email');
      setUser(null);
    }
  }, []);

  const value = useMemo<AuthContextType>(
    () => ({
      user,
      isAuthenticated,
      isLoading,
      authEnabled,
      login,
      logout,
    }),
    [user, isAuthenticated, isLoading, authEnabled, login, logout]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
