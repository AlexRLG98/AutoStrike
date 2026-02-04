import axios, { HttpStatusCode, InternalAxiosRequestConfig } from 'axios';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for auth token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Flag to prevent infinite refresh loops
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
}> = [];

const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (token) {
      prom.resolve(token);
    } else {
      prom.reject(error);
    }
  });
  failedQueue = [];
};

// Helper: Clear tokens and redirect to login
const clearTokensAndRedirect = () => {
  localStorage.removeItem('token');
  localStorage.removeItem('refreshToken');
  if (globalThis.location?.pathname !== '/login') {
    globalThis.location.href = '/login';
  }
};

// Helper: Check if request should skip token refresh
const shouldSkipRefresh = (config: InternalAxiosRequestConfig | undefined): boolean => {
  if (!config) return true;
  const retryFlag = (config as InternalAxiosRequestConfig & { _retry?: boolean })._retry;
  if (retryFlag) return true;
  const url = config.url ?? '';
  return url.includes('/auth/refresh') || url.includes('/auth/login');
};

// Helper: Wait for ongoing refresh and retry request
const waitForRefreshAndRetry = async (
  originalRequest: InternalAxiosRequestConfig & { _retry?: boolean }
): Promise<unknown> => {
  const token = await new Promise<string>((resolve, reject) => {
    failedQueue.push({ resolve, reject });
  });
  originalRequest.headers.Authorization = `Bearer ${token}`;
  return api(originalRequest);
};

// Helper: Attempt token refresh
const attemptTokenRefresh = async (
  originalRequest: InternalAxiosRequestConfig & { _retry?: boolean }
): Promise<unknown> => {
  const refreshToken = localStorage.getItem('refreshToken');
  if (!refreshToken) {
    isRefreshing = false; // Reset flag to prevent hanging on subsequent 401s
    localStorage.removeItem('token');
    if (globalThis.location?.pathname !== '/login') {
      globalThis.location.href = '/login';
    }
    throw new Error('No refresh token');
  }

  try {
    const response = await api.post<TokenResponse>('/auth/refresh', {
      refresh_token: refreshToken,
    });
    const { access_token, refresh_token: newRefreshToken } = response.data;

    localStorage.setItem('token', access_token);
    localStorage.setItem('refreshToken', newRefreshToken);
    processQueue(null, access_token);

    originalRequest.headers.Authorization = `Bearer ${access_token}`;
    return api(originalRequest);
  } catch (refreshError) {
    processQueue(refreshError, null);
    clearTokensAndRedirect();
    throw refreshError;
  } finally {
    isRefreshing = false;
  }
};

// Response interceptor for error handling with automatic token refresh
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config as
      | (InternalAxiosRequestConfig & { _retry?: boolean })
      | undefined;

    if (error.response?.status !== HttpStatusCode.Unauthorized) {
      throw error;
    }

    if (shouldSkipRefresh(originalRequest)) {
      clearTokensAndRedirect();
      throw error;
    }

    if (isRefreshing && originalRequest) {
      return waitForRefreshAndRetry(originalRequest);
    }

    if (originalRequest) {
      originalRequest._retry = true;
      isRefreshing = true;
      return attemptTokenRefresh(originalRequest);
    }

    throw error;
  }
);

// Auth types
export interface LoginCredentials {
  username: string;
  password: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
}

export interface User {
  id: string;
  username: string;
  email: string;
  role: 'admin' | 'operator' | 'viewer';
  created_at: string;
  updated_at: string;
}

// Auth API methods
export const authApi = {
  /**
   * Login with username and password
   */
  login: (credentials: LoginCredentials) =>
    api.post<TokenResponse>('/auth/login', credentials),

  /**
   * Refresh access token using refresh token
   */
  refresh: (refreshToken: string) =>
    api.post<TokenResponse>('/auth/refresh', { refresh_token: refreshToken }),

  /**
   * Logout (client-side token removal)
   */
  logout: () => api.post('/auth/logout'),

  /**
   * Get current authenticated user
   */
  me: () => api.get<User>('/auth/me'),
};

// Execution API methods
export const executionApi = {
  /**
   * List all executions
   */
  list: () => api.get('/executions'),

  /**
   * Get execution by ID
   */
  get: (id: string) => api.get(`/executions/${id}`),

  /**
   * Get execution results
   */
  getResults: (id: string) => api.get(`/executions/${id}/results`),

  /**
   * Start a new execution
   */
  start: (scenarioId: string, agentPaws: string[], safeMode: boolean) =>
    api.post('/executions', {
      scenario_id: scenarioId,
      agent_paws: agentPaws,
      safe_mode: safeMode,
    }),

  /**
   * Stop a running execution
   */
  stop: (id: string) => api.post(`/executions/${id}/stop`),

  /**
   * Complete an execution
   */
  complete: (id: string) => api.post(`/executions/${id}/complete`),
};

// Health check response
export interface HealthResponse {
  status: string;
  auth_enabled: boolean;
}

// Health API (uses root path, not /api/v1)
export const healthApi = {
  /**
   * Check server health and auth status
   */
  check: () => axios.get<HealthResponse>('/health'),
};
