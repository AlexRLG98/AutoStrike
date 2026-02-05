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

export type UserRole = 'admin' | 'rssi' | 'operator' | 'analyst' | 'viewer';

export interface User {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  is_active: boolean;
  last_login_at?: string;
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

// Admin types
export interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  role: UserRole;
}

export interface UpdateUserRequest {
  username?: string;
  email?: string;
}

export interface UpdateRoleRequest {
  role: UserRole;
}

export interface ResetPasswordRequest {
  new_password: string;
}

export interface ListUsersResponse {
  users: User[];
  total: number;
}

// Admin API methods (requires admin role)
export const adminApi = {
  /**
   * List all users
   */
  listUsers: (includeInactive = false) =>
    api.get<ListUsersResponse>('/admin/users', {
      params: includeInactive ? { include_inactive: 'true' } : undefined,
    }),

  /**
   * Get a specific user by ID
   */
  getUser: (id: string) => api.get<User>(`/admin/users/${id}`),

  /**
   * Create a new user
   */
  createUser: (data: CreateUserRequest) => api.post<User>('/admin/users', data),

  /**
   * Update a user's basic info
   */
  updateUser: (id: string, data: UpdateUserRequest) =>
    api.put<User>(`/admin/users/${id}`, data),

  /**
   * Update a user's role
   */
  updateUserRole: (id: string, data: UpdateRoleRequest) =>
    api.put<User>(`/admin/users/${id}/role`, data),

  /**
   * Deactivate a user (soft delete)
   */
  deactivateUser: (id: string) => api.delete(`/admin/users/${id}`),

  /**
   * Reactivate a deactivated user
   */
  reactivateUser: (id: string) => api.post<User>(`/admin/users/${id}/reactivate`),

  /**
   * Reset a user's password
   */
  resetPassword: (id: string, data: ResetPasswordRequest) =>
    api.post(`/admin/users/${id}/reset-password`, data),
};

// Technique types
export interface Technique {
  id: string;
  name: string;
  description: string;
  tactic: string;
  platforms: string[];
  executors: TechniqueExecutor[];
  detection: TechniqueDetection[];
  is_safe: boolean;
}

export interface TechniqueExecutor {
  type: string;
  command: string;
  platform?: string;
  timeout: number;
}

export interface TechniqueDetection {
  source: string;
  indicator: string;
}

export interface ImportTechniquesRequest {
  techniques: Technique[];
}

export interface ImportTechniquesResponse {
  imported: number;
  failed: number;
  errors?: string[];
}

// Technique API methods
export const techniqueApi = {
  /**
   * List all techniques
   */
  list: () => api.get<Technique[]>('/techniques'),

  /**
   * Get technique by ID
   */
  get: (id: string) => api.get<Technique>(`/techniques/${id}`),

  /**
   * Get techniques by tactic
   */
  getByTactic: (tactic: string) => api.get<Technique[]>(`/techniques/tactic/${tactic}`),

  /**
   * Get techniques by platform
   */
  getByPlatform: (platform: string) => api.get<Technique[]>(`/techniques/platform/${platform}`),

  /**
   * Get MITRE ATT&CK coverage statistics
   */
  getCoverage: () => api.get<Record<string, number>>('/techniques/coverage'),

  /**
   * Import techniques from JSON
   */
  import: (techniques: Technique[]) =>
    api.post<ImportTechniquesResponse>('/techniques/import/json', { techniques }),
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

// Scenario Import/Export types
export interface ScenarioExport {
  version: string;
  exported_at: string;
  scenarios: Scenario[];
}

export interface ImportScenarioRequest {
  name: string;
  description?: string;
  phases: ScenarioPhase[];
  tags?: string[];
}

export interface ImportScenariosRequest {
  version?: string;
  scenarios: ImportScenarioRequest[];
}

export interface ImportScenariosResponse {
  imported: number;
  failed: number;
  errors?: string[];
  scenarios: Scenario[];
}

// Need to import Scenario type - define locally for API
export interface Scenario {
  id: string;
  name: string;
  description: string;
  phases: ScenarioPhase[];
  tags: string[];
}

export interface ScenarioPhase {
  name: string;
  techniques: string[];
  order?: number;
}

// Scenario API methods
export const scenarioApi = {
  /**
   * List all scenarios
   */
  list: () => api.get<Scenario[]>('/scenarios'),

  /**
   * Get scenario by ID
   */
  get: (id: string) => api.get<Scenario>(`/scenarios/${id}`),

  /**
   * Create a new scenario
   */
  create: (data: Omit<Scenario, 'id'>) => api.post<Scenario>('/scenarios', data),

  /**
   * Update a scenario
   */
  update: (id: string, data: Omit<Scenario, 'id'>) =>
    api.put<Scenario>(`/scenarios/${id}`, data),

  /**
   * Delete a scenario
   */
  delete: (id: string) => api.delete(`/scenarios/${id}`),

  /**
   * Export all scenarios (or specific ones by IDs)
   */
  exportAll: (ids?: string[]) => {
    const params = ids?.length ? { ids: ids.join(',') } : undefined;
    return api.get<ScenarioExport>('/scenarios/export', { params });
  },

  /**
   * Export a single scenario
   */
  exportOne: (id: string) => api.get<ScenarioExport>(`/scenarios/${id}/export`),

  /**
   * Import scenarios from JSON
   */
  import: (data: ImportScenariosRequest) =>
    api.post<ImportScenariosResponse>('/scenarios/import', data),
};

// Permission types
export interface PermissionInfo {
  permission: string;
  name: string;
  description: string;
  category: string;
}

export interface PermissionCategory {
  name: string;
  description: string;
  permissions: string[];
}

export interface PermissionMatrix {
  roles: UserRole[];
  categories: PermissionCategory[];
  permissions: PermissionInfo[];
  matrix: Record<UserRole, string[]>;
}

export interface MyPermissionsResponse {
  role: string;
  permissions: string[];
}

export interface RoleInfo {
  role: string;
  display_name: string;
  permissions: string[];
}

export interface RolesResponse {
  roles: RoleInfo[];
}

// Permission API methods
export const permissionApi = {
  /**
   * Get the complete permission matrix
   */
  getMatrix: () => api.get<PermissionMatrix>('/permissions/matrix'),

  /**
   * Get current user's permissions
   */
  getMyPermissions: () => api.get<MyPermissionsResponse>('/permissions/me'),
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

// Analytics types
export interface PeriodStats {
  period: string;
  start_date: string;
  end_date: string;
  execution_count: number;
  average_score: number;
  total_blocked: number;
  total_detected: number;
  total_successful: number;
  total_techniques: number;
  score_by_tactic?: Record<string, number>;
}

export interface ScoreComparison {
  current: PeriodStats;
  previous: PeriodStats;
  score_change: number;
  score_trend: 'improving' | 'declining' | 'stable';
  blocked_change: number;
  detected_change: number;
}

export interface TrendDataPoint {
  date: string;
  average_score: number;
  execution_count: number;
  blocked: number;
  detected: number;
  successful: number;
}

export interface TrendSummary {
  start_score: number;
  end_score: number;
  average_score: number;
  max_score: number;
  min_score: number;
  total_executions: number;
  overall_trend: 'improving' | 'declining' | 'stable';
  percentage_change: number;
}

export interface ScoreTrend {
  period: string;
  data_points: TrendDataPoint[];
  summary: TrendSummary;
}

export interface ExecutionSummary {
  total_executions: number;
  completed_executions: number;
  average_score: number;
  best_score: number;
  worst_score: number;
  scores_by_scenario: Record<string, number>;
  executions_by_status: Record<string, number>;
}

// Analytics API methods
export const analyticsApi = {
  /**
   * Compare scores between current and previous period
   */
  compare: (days: number = 7) =>
    api.get<ScoreComparison>('/analytics/comparison', { params: { days } }),

  /**
   * Get score trend over time
   */
  trend: (days: number = 30) =>
    api.get<ScoreTrend>('/analytics/trend', { params: { days } }),

  /**
   * Get execution summary
   */
  summary: (days: number = 30) =>
    api.get<ExecutionSummary>('/analytics/summary', { params: { days } }),

  /**
   * Get period statistics
   */
  periodStats: (start: string, end: string) =>
    api.get<PeriodStats>('/analytics/period', { params: { start, end } }),
};

// Notification types
export type NotificationType =
  | 'execution_started'
  | 'execution_completed'
  | 'execution_failed'
  | 'score_alert'
  | 'agent_offline';

export type NotificationChannel = 'email' | 'webhook';

export interface Notification {
  id: string;
  user_id: string;
  type: NotificationType;
  title: string;
  message: string;
  data?: Record<string, unknown>;
  read: boolean;
  sent_at?: string;
  created_at: string;
}

export interface NotificationSettings {
  id: string;
  user_id: string;
  channel: NotificationChannel;
  enabled: boolean;
  email_address?: string;
  webhook_url?: string;
  notify_on_start: boolean;
  notify_on_complete: boolean;
  notify_on_failure: boolean;
  notify_on_score_alert: boolean;
  score_alert_threshold: number;
  notify_on_agent_offline: boolean;
  created_at: string;
  updated_at: string;
}

export interface NotificationSettingsRequest {
  channel: NotificationChannel;
  enabled: boolean;
  email_address?: string;
  webhook_url?: string;
  notify_on_start: boolean;
  notify_on_complete: boolean;
  notify_on_failure: boolean;
  notify_on_score_alert: boolean;
  score_alert_threshold: number;
  notify_on_agent_offline: boolean;
}

export interface SMTPConfig {
  host: string;
  port: number;
  username: string;
  from: string;
  use_tls: boolean;
}

export interface UnreadCountResponse {
  count: number;
}

// Notification API methods
export const notificationApi = {
  /**
   * Get notifications for current user
   */
  list: (limit: number = 50) =>
    api.get<Notification[]>('/notifications', { params: { limit } }),

  /**
   * Get unread notification count
   */
  getUnreadCount: () => api.get<UnreadCountResponse>('/notifications/unread/count'),

  /**
   * Mark a notification as read
   */
  markAsRead: (id: string) => api.post(`/notifications/${id}/read`),

  /**
   * Mark all notifications as read
   */
  markAllAsRead: () => api.post('/notifications/read-all'),

  /**
   * Get notification settings
   */
  getSettings: () => api.get<NotificationSettings>('/notifications/settings'),

  /**
   * Create notification settings
   */
  createSettings: (data: NotificationSettingsRequest) =>
    api.post<NotificationSettings>('/notifications/settings', data),

  /**
   * Update notification settings
   */
  updateSettings: (id: string, data: NotificationSettingsRequest) =>
    api.put<NotificationSettings>(`/notifications/settings/${id}`, data),

  /**
   * Delete notification settings
   */
  deleteSettings: (id: string) => api.delete(`/notifications/settings/${id}`),

  /**
   * Get SMTP configuration
   */
  getSMTPConfig: () => api.get<SMTPConfig>('/notifications/smtp'),

  /**
   * Test SMTP connection
   */
  testSMTP: (email: string) =>
    api.post('/notifications/smtp/test', { email }),
};

// Schedule types
export type ScheduleFrequency = 'once' | 'hourly' | 'daily' | 'weekly' | 'monthly' | 'cron';
export type ScheduleStatus = 'active' | 'paused' | 'disabled';

export interface Schedule {
  id: string;
  name: string;
  description: string;
  scenario_id: string;
  agent_paw: string;
  frequency: ScheduleFrequency;
  cron_expr: string;
  safe_mode: boolean;
  status: ScheduleStatus;
  next_run_at: string | null;
  last_run_at: string | null;
  last_run_id: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface ScheduleRun {
  id: string;
  schedule_id: string;
  execution_id: string;
  started_at: string;
  completed_at: string | null;
  status: string;
  error: string;
}

export interface CreateScheduleRequest {
  name: string;
  description?: string;
  scenario_id: string;
  agent_paw?: string;
  frequency: ScheduleFrequency;
  cron_expr?: string;
  safe_mode: boolean;
  start_at?: string;
}

// Schedule API methods
export const scheduleApi = {
  /**
   * List all schedules
   */
  list: () => api.get<Schedule[]>('/schedules'),

  /**
   * Get schedule by ID
   */
  get: (id: string) => api.get<Schedule>(`/schedules/${id}`),

  /**
   * Create a new schedule
   */
  create: (data: CreateScheduleRequest) => api.post<Schedule>('/schedules', data),

  /**
   * Update a schedule
   */
  update: (id: string, data: CreateScheduleRequest) =>
    api.put<Schedule>(`/schedules/${id}`, data),

  /**
   * Delete a schedule
   */
  delete: (id: string) => api.delete(`/schedules/${id}`),

  /**
   * Pause a schedule
   */
  pause: (id: string) => api.post<Schedule>(`/schedules/${id}/pause`),

  /**
   * Resume a paused schedule
   */
  resume: (id: string) => api.post<Schedule>(`/schedules/${id}/resume`),

  /**
   * Run a schedule immediately
   */
  runNow: (id: string) => api.post<ScheduleRun>(`/schedules/${id}/run`),

  /**
   * Get schedule runs history
   */
  getRuns: (id: string, limit: number = 20) =>
    api.get<ScheduleRun[]>(`/schedules/${id}/runs`, { params: { limit } }),
};
