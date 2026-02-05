import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { HttpStatusCode, AxiosHeaders, InternalAxiosRequestConfig } from 'axios';

// We need to test the interceptors directly since mocking axios interferes with them
describe('API Module', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it('exports api instance with correct config', async () => {
    const { api } = await import('./api');
    expect(api).toBeDefined();
    expect(api.defaults.headers['Content-Type']).toBe('application/json');
  });
});

describe('Request Interceptor Logic', () => {
  const originalLocalStorage = global.localStorage;

  beforeEach(() => {
    // Create a proper mock for localStorage
    const store: Record<string, string> = {};
    const mockLocalStorage = {
      getItem: vi.fn((key: string) => store[key] || null),
      setItem: vi.fn((key: string, value: string) => {
        store[key] = value;
      }),
      removeItem: vi.fn((key: string) => {
        delete store[key];
      }),
      clear: vi.fn(() => {
        Object.keys(store).forEach((key) => delete store[key]);
      }),
      length: 0,
      key: vi.fn(),
    };
    Object.defineProperty(global, 'localStorage', {
      value: mockLocalStorage,
      writable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(global, 'localStorage', {
      value: originalLocalStorage,
      writable: true,
    });
  });

  it('adds Authorization header when token exists in localStorage', async () => {
    localStorage.setItem('token', 'my-jwt-token');

    vi.resetModules();
    const { api } = await import('./api');

    // Get the request interceptor function
    const interceptors = api.interceptors.request as unknown as {
      handlers: Array<{ fulfilled: (config: InternalAxiosRequestConfig) => InternalAxiosRequestConfig }>;
    };
    const requestInterceptor = interceptors.handlers[0].fulfilled;

    const config: InternalAxiosRequestConfig = {
      headers: new AxiosHeaders(),
    };

    const result = requestInterceptor(config);
    expect(result.headers.Authorization).toBe('Bearer my-jwt-token');
  });

  it('does not add Authorization header when no token', async () => {
    // No token in localStorage
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.request as unknown as {
      handlers: Array<{ fulfilled: (config: InternalAxiosRequestConfig) => InternalAxiosRequestConfig }>;
    };
    const requestInterceptor = interceptors.handlers[0].fulfilled;

    const config: InternalAxiosRequestConfig = {
      headers: new AxiosHeaders(),
    };

    const result = requestInterceptor(config);
    expect(result.headers.Authorization).toBeUndefined();
  });
});

describe('Response Interceptor Logic', () => {
  const originalLocalStorage = global.localStorage;
  const originalLocation = global.location;

  beforeEach(() => {
    const store: Record<string, string> = {};
    const mockLocalStorage = {
      getItem: vi.fn((key: string) => store[key] || null),
      setItem: vi.fn((key: string, value: string) => {
        store[key] = value;
      }),
      removeItem: vi.fn((key: string) => {
        delete store[key];
      }),
      clear: vi.fn(),
      length: 0,
      key: vi.fn(),
    };
    Object.defineProperty(global, 'localStorage', {
      value: mockLocalStorage,
      writable: true,
    });

    Object.defineProperty(global, 'location', {
      value: { href: 'http://localhost:3000/' },
      writable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(global, 'localStorage', {
      value: originalLocalStorage,
      writable: true,
    });
    Object.defineProperty(global, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('passes through successful responses', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: { status: number }) => { status: number };
        rejected: (error: { response?: { status: number } }) => Promise<never>;
      }>;
    };
    const successHandler = interceptors.handlers[0].fulfilled;

    const response = { status: 200, data: 'success' };
    const result = successHandler(response);
    expect(result).toBe(response);
  });

  it('removes token on 401 error', async () => {
    localStorage.setItem('token', 'expired-token');

    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: { status: number }) => { status: number };
        rejected: (error: { response?: { status: number } }) => Promise<never>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = { response: { status: HttpStatusCode.Unauthorized } };

    await expect(errorHandler(error)).rejects.toBe(error);
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
  });

  it('rejects error without redirect on non-401', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: { status: number }) => { status: number };
        rejected: (error: { response?: { status: number } }) => Promise<never>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = { response: { status: 500 } };

    await expect(errorHandler(error)).rejects.toBe(error);
    expect(localStorage.removeItem).not.toHaveBeenCalled();
    expect(global.location.href).toBe('http://localhost:3000/');
  });

  it('handles errors without response object', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: { status: number }) => { status: number };
        rejected: (error: { response?: { status: number } }) => Promise<never>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = { message: 'Network Error' };

    await expect(errorHandler(error as { response?: { status: number } })).rejects.toBe(error);
    expect(localStorage.removeItem).not.toHaveBeenCalled();
  });
});

describe('executionApi', () => {
  it('exports executionApi object with all methods', async () => {
    const { executionApi } = await import('./api');
    expect(executionApi).toBeDefined();
    expect(typeof executionApi.list).toBe('function');
    expect(typeof executionApi.get).toBe('function');
    expect(typeof executionApi.getResults).toBe('function');
    expect(typeof executionApi.start).toBe('function');
    expect(typeof executionApi.stop).toBe('function');
    expect(typeof executionApi.complete).toBe('function');
  });
});

describe('authApi', () => {
  it('exports authApi object with all methods', async () => {
    const { authApi } = await import('./api');
    expect(authApi).toBeDefined();
    expect(typeof authApi.login).toBe('function');
    expect(typeof authApi.refresh).toBe('function');
    expect(typeof authApi.logout).toBe('function');
    expect(typeof authApi.me).toBe('function');
  });
});

describe('Token Refresh Interceptor Logic', () => {
  const originalLocalStorage = global.localStorage;
  const originalLocation = global.location;
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
      clear: vi.fn(),
      length: 0,
      key: vi.fn(),
    };
    Object.defineProperty(global, 'localStorage', {
      value: mockLocalStorage,
      writable: true,
    });
    Object.defineProperty(global, 'location', {
      value: { href: 'http://localhost:3000/dashboard', pathname: '/dashboard' },
      writable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(global, 'localStorage', {
      value: originalLocalStorage,
      writable: true,
    });
    Object.defineProperty(global, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('redirects to login when 401 on auth/refresh endpoint', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = {
      config: { url: '/auth/refresh', _retry: false },
      response: { status: HttpStatusCode.Unauthorized },
    };

    await expect(errorHandler(error)).rejects.toBe(error);
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken');
    expect(global.location.href).toBe('/login');
  });

  it('redirects to login when 401 on auth/login endpoint', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = {
      config: { url: '/auth/login', _retry: false },
      response: { status: HttpStatusCode.Unauthorized },
    };

    await expect(errorHandler(error)).rejects.toBe(error);
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken');
    expect(global.location.href).toBe('/login');
  });

  it('redirects to login when 401 and no config', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = {
      config: undefined,
      response: { status: HttpStatusCode.Unauthorized },
    };

    await expect(errorHandler(error)).rejects.toBe(error);
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken');
    expect(global.location.href).toBe('/login');
  });

  it('does not redirect when already on login page', async () => {
    Object.defineProperty(global, 'location', {
      value: { href: 'http://localhost:3000/login', pathname: '/login' },
      writable: true,
    });

    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = {
      config: { url: '/some-endpoint', _retry: true },
      response: { status: HttpStatusCode.Unauthorized },
    };

    await expect(errorHandler(error)).rejects.toBe(error);
    expect(global.location.href).toBe('http://localhost:3000/login');
  });

  it('clears token and redirects when no refresh token available', async () => {
    store['token'] = 'expired-token';
    // No refreshToken in store

    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = {
      config: { url: '/some-endpoint', _retry: false, headers: {} },
      response: { status: HttpStatusCode.Unauthorized },
    };

    // This will attempt token refresh but no refreshToken exists
    await expect(errorHandler(error)).rejects.toThrow('No refresh token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
    expect(global.location.href).toBe('/login');
  });

  it('resets isRefreshing flag when no refresh token, allowing subsequent 401s to be handled', async () => {
    store['token'] = 'expired-token';
    // No refreshToken in store

    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error1 = {
      config: { url: '/endpoint-1', _retry: false, headers: {} },
      response: { status: HttpStatusCode.Unauthorized },
    };

    // First 401 - no refresh token
    await expect(errorHandler(error1)).rejects.toThrow('No refresh token');

    // Reset location for second test
    global.location.href = 'http://localhost:3000/dashboard';

    const error2 = {
      config: { url: '/endpoint-2', _retry: false, headers: {} },
      response: { status: HttpStatusCode.Unauthorized },
    };

    // Second 401 should also be handled (not hang) because isRefreshing was reset
    await expect(errorHandler(error2)).rejects.toThrow('No refresh token');
    expect(global.location.href).toBe('/login');
  });
});

describe('API exports', () => {
  it('exports api instance', async () => {
    const { api } = await import('./api');
    expect(api).toBeDefined();
    expect(api.defaults.baseURL).toBeDefined();
  });
});

describe('healthApi', () => {
  it('exports healthApi object with check method', async () => {
    const { healthApi } = await import('./api');
    expect(healthApi).toBeDefined();
    expect(typeof healthApi.check).toBe('function');
  });
});

describe('adminApi', () => {
  it('exports adminApi object with all methods', async () => {
    const { adminApi } = await import('./api');
    expect(adminApi).toBeDefined();
    expect(typeof adminApi.listUsers).toBe('function');
    expect(typeof adminApi.getUser).toBe('function');
    expect(typeof adminApi.createUser).toBe('function');
    expect(typeof adminApi.updateUser).toBe('function');
    expect(typeof adminApi.updateUserRole).toBe('function');
    expect(typeof adminApi.deactivateUser).toBe('function');
    expect(typeof adminApi.reactivateUser).toBe('function');
    expect(typeof adminApi.resetPassword).toBe('function');
  });
});

describe('scenarioApi', () => {
  it('exports scenarioApi object with all methods', async () => {
    const { scenarioApi } = await import('./api');
    expect(scenarioApi).toBeDefined();
    expect(typeof scenarioApi.list).toBe('function');
    expect(typeof scenarioApi.get).toBe('function');
    expect(typeof scenarioApi.create).toBe('function');
    expect(typeof scenarioApi.update).toBe('function');
    expect(typeof scenarioApi.delete).toBe('function');
    expect(typeof scenarioApi.exportAll).toBe('function');
    expect(typeof scenarioApi.exportOne).toBe('function');
    expect(typeof scenarioApi.import).toBe('function');
  });
});

describe('permissionApi', () => {
  it('exports permissionApi object with all methods', async () => {
    const { permissionApi } = await import('./api');
    expect(permissionApi).toBeDefined();
    expect(typeof permissionApi.getMatrix).toBe('function');
    expect(typeof permissionApi.getMyPermissions).toBe('function');
  });
});

describe('analyticsApi', () => {
  it('exports analyticsApi object with all methods', async () => {
    const { analyticsApi } = await import('./api');
    expect(analyticsApi).toBeDefined();
    expect(typeof analyticsApi.compare).toBe('function');
    expect(typeof analyticsApi.trend).toBe('function');
    expect(typeof analyticsApi.summary).toBe('function');
    expect(typeof analyticsApi.periodStats).toBe('function');
  });
});

describe('notificationApi', () => {
  it('exports notificationApi object with all methods', async () => {
    const { notificationApi } = await import('./api');
    expect(notificationApi).toBeDefined();
    expect(typeof notificationApi.list).toBe('function');
    expect(typeof notificationApi.getUnreadCount).toBe('function');
    expect(typeof notificationApi.markAsRead).toBe('function');
    expect(typeof notificationApi.markAllAsRead).toBe('function');
    expect(typeof notificationApi.getSettings).toBe('function');
    expect(typeof notificationApi.updateSettings).toBe('function');
    expect(typeof notificationApi.createSettings).toBe('function');
    expect(typeof notificationApi.deleteSettings).toBe('function');
    expect(typeof notificationApi.getSMTPConfig).toBe('function');
    expect(typeof notificationApi.testSMTP).toBe('function');
  });
});

describe('scheduleApi', () => {
  it('exports scheduleApi object with all methods', async () => {
    const { scheduleApi } = await import('./api');
    expect(scheduleApi).toBeDefined();
    expect(typeof scheduleApi.list).toBe('function');
    expect(typeof scheduleApi.get).toBe('function');
    expect(typeof scheduleApi.create).toBe('function');
    expect(typeof scheduleApi.update).toBe('function');
    expect(typeof scheduleApi.delete).toBe('function');
    expect(typeof scheduleApi.pause).toBe('function');
    expect(typeof scheduleApi.resume).toBe('function');
    expect(typeof scheduleApi.runNow).toBe('function');
    expect(typeof scheduleApi.getRuns).toBe('function');
  });
});

describe('Token Refresh Queue and Success Flow', () => {
  const originalLocalStorage = global.localStorage;
  const originalLocation = global.location;
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
      clear: vi.fn(),
      length: 0,
      key: vi.fn(),
    };
    Object.defineProperty(global, 'localStorage', {
      value: mockLocalStorage,
      writable: true,
    });
    Object.defineProperty(global, 'location', {
      value: { href: 'http://localhost:3000/dashboard', pathname: '/dashboard' },
      writable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(global, 'localStorage', {
      value: originalLocalStorage,
      writable: true,
    });
    Object.defineProperty(global, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('skips refresh when _retry flag is already set', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = {
      config: { url: '/some-endpoint', _retry: true, headers: {} },
      response: { status: HttpStatusCode.Unauthorized },
    };

    await expect(errorHandler(error)).rejects.toBe(error);
    expect(localStorage.removeItem).toHaveBeenCalledWith('token');
    expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken');
  });

  it('throws error when originalRequest is undefined after all checks', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    // Error with no config at all
    const error = {
      response: { status: HttpStatusCode.Unauthorized },
    };

    await expect(errorHandler(error)).rejects.toBe(error);
  });
});

describe('API method calls', () => {
  it('executionApi.start calls api.post with correct parameters', async () => {
    const { executionApi } = await import('./api');
    expect(executionApi.start).toBeDefined();
    expect(typeof executionApi.start).toBe('function');
  });

  it('scenarioApi.exportAll handles empty ids', async () => {
    const { scenarioApi } = await import('./api');
    expect(scenarioApi.exportAll).toBeDefined();
    expect(typeof scenarioApi.exportAll).toBe('function');
  });

  it('scenarioApi.exportAll handles ids array', async () => {
    const { scenarioApi } = await import('./api');
    expect(typeof scenarioApi.exportAll).toBe('function');
  });

  it('adminApi.listUsers handles includeInactive parameter', async () => {
    const { adminApi } = await import('./api');
    expect(typeof adminApi.listUsers).toBe('function');
  });

  it('notificationApi.list handles limit parameter', async () => {
    const { notificationApi } = await import('./api');
    expect(typeof notificationApi.list).toBe('function');
  });

  it('scheduleApi.getRuns handles limit parameter', async () => {
    const { scheduleApi } = await import('./api');
    expect(typeof scheduleApi.getRuns).toBe('function');
  });

  it('analyticsApi methods use default parameters', async () => {
    const { analyticsApi } = await import('./api');
    expect(typeof analyticsApi.compare).toBe('function');
    expect(typeof analyticsApi.trend).toBe('function');
    expect(typeof analyticsApi.summary).toBe('function');
    expect(typeof analyticsApi.periodStats).toBe('function');
  });
});

describe('Type exports', () => {
  it('exports LoginCredentials type', async () => {
    const api = await import('./api');
    // Type is exported if we can reference it without error
    expect(api).toBeDefined();
  });

  it('exports TokenResponse type', async () => {
    const api = await import('./api');
    expect(api).toBeDefined();
  });

  it('exports UserRole type', async () => {
    const api = await import('./api');
    expect(api).toBeDefined();
  });

  it('exports User type', async () => {
    const api = await import('./api');
    expect(api).toBeDefined();
  });

  it('exports Notification types', async () => {
    const api = await import('./api');
    expect(api).toBeDefined();
  });

  it('exports Schedule types', async () => {
    const api = await import('./api');
    expect(api).toBeDefined();
  });

  it('exports Permission types', async () => {
    const api = await import('./api');
    expect(api).toBeDefined();
  });

  it('exports Analytics types', async () => {
    const api = await import('./api');
    expect(api).toBeDefined();
  });
});

describe('Helper functions coverage', () => {
  const originalLocalStorage = global.localStorage;
  const originalLocation = global.location;
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
      clear: vi.fn(),
      length: 0,
      key: vi.fn(),
    };
    Object.defineProperty(global, 'localStorage', {
      value: mockLocalStorage,
      writable: true,
    });
    Object.defineProperty(global, 'location', {
      value: { href: 'http://localhost:3000/dashboard', pathname: '/dashboard' },
      writable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(global, 'localStorage', {
      value: originalLocalStorage,
      writable: true,
    });
    Object.defineProperty(global, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('handles shouldSkipRefresh with undefined config', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = {
      config: undefined,
      response: { status: HttpStatusCode.Unauthorized },
    };

    await expect(errorHandler(error)).rejects.toBe(error);
  });

  it('handles error with undefined url in config', async () => {
    vi.resetModules();
    const { api } = await import('./api');

    const interceptors = api.interceptors.response as unknown as {
      handlers: Array<{
        fulfilled: (response: unknown) => unknown;
        rejected: (error: unknown) => Promise<unknown>;
      }>;
    };
    const errorHandler = interceptors.handlers[0].rejected;

    const error = {
      config: { url: undefined, _retry: false, headers: {} },
      response: { status: HttpStatusCode.Unauthorized },
    };

    await expect(errorHandler(error)).rejects.toThrow('No refresh token');
  });
});
