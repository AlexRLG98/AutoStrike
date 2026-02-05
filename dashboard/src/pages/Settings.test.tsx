import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Settings from './Settings';

// Mock react-hot-toast
vi.mock('react-hot-toast', () => ({
  default: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

// Mock the notification API
vi.mock('../lib/api', () => ({
  notificationApi: {
    getSettings: vi.fn(() => Promise.reject({ response: { status: 404 } })),
    getSMTPConfig: vi.fn(() => Promise.reject({ response: { status: 404 } })),
    createSettings: vi.fn(() => Promise.resolve({ data: {} })),
    updateSettings: vi.fn(() => Promise.resolve({ data: {} })),
    testSMTP: vi.fn(() => Promise.resolve({ data: {} })),
  },
}));

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
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
  };
})();
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
}

function renderSettings() {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Settings />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('Settings Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('renders settings title', () => {
    renderSettings();
    expect(screen.getByText('Settings')).toBeInTheDocument();
  });

  it('renders all configuration sections', async () => {
    renderSettings();
    expect(screen.getByText('Server Configuration')).toBeInTheDocument();
    expect(screen.getByText('Execution Settings')).toBeInTheDocument();
    expect(screen.getByText('Agent Settings')).toBeInTheDocument();
    expect(screen.getByText('TLS / mTLS Configuration')).toBeInTheDocument();
  });

  it('renders notification settings section', () => {
    renderSettings();
    expect(screen.getByText('Notification Settings')).toBeInTheDocument();
  });

  it('renders server URL input with default value', async () => {
    renderSettings();
    await waitFor(() => {
      const serverUrlInput = screen.getByLabelText('Server URL');
      expect(serverUrlInput).toHaveValue('https://localhost:8443');
    });
  });

  it('renders heartbeat interval input with default value', async () => {
    renderSettings();
    await waitFor(() => {
      const heartbeatInput = screen.getByLabelText('Heartbeat Interval (seconds)');
      expect(heartbeatInput).toHaveValue(30);
    });
  });

  it('renders stale timeout input with default value', async () => {
    renderSettings();
    await waitFor(() => {
      const staleInput = screen.getByLabelText('Stale Agent Timeout (seconds)');
      expect(staleInput).toHaveValue(120);
    });
  });

  it('renders safe mode toggle', () => {
    renderSettings();
    expect(screen.getByText('Safe Mode by Default')).toBeInTheDocument();
  });

  it('renders TLS certificate labels', () => {
    renderSettings();
    expect(screen.getByText('CA Certificate Path')).toBeInTheDocument();
    expect(screen.getByText('Server Certificate Path')).toBeInTheDocument();
    expect(screen.getByText('Server Key Path')).toBeInTheDocument();
  });

  it('renders save button', () => {
    renderSettings();
    expect(screen.getByText('Save Settings')).toBeInTheDocument();
  });
});

describe('Settings Form Interactions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('updates server URL on input change', async () => {
    renderSettings();
    await waitFor(() => {
      const serverUrlInput = screen.getByLabelText('Server URL');
      fireEvent.change(serverUrlInput, { target: { value: 'https://newserver:8443' } });
      expect(serverUrlInput).toHaveValue('https://newserver:8443');
    });
  });

  it('updates heartbeat interval on input change', async () => {
    renderSettings();
    await waitFor(() => {
      const heartbeatInput = screen.getByLabelText('Heartbeat Interval (seconds)');
      fireEvent.change(heartbeatInput, { target: { value: '60' } });
      expect(heartbeatInput).toHaveValue(60);
    });
  });

  it('saves settings to localStorage on save button click', async () => {
    const toast = await import('react-hot-toast');
    renderSettings();

    const saveButton = screen.getByText('Save Settings');
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'autostrike_settings',
        expect.any(String)
      );
      expect(toast.default.success).toHaveBeenCalledWith('Settings saved successfully');
    });
  });

  it('loads settings from localStorage on mount', async () => {
    const savedSettings = {
      serverUrl: 'https://custom:9999',
      heartbeatInterval: 45,
      staleTimeout: 180,
    };
    localStorageMock.getItem.mockReturnValue(JSON.stringify(savedSettings));

    renderSettings();

    await waitFor(() => {
      const serverUrlInput = screen.getByLabelText('Server URL');
      expect(serverUrlInput).toHaveValue('https://custom:9999');
    });
  });
});

describe('Settings Default Values', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
    localStorageMock.getItem.mockReturnValue(null);
  });

  it('uses default heartbeat interval when invalid input', async () => {
    renderSettings();
    await waitFor(() => {
      const heartbeatInput = screen.getByLabelText('Heartbeat Interval (seconds)');
      fireEvent.change(heartbeatInput, { target: { value: '' } });
      expect(heartbeatInput).toHaveValue(30);
    });
  });

  it('uses default stale timeout when invalid input', async () => {
    renderSettings();
    await waitFor(() => {
      const staleInput = screen.getByLabelText('Stale Agent Timeout (seconds)');
      fireEvent.change(staleInput, { target: { value: '' } });
      expect(staleInput).toHaveValue(120);
    });
  });

  it('handles malformed JSON in localStorage gracefully', async () => {
    localStorageMock.getItem.mockReturnValue('invalid-json');

    // Should not throw
    expect(() => renderSettings()).not.toThrow();

    // Should use defaults
    await waitFor(() => {
      const serverUrlInput = screen.getByLabelText('Server URL');
      expect(serverUrlInput).toHaveValue('https://localhost:8443');
    });
  });

  it('updates CA certificate path on input change', async () => {
    renderSettings();
    await waitFor(() => {
      const caCertInput = screen.getByLabelText('CA Certificate Path');
      fireEvent.change(caCertInput, { target: { value: '/path/to/ca.crt' } });
      expect(caCertInput).toHaveValue('/path/to/ca.crt');
    });
  });

  it('updates server certificate path on input change', async () => {
    renderSettings();
    await waitFor(() => {
      const serverCertInput = screen.getByLabelText('Server Certificate Path');
      fireEvent.change(serverCertInput, { target: { value: '/path/to/server.crt' } });
      expect(serverCertInput).toHaveValue('/path/to/server.crt');
    });
  });

  it('updates server key path on input change', async () => {
    renderSettings();
    await waitFor(() => {
      const serverKeyInput = screen.getByLabelText('Server Key Path');
      fireEvent.change(serverKeyInput, { target: { value: '/path/to/server.key' } });
      expect(serverKeyInput).toHaveValue('/path/to/server.key');
    });
  });

  it('updates stale timeout on valid input change', async () => {
    renderSettings();
    await waitFor(() => {
      const staleInput = screen.getByLabelText('Stale Agent Timeout (seconds)');
      fireEvent.change(staleInput, { target: { value: '300' } });
      expect(staleInput).toHaveValue(300);
    });
  });
});

describe('Notification Settings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('renders notification settings section', () => {
    renderSettings();
    expect(screen.getByText('Notification Settings')).toBeInTheDocument();
  });

  it('renders enable notifications toggle after loading', async () => {
    renderSettings();
    await waitFor(() => {
      expect(screen.getByText('Enable Notifications')).toBeInTheDocument();
    });
  });

  it('renders notification settings description after loading', async () => {
    renderSettings();
    await waitFor(() => {
      expect(screen.getByText('Receive notifications for execution events')).toBeInTheDocument();
    });
  });

  it('shows notification channel options when enabled', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        channel: 'email',
        enabled: true,
        email_address: 'test@example.com',
        notify_on_start: false,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: true,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Notification Channel')).toBeInTheDocument();
    });
  });

  it('shows email input when email channel selected', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        channel: 'email',
        enabled: true,
        email_address: 'test@example.com',
        notify_on_start: false,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: true,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Email Address')).toBeInTheDocument();
    });
  });

  it('shows notification type toggles when enabled', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        channel: 'email',
        enabled: true,
        email_address: 'test@example.com',
        notify_on_start: false,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: true,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Notify me when:')).toBeInTheDocument();
      expect(screen.getByText('Execution starts')).toBeInTheDocument();
      expect(screen.getByText('Execution completes')).toBeInTheDocument();
      expect(screen.getByText('Execution fails')).toBeInTheDocument();
      expect(screen.getByText('Agent goes offline')).toBeInTheDocument();
    });
  });

  it('shows save notification settings button when enabled', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        channel: 'email',
        enabled: true,
        email_address: 'test@example.com',
        notify_on_start: false,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: true,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Save Notification Settings')).toBeInTheDocument();
    });
  });

  it('saves notification settings on button click', async () => {
    const { notificationApi } = await import('../lib/api');
    const toast = await import('react-hot-toast');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        id: 'settings-123',
        user_id: 'user-123',
        channel: 'email',
        enabled: true,
        email_address: 'test@example.com',
        notify_on_start: false,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: true,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    } as never);
    vi.mocked(notificationApi.updateSettings).mockResolvedValue({ data: {} } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Save Notification Settings')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Save Notification Settings'));

    await waitFor(() => {
      expect(notificationApi.updateSettings).toHaveBeenCalled();
      expect(toast.default.success).toHaveBeenCalledWith('Notification settings saved');
    });
  });

  it('shows error toast when save fails', async () => {
    const { notificationApi } = await import('../lib/api');
    const toast = await import('react-hot-toast');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        id: 'settings-456',
        user_id: 'user-123',
        channel: 'email',
        enabled: true,
        email_address: '',
        notify_on_start: false,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: true,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    } as never);
    vi.mocked(notificationApi.updateSettings).mockRejectedValue(new Error('Save failed'));

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Save Notification Settings')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Save Notification Settings'));

    await waitFor(() => {
      expect(toast.default.error).toHaveBeenCalledWith('Failed to save notification settings');
    });
  });
});

describe('SMTP Test', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('shows SMTP test section when SMTP is configured', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSMTPConfig).mockResolvedValue({
      data: {
        host: 'smtp.example.com',
        port: 587,
        use_tls: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Test Email')).toBeInTheDocument();
    });
  });

  it('sends test email on button click', async () => {
    const { notificationApi } = await import('../lib/api');
    const toast = await import('react-hot-toast');
    vi.mocked(notificationApi.getSMTPConfig).mockResolvedValue({
      data: {
        host: 'smtp.example.com',
        port: 587,
        use_tls: true,
      },
    } as never);
    vi.mocked(notificationApi.testSMTP).mockResolvedValue({ data: {} } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Test Email')).toBeInTheDocument();
    });

    const emailInput = screen.getByPlaceholderText('Enter email to send test');
    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });

    fireEvent.click(screen.getByText('Send Test'));

    await waitFor(() => {
      expect(notificationApi.testSMTP).toHaveBeenCalledWith('test@example.com');
      expect(toast.default.success).toHaveBeenCalledWith('Test email sent successfully');
    });
  });

  it('shows error when test email fails', async () => {
    const { notificationApi } = await import('../lib/api');
    const toast = await import('react-hot-toast');
    vi.mocked(notificationApi.getSMTPConfig).mockResolvedValue({
      data: {
        host: 'smtp.example.com',
        port: 587,
        use_tls: true,
      },
    } as never);
    vi.mocked(notificationApi.testSMTP).mockRejectedValue(new Error('SMTP error'));

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Test Email')).toBeInTheDocument();
    });

    const emailInput = screen.getByPlaceholderText('Enter email to send test');
    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });

    fireEvent.click(screen.getByText('Send Test'));

    await waitFor(() => {
      expect(toast.default.error).toHaveBeenCalledWith('Failed to send test email');
    });
  });

  it('shows error when no email entered', async () => {
    const { notificationApi } = await import('../lib/api');
    const toast = await import('react-hot-toast');
    vi.mocked(notificationApi.getSMTPConfig).mockResolvedValue({
      data: {
        host: 'smtp.example.com',
        port: 587,
        use_tls: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Test Email')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Send Test'));

    await waitFor(() => {
      expect(toast.default.error).toHaveBeenCalledWith('Please enter an email address');
    });
  });
});

describe('Safe Mode Toggle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('renders safe mode toggle with description', () => {
    renderSettings();
    expect(screen.getByText('Safe Mode by Default')).toBeInTheDocument();
    expect(screen.getByText("Only run safe techniques that don't modify the system")).toBeInTheDocument();
  });
});

describe('Settings Server Configuration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
    localStorageMock.getItem.mockReturnValue(null);
  });

  it('renders server configuration section', () => {
    renderSettings();
    expect(screen.getByText('Server Configuration')).toBeInTheDocument();
  });

  it('renders execution settings section', () => {
    renderSettings();
    expect(screen.getByText('Execution Settings')).toBeInTheDocument();
  });

  it('renders agent settings section', () => {
    renderSettings();
    expect(screen.getByText('Agent Settings')).toBeInTheDocument();
  });
});

describe('Settings Webhook Configuration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('shows webhook channel option when notifications enabled', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        channel: 'webhook',
        enabled: true,
        email_address: '',
        webhook_url: 'https://example.com/webhook',
        notify_on_start: false,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: true,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Notification Channel')).toBeInTheDocument();
    });
  });
});

describe('Settings Score Alert Threshold', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('shows notification settings when enabled', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        channel: 'email',
        enabled: true,
        email_address: 'test@example.com',
        notify_on_start: false,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: true,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Notification Settings')).toBeInTheDocument();
    });
  });
});

describe('Settings Notification Toggle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('toggles notification enabled state', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        channel: 'email',
        enabled: true,
        email_address: 'test@example.com',
        notify_on_start: false,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: true,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Enable Notifications')).toBeInTheDocument();
    });
  });

  it('shows expanded settings when notifications enabled', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockResolvedValue({
      data: {
        channel: 'email',
        enabled: true,
        email_address: 'test@example.com',
        notify_on_start: true,
        notify_on_complete: true,
        notify_on_failure: true,
        notify_on_score_alert: false,
        score_alert_threshold: 70,
        notify_on_agent_offline: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Execution starts')).toBeInTheDocument();
    });
  });
});

describe('Settings Create Notification Settings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('shows enable notifications toggle when no settings exist', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockRejectedValue({ response: { status: 404 } });

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText('Enable Notifications')).toBeInTheDocument();
    });
  });
});

describe('Settings Form Persistence', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('persists server URL across page reload', async () => {
    const savedSettings = {
      serverUrl: 'https://myserver:9000',
      heartbeatInterval: 60,
      staleTimeout: 240,
      safeMode: false,
    };
    localStorageMock.getItem.mockReturnValue(JSON.stringify(savedSettings));

    renderSettings();

    await waitFor(() => {
      const serverUrlInput = screen.getByLabelText('Server URL');
      expect(serverUrlInput).toHaveValue('https://myserver:9000');
    });
  });
});

describe('Settings TLS Configuration Details', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('shows TLS section collapsed by default', () => {
    renderSettings();
    expect(screen.getByText('TLS / mTLS Configuration')).toBeInTheDocument();
  });

  it('allows input in all TLS fields', async () => {
    renderSettings();
    await waitFor(() => {
      const caCertInput = screen.getByLabelText('CA Certificate Path');
      const serverCertInput = screen.getByLabelText('Server Certificate Path');
      const serverKeyInput = screen.getByLabelText('Server Key Path');

      fireEvent.change(caCertInput, { target: { value: '/custom/ca.crt' } });
      fireEvent.change(serverCertInput, { target: { value: '/custom/server.crt' } });
      fireEvent.change(serverKeyInput, { target: { value: '/custom/server.key' } });

      expect(caCertInput).toHaveValue('/custom/ca.crt');
      expect(serverCertInput).toHaveValue('/custom/server.crt');
      expect(serverKeyInput).toHaveValue('/custom/server.key');
    });
  });
});

describe('Settings SMTP Configuration Display', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('shows SMTP host when configured', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSMTPConfig).mockResolvedValue({
      data: {
        host: 'smtp.example.com',
        port: 587,
        username: 'user@example.com',
        use_tls: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText(/smtp.example.com/i)).toBeInTheDocument();
    });
  });

  it('shows TLS status when SMTP configured', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSMTPConfig).mockResolvedValue({
      data: {
        host: 'smtp.example.com',
        port: 587,
        username: 'user@example.com',
        use_tls: true,
      },
    } as never);

    renderSettings();

    await waitFor(() => {
      expect(screen.getByText(/TLS/i)).toBeInTheDocument();
    });
  });
});

describe('Settings Error States', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('handles notification settings fetch error', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSettings).mockRejectedValue(new Error('Network error'));

    // Should not crash
    expect(() => renderSettings()).not.toThrow();
  });

  it('handles SMTP config fetch error', async () => {
    const { notificationApi } = await import('../lib/api');
    vi.mocked(notificationApi.getSMTPConfig).mockRejectedValue(new Error('Network error'));

    // Should not crash
    expect(() => renderSettings()).not.toThrow();
  });
});
