import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import {
  notificationApi,
  NotificationSettingsRequest,
  NotificationChannel,
} from '../lib/api';
import { BellIcon, EnvelopeIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline';

// Configuration constants
const DEFAULT_SERVER_URL = import.meta.env.VITE_SERVER_URL || 'https://localhost:8443';
const DEFAULT_HEARTBEAT_INTERVAL = 30;
const DEFAULT_STALE_TIMEOUT = 120;
const STORAGE_KEY = 'autostrike_settings';

interface LocalSettings {
  serverUrl: string;
  safeMode: boolean;
  heartbeatInterval: number;
  staleTimeout: number;
  caCertPath: string;
  serverCertPath: string;
  serverKeyPath: string;
}

const defaultSettings: LocalSettings = {
  serverUrl: DEFAULT_SERVER_URL,
  safeMode: true,
  heartbeatInterval: DEFAULT_HEARTBEAT_INTERVAL,
  staleTimeout: DEFAULT_STALE_TIMEOUT,
  caCertPath: '',
  serverCertPath: '',
  serverKeyPath: '',
};

const defaultNotificationSettings: NotificationSettingsRequest = {
  channel: 'email',
  enabled: false,
  email_address: '',
  webhook_url: '',
  notify_on_start: false,
  notify_on_complete: true,
  notify_on_failure: true,
  notify_on_score_alert: true,
  score_alert_threshold: 70,
  notify_on_agent_offline: true,
};

// Toggle component defined outside of Settings to avoid recreation on render
function Toggle({
  enabled,
  onChange,
  disabled = false,
}: {
  readonly enabled: boolean;
  readonly onChange: () => void;
  readonly disabled?: boolean;
}) {
  return (
    <button
      onClick={onChange}
      disabled={disabled}
      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
        enabled ? 'bg-primary-600' : 'bg-gray-200'
      } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
    >
      <span
        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
          enabled ? 'translate-x-6' : 'translate-x-1'
        }`}
      />
    </button>
  );
}

export default function Settings() {
  const queryClient = useQueryClient();
  const [settings, setSettings] = useState<LocalSettings>(defaultSettings);
  const [notifSettings, setNotifSettings] = useState<NotificationSettingsRequest>(defaultNotificationSettings);
  const [hasExistingNotifSettings, setHasExistingNotifSettings] = useState(false);
  const [testEmail, setTestEmail] = useState('');

  // Load local settings from localStorage on mount
  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        setSettings({ ...defaultSettings, ...parsed });
      } catch {
        // Use defaults if parsing fails
      }
    }
  }, []);

  // Fetch notification settings
  const { data: notificationSettingsData, isLoading: notifLoading } = useQuery({
    queryKey: ['notificationSettings'],
    queryFn: async () => {
      try {
        const response = await notificationApi.getSettings();
        return response.data;
      } catch (error: unknown) {
        // 404 means no settings exist yet
        if ((error as { response?: { status: number } })?.response?.status === 404) {
          return null;
        }
        throw error;
      }
    },
  });

  // Fetch SMTP config
  const { data: smtpConfig } = useQuery({
    queryKey: ['smtpConfig'],
    queryFn: async () => {
      try {
        const response = await notificationApi.getSMTPConfig();
        return response.data;
      } catch {
        return null;
      }
    },
  });

  // Update local state when notification settings are loaded
  useEffect(() => {
    if (notificationSettingsData) {
      setHasExistingNotifSettings(true);
      setNotifSettings({
        channel: notificationSettingsData.channel,
        enabled: notificationSettingsData.enabled,
        email_address: notificationSettingsData.email_address || '',
        webhook_url: notificationSettingsData.webhook_url || '',
        notify_on_start: notificationSettingsData.notify_on_start,
        notify_on_complete: notificationSettingsData.notify_on_complete,
        notify_on_failure: notificationSettingsData.notify_on_failure,
        notify_on_score_alert: notificationSettingsData.notify_on_score_alert,
        score_alert_threshold: notificationSettingsData.score_alert_threshold,
        notify_on_agent_offline: notificationSettingsData.notify_on_agent_offline,
      });
    }
  }, [notificationSettingsData]);

  // Save notification settings mutation
  const saveNotifMutation = useMutation({
    mutationFn: async (data: NotificationSettingsRequest) => {
      if (hasExistingNotifSettings && notificationSettingsData?.id) {
        return notificationApi.updateSettings(notificationSettingsData.id, data);
      } else {
        return notificationApi.createSettings(data);
      }
    },
    onSuccess: () => {
      setHasExistingNotifSettings(true);
      queryClient.invalidateQueries({ queryKey: ['notificationSettings'] });
      toast.success('Notification settings saved');
    },
    onError: () => {
      toast.error('Failed to save notification settings');
    },
  });

  // Test SMTP mutation
  const testSMTPMutation = useMutation({
    mutationFn: (email: string) => notificationApi.testSMTP(email),
    onSuccess: () => {
      toast.success('Test email sent successfully');
    },
    onError: () => {
      toast.error('Failed to send test email');
    },
  });

  const handleSaveLocalSettings = () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
    toast.success('Settings saved successfully');
  };

  const handleSaveNotificationSettings = () => {
    saveNotifMutation.mutate(notifSettings);
  };

  const handleTestSMTP = () => {
    if (!testEmail) {
      toast.error('Please enter an email address');
      return;
    }
    testSMTPMutation.mutate(testEmail);
  };

  const updateSetting = <K extends keyof LocalSettings>(key: K, value: LocalSettings[K]) => {
    setSettings(prev => ({ ...prev, [key]: value }));
  };

  const updateNotifSetting = <K extends keyof NotificationSettingsRequest>(
    key: K,
    value: NotificationSettingsRequest[K]
  ) => {
    setNotifSettings(prev => ({ ...prev, [key]: value }));
  };

  return (
    <div>
      <h1 className="text-3xl font-bold mb-8">Settings</h1>

      <div className="max-w-2xl space-y-6">
        {/* Notification Settings */}
        <div className="card">
          <div className="flex items-center gap-2 mb-4">
            <BellIcon className="h-5 w-5 text-primary-600" />
            <h2 className="text-lg font-semibold">Notification Settings</h2>
          </div>

          {notifLoading ? (
            <div className="text-center py-4 text-gray-500">Loading...</div>
          ) : (
            <div className="space-y-4">
              {/* Enable Notifications */}
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium">Enable Notifications</p>
                  <p className="text-sm text-gray-500">
                    Receive notifications for execution events
                  </p>
                </div>
                <Toggle
                  enabled={notifSettings.enabled}
                  onChange={() => updateNotifSetting('enabled', !notifSettings.enabled)}
                />
              </div>

              {notifSettings.enabled && (
                <>
                  {/* Channel Selection */}
                  <div>
                    <label htmlFor="notif-channel" className="block text-sm font-medium text-gray-700 mb-1">
                      Notification Channel
                    </label>
                    <select
                      id="notif-channel"
                      className="input"
                      value={notifSettings.channel}
                      onChange={(e) =>
                        updateNotifSetting('channel', e.target.value as NotificationChannel)
                      }
                    >
                      <option value="email">Email</option>
                      <option value="webhook">Webhook</option>
                    </select>
                  </div>

                  {/* Email Settings */}
                  {notifSettings.channel === 'email' && (
                    <div>
                      <label htmlFor="notif-email" className="block text-sm font-medium text-gray-700 mb-1">
                        Email Address
                      </label>
                      <input
                        id="notif-email"
                        type="email"
                        className="input"
                        placeholder="your@email.com"
                        value={notifSettings.email_address || ''}
                        onChange={(e) => updateNotifSetting('email_address', e.target.value)}
                      />
                      {!smtpConfig && (
                        <p className="mt-1 text-sm text-amber-600 flex items-center gap-1">
                          <ExclamationTriangleIcon className="h-4 w-4" />
                          SMTP not configured on server. Email notifications may not work.
                        </p>
                      )}
                    </div>
                  )}

                  {/* Webhook Settings */}
                  {notifSettings.channel === 'webhook' && (
                    <div>
                      <label htmlFor="notif-webhook" className="block text-sm font-medium text-gray-700 mb-1">
                        Webhook URL
                      </label>
                      <input
                        id="notif-webhook"
                        type="url"
                        className="input"
                        placeholder="https://your-webhook-url.com/notify"
                        value={notifSettings.webhook_url || ''}
                        onChange={(e) => updateNotifSetting('webhook_url', e.target.value)}
                      />
                    </div>
                  )}

                  {/* Notification Types */}
                  <div className="border-t pt-4 mt-4">
                    <p className="font-medium mb-3">Notify me when:</p>
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-700">Execution starts</span>
                        <Toggle
                          enabled={notifSettings.notify_on_start}
                          onChange={() =>
                            updateNotifSetting('notify_on_start', !notifSettings.notify_on_start)
                          }
                        />
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-700">Execution completes</span>
                        <Toggle
                          enabled={notifSettings.notify_on_complete}
                          onChange={() =>
                            updateNotifSetting('notify_on_complete', !notifSettings.notify_on_complete)
                          }
                        />
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-700">Execution fails</span>
                        <Toggle
                          enabled={notifSettings.notify_on_failure}
                          onChange={() =>
                            updateNotifSetting('notify_on_failure', !notifSettings.notify_on_failure)
                          }
                        />
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-700">Agent goes offline</span>
                        <Toggle
                          enabled={notifSettings.notify_on_agent_offline}
                          onChange={() =>
                            updateNotifSetting(
                              'notify_on_agent_offline',
                              !notifSettings.notify_on_agent_offline
                            )
                          }
                        />
                      </div>
                      <div className="flex items-center justify-between">
                        <div className="flex-1">
                          <span className="text-sm text-gray-700">
                            Security score below threshold
                          </span>
                          {notifSettings.notify_on_score_alert && (
                            <div className="mt-1">
                              <input
                                type="number"
                                className="input w-24"
                                min="0"
                                max="100"
                                value={notifSettings.score_alert_threshold}
                                onChange={(e) =>
                                  updateNotifSetting(
                                    'score_alert_threshold',
                                    Number.parseInt(e.target.value, 10) || 0
                                  )
                                }
                              />
                              <span className="text-sm text-gray-500 ml-2">%</span>
                            </div>
                          )}
                        </div>
                        <Toggle
                          enabled={notifSettings.notify_on_score_alert}
                          onChange={() =>
                            updateNotifSetting(
                              'notify_on_score_alert',
                              !notifSettings.notify_on_score_alert
                            )
                          }
                        />
                      </div>
                    </div>
                  </div>

                  <div className="flex justify-end pt-4">
                    <button
                      className="btn-primary"
                      onClick={handleSaveNotificationSettings}
                      disabled={saveNotifMutation.isPending}
                    >
                      {saveNotifMutation.isPending ? 'Saving...' : 'Save Notification Settings'}
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </div>

        {/* SMTP Test (only shown if SMTP is configured) */}
        {smtpConfig && (
          <div className="card">
            <div className="flex items-center gap-2 mb-4">
              <EnvelopeIcon className="h-5 w-5 text-primary-600" />
              <h2 className="text-lg font-semibold">Test Email</h2>
            </div>
            <div className="space-y-4">
              <p className="text-sm text-gray-600">
                SMTP Server: {smtpConfig.host}:{smtpConfig.port} ({smtpConfig.use_tls ? 'TLS' : 'Plain'})
              </p>
              <div className="flex gap-2">
                <input
                  type="email"
                  className="input flex-1"
                  placeholder="Enter email to send test"
                  value={testEmail}
                  onChange={(e) => setTestEmail(e.target.value)}
                />
                <button
                  className="btn-secondary"
                  onClick={handleTestSMTP}
                  disabled={testSMTPMutation.isPending}
                >
                  {testSMTPMutation.isPending ? 'Sending...' : 'Send Test'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Server Configuration */}
        <div className="card">
          <h2 className="text-lg font-semibold mb-4">Server Configuration</h2>
          <div className="space-y-4">
            <div>
              <label htmlFor="server-url" className="block text-sm font-medium text-gray-700 mb-1">
                Server URL
              </label>
              <input
                id="server-url"
                type="text"
                className="input"
                value={settings.serverUrl}
                onChange={(e) => updateSetting('serverUrl', e.target.value)}
              />
            </div>
          </div>
        </div>

        {/* Execution Settings */}
        <div className="card">
          <h2 className="text-lg font-semibold mb-4">Execution Settings</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">Safe Mode by Default</p>
                <p className="text-sm text-gray-500">
                  Only run safe techniques that don't modify the system
                </p>
              </div>
              <Toggle
                enabled={settings.safeMode}
                onChange={() => updateSetting('safeMode', !settings.safeMode)}
              />
            </div>
          </div>
        </div>

        {/* Agent Settings */}
        <div className="card">
          <h2 className="text-lg font-semibold mb-4">Agent Settings</h2>
          <div className="space-y-4">
            <div>
              <label htmlFor="heartbeat-interval" className="block text-sm font-medium text-gray-700 mb-1">
                Heartbeat Interval (seconds)
              </label>
              <input
                id="heartbeat-interval"
                type="number"
                className="input"
                value={settings.heartbeatInterval}
                onChange={(e) => updateSetting('heartbeatInterval', Number.parseInt(e.target.value, 10) || DEFAULT_HEARTBEAT_INTERVAL)}
              />
            </div>
            <div>
              <label htmlFor="stale-timeout" className="block text-sm font-medium text-gray-700 mb-1">
                Stale Agent Timeout (seconds)
              </label>
              <input
                id="stale-timeout"
                type="number"
                className="input"
                value={settings.staleTimeout}
                onChange={(e) => updateSetting('staleTimeout', Number.parseInt(e.target.value, 10) || DEFAULT_STALE_TIMEOUT)}
              />
            </div>
          </div>
        </div>

        {/* TLS Settings */}
        <div className="card">
          <h2 className="text-lg font-semibold mb-4">TLS / mTLS Configuration</h2>
          <div className="space-y-4">
            <div>
              <label htmlFor="ca-cert-path" className="block text-sm font-medium text-gray-700 mb-1">
                CA Certificate Path
              </label>
              <input
                id="ca-cert-path"
                type="text"
                className="input"
                placeholder="/path/to/ca.crt"
                value={settings.caCertPath}
                onChange={(e) => updateSetting('caCertPath', e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="server-cert-path" className="block text-sm font-medium text-gray-700 mb-1">
                Server Certificate Path
              </label>
              <input
                id="server-cert-path"
                type="text"
                className="input"
                placeholder="/path/to/server.crt"
                value={settings.serverCertPath}
                onChange={(e) => updateSetting('serverCertPath', e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="server-key-path" className="block text-sm font-medium text-gray-700 mb-1">
                Server Key Path
              </label>
              <input
                id="server-key-path"
                type="text"
                className="input"
                placeholder="/path/to/server.key"
                value={settings.serverKeyPath}
                onChange={(e) => updateSetting('serverKeyPath', e.target.value)}
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end">
          <button className="btn-primary" onClick={handleSaveLocalSettings}>
            Save Settings
          </button>
        </div>
      </div>
    </div>
  );
}
