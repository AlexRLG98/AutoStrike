import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import {
  CalendarIcon,
  PlusIcon,
  PlayIcon,
  PauseIcon,
  TrashIcon,
  ClockIcon,
  XMarkIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  PencilIcon,
} from '@heroicons/react/24/outline';
import {
  scheduleApi,
  scenarioApi,
  Schedule,
  ScheduleRun,
  CreateScheduleRequest,
  ScheduleFrequency,
} from '../lib/api';
import { Scenario } from '../types';
import { LoadingState } from '../components/LoadingState';
import { EmptyState } from '../components/EmptyState';
import toast from 'react-hot-toast';

function getRunStatusColor(status: string): string {
  switch (status) {
    case 'completed':
      return 'bg-green-500';
    case 'failed':
      return 'bg-red-500';
    default:
      return 'bg-yellow-500';
  }
}

function getSubmitButtonText(isPending: boolean, isEditMode: boolean): string {
  if (isPending) {
    return isEditMode ? 'Updating...' : 'Creating...';
  }
  return isEditMode ? 'Update Schedule' : 'Create Schedule';
}

interface ScheduleStatusButtonProps {
  readonly status: string;
  readonly scheduleId: string;
  readonly onPause: (id: string) => void;
  readonly onResume: (id: string) => void;
  readonly isPauseDisabled: boolean;
  readonly isResumeDisabled: boolean;
}

function ScheduleStatusButton({
  status,
  scheduleId,
  onPause,
  onResume,
  isPauseDisabled,
  isResumeDisabled,
}: ScheduleStatusButtonProps) {
  if (status === 'active') {
    return (
      <button
        onClick={() => onPause(scheduleId)}
        className="p-2 text-yellow-600 hover:bg-yellow-50 rounded-lg"
        title="Pause"
        disabled={isPauseDisabled}
      >
        <PauseIcon className="h-5 w-5" />
      </button>
    );
  }
  if (status === 'paused') {
    return (
      <button
        onClick={() => onResume(scheduleId)}
        className="p-2 text-green-600 hover:bg-green-50 rounded-lg"
        title="Resume"
        disabled={isResumeDisabled}
      >
        <PlayIcon className="h-5 w-5" />
      </button>
    );
  }
  return null;
}

const frequencyLabels: Record<ScheduleFrequency, string> = {
  once: 'Once',
  hourly: 'Hourly',
  daily: 'Daily',
  weekly: 'Weekly',
  monthly: 'Monthly',
  cron: 'Custom (Cron)',
};

const statusColors: Record<string, string> = {
  active: 'bg-green-100 text-green-700',
  paused: 'bg-yellow-100 text-yellow-700',
  disabled: 'bg-gray-100 text-gray-700',
};

function formatDate(date: string | null): string {
  if (!date) return 'Never';
  return new Date(date).toLocaleString();
}

function formatRelativeTime(date: string | null): string {
  if (!date) return 'N/A';
  const d = new Date(date);
  const now = new Date();
  const diff = d.getTime() - now.getTime();

  if (diff < 0) return 'Overdue';

  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `in ${days}d ${hours % 24}h`;
  if (hours > 0) return `in ${hours}h ${minutes % 60}m`;
  return `in ${minutes}m`;
}

export default function Scheduler() {
  const queryClient = useQueryClient();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [scheduleToEdit, setScheduleToEdit] = useState<Schedule | null>(null);
  const [expandedSchedule, setExpandedSchedule] = useState<string | null>(null);
  const [scheduleToDelete, setScheduleToDelete] = useState<Schedule | null>(null);

  const { data: schedules, isLoading } = useQuery<Schedule[]>({
    queryKey: ['schedules'],
    queryFn: () => scheduleApi.list().then((res) => res.data),
  });

  const { data: scenarios } = useQuery<Scenario[]>({
    queryKey: ['scenarios'],
    queryFn: () => scenarioApi.list().then((res) => res.data),
  });

  const pauseMutation = useMutation({
    mutationFn: (id: string) => scheduleApi.pause(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
      toast.success('Schedule paused');
    },
    onError: () => toast.error('Failed to pause schedule'),
  });

  const resumeMutation = useMutation({
    mutationFn: (id: string) => scheduleApi.resume(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
      toast.success('Schedule resumed');
    },
    onError: () => toast.error('Failed to resume schedule'),
  });

  const runNowMutation = useMutation({
    mutationFn: (id: string) => scheduleApi.runNow(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
      queryClient.invalidateQueries({ queryKey: ['schedule-runs'] });
      toast.success('Schedule execution started');
    },
    onError: () => toast.error('Failed to run schedule'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => scheduleApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
      toast.success('Schedule deleted');
      setScheduleToDelete(null);
    },
    onError: () => toast.error('Failed to delete schedule'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateScheduleRequest }) =>
      scheduleApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
      toast.success('Schedule updated');
      setScheduleToEdit(null);
    },
    onError: () => toast.error('Failed to update schedule'),
  });

  const getScenarioName = (scenarioId: string): string => {
    const scenario = scenarios?.find((s) => s.id === scenarioId);
    return scenario?.name || 'Unknown Scenario';
  };

  const toggleExpanded = (id: string) => {
    setExpandedSchedule(expandedSchedule === id ? null : id);
  };

  if (isLoading) {
    return <LoadingState message="Loading schedules..." />;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold">Scheduler</h1>
        <button
          onClick={() => setShowCreateModal(true)}
          className="btn-primary flex items-center gap-2"
        >
          <PlusIcon className="h-5 w-5" />
          Create Schedule
        </button>
      </div>

      {schedules && schedules.length > 0 ? (
        <div className="space-y-4">
          {schedules.map((schedule) => (
            <div key={schedule.id} className="card">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <h3 className="font-semibold text-lg">{schedule.name}</h3>
                    <span
                      className={`px-2 py-1 rounded-full text-xs font-medium ${statusColors[schedule.status]}`}
                    >
                      {schedule.status}
                    </span>
                  </div>
                  {schedule.description && (
                    <p className="text-sm text-gray-500 mt-1">{schedule.description}</p>
                  )}

                  <div className="mt-3 grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div>
                      <span className="text-gray-500">Scenario:</span>
                      <p className="font-medium">{getScenarioName(schedule.scenario_id)}</p>
                    </div>
                    <div>
                      <span className="text-gray-500">Frequency:</span>
                      <p className="font-medium">
                        {frequencyLabels[schedule.frequency]}
                        {schedule.frequency === 'cron' && schedule.cron_expr && (
                          <span className="text-gray-400 ml-1">({schedule.cron_expr})</span>
                        )}
                      </p>
                    </div>
                    <div>
                      <span className="text-gray-500">Next Run:</span>
                      <p className="font-medium">
                        {schedule.status === 'active'
                          ? formatRelativeTime(schedule.next_run_at)
                          : 'Paused'}
                      </p>
                    </div>
                    <div>
                      <span className="text-gray-500">Last Run:</span>
                      <p className="font-medium">{formatDate(schedule.last_run_at)}</p>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-2 ml-4">
                  <ScheduleStatusButton
                    status={schedule.status}
                    scheduleId={schedule.id}
                    onPause={(id) => pauseMutation.mutate(id)}
                    onResume={(id) => resumeMutation.mutate(id)}
                    isPauseDisabled={pauseMutation.isPending}
                    isResumeDisabled={resumeMutation.isPending}
                  />
                  <button
                    onClick={() => runNowMutation.mutate(schedule.id)}
                    className="p-2 text-primary-600 hover:bg-primary-50 rounded-lg"
                    title="Run Now"
                    disabled={runNowMutation.isPending}
                  >
                    <ClockIcon className="h-5 w-5" />
                  </button>
                  <button
                    onClick={() => setScheduleToEdit(schedule)}
                    className="p-2 text-gray-600 hover:bg-gray-50 rounded-lg"
                    title="Edit"
                  >
                    <PencilIcon className="h-5 w-5" />
                  </button>
                  <button
                    onClick={() => setScheduleToDelete(schedule)}
                    className="p-2 text-red-600 hover:bg-red-50 rounded-lg"
                    title="Delete"
                  >
                    <TrashIcon className="h-5 w-5" />
                  </button>
                  <button
                    onClick={() => toggleExpanded(schedule.id)}
                    className="p-2 text-gray-500 hover:bg-gray-50 rounded-lg"
                    title="Show history"
                  >
                    {expandedSchedule === schedule.id ? (
                      <ChevronUpIcon className="h-5 w-5" />
                    ) : (
                      <ChevronDownIcon className="h-5 w-5" />
                    )}
                  </button>
                </div>
              </div>

              {expandedSchedule === schedule.id && (
                <ScheduleRunsHistory scheduleId={schedule.id} />
              )}
            </div>
          ))}
        </div>
      ) : (
        <EmptyState
          icon={CalendarIcon}
          title="No schedules created"
          description="Create a schedule to automatically run scenarios"
        />
      )}

      {/* Create Schedule Modal */}
      {showCreateModal && (
        <ScheduleFormModal
          scenarios={scenarios || []}
          onClose={() => setShowCreateModal(false)}
        />
      )}

      {/* Edit Schedule Modal */}
      {scheduleToEdit && (
        <ScheduleFormModal
          schedule={scheduleToEdit}
          scenarios={scenarios || []}
          onClose={() => setScheduleToEdit(null)}
          onUpdate={(data) => updateMutation.mutate({ id: scheduleToEdit.id, data })}
          isUpdating={updateMutation.isPending}
        />
      )}

      {/* Delete Confirmation Modal */}
      {scheduleToDelete && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
            <h2 className="text-xl font-semibold mb-4">Delete Schedule</h2>
            <p className="text-gray-600 mb-6">
              Are you sure you want to delete "{scheduleToDelete.name}"? This action cannot be undone.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setScheduleToDelete(null)}
                className="btn-secondary"
              >
                Cancel
              </button>
              <button
                onClick={() => deleteMutation.mutate(scheduleToDelete.id)}
                className="btn-primary bg-red-600 hover:bg-red-700"
                disabled={deleteMutation.isPending}
              >
                {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface ScheduleRunsHistoryProps {
  readonly scheduleId: string;
}

function ScheduleRunsHistory({ scheduleId }: ScheduleRunsHistoryProps) {
  const { data: runs, isLoading } = useQuery<ScheduleRun[]>({
    queryKey: ['schedule-runs', scheduleId],
    queryFn: () => scheduleApi.getRuns(scheduleId, 10).then((res) => res.data),
  });

  if (isLoading) {
    return (
      <div className="mt-4 pt-4 border-t border-gray-100">
        <p className="text-sm text-gray-500">Loading history...</p>
      </div>
    );
  }

  if (!runs || runs.length === 0) {
    return (
      <div className="mt-4 pt-4 border-t border-gray-100">
        <p className="text-sm text-gray-500">No runs yet</p>
      </div>
    );
  }

  return (
    <div className="mt-4 pt-4 border-t border-gray-100">
      <h4 className="text-sm font-medium text-gray-700 mb-3">Recent Runs</h4>
      <div className="space-y-2">
        {runs.map((run) => (
          <div
            key={run.id}
            className="flex items-center justify-between py-2 px-3 bg-gray-50 rounded-lg text-sm"
          >
            <div className="flex items-center gap-3">
              <span
                className={`w-2 h-2 rounded-full ${getRunStatusColor(run.status)}`}
              />
              <span>{formatDate(run.started_at)}</span>
              {run.error && (
                <span className="text-red-600 text-xs">({run.error})</span>
              )}
            </div>
            {run.execution_id && (
              <Link
                to={`/executions/${run.execution_id}`}
                className="text-primary-600 hover:text-primary-700"
              >
                View Execution
              </Link>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

interface ScheduleFormModalProps {
  readonly schedule?: Schedule;
  readonly scenarios: Scenario[];
  readonly onClose: () => void;
  readonly onUpdate?: (data: CreateScheduleRequest) => void;
  readonly isUpdating?: boolean;
}

function ScheduleFormModal({
  schedule,
  scenarios,
  onClose,
  onUpdate,
  isUpdating,
}: ScheduleFormModalProps) {
  const queryClient = useQueryClient();
  const isEditMode = !!schedule;

  const [formData, setFormData] = useState<CreateScheduleRequest>({
    name: schedule?.name || '',
    description: schedule?.description || '',
    scenario_id: schedule?.scenario_id || '',
    agent_paw: schedule?.agent_paw || '',
    frequency: schedule?.frequency || 'daily',
    cron_expr: schedule?.cron_expr || '',
    safe_mode: schedule?.safe_mode ?? true,
    start_at: '',
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateScheduleRequest) => scheduleApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
      toast.success('Schedule created');
      onClose();
    },
    onError: () => toast.error('Failed to create schedule'),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const data = { ...formData };
    if (!data.start_at) {
      delete data.start_at;
    }
    if (data.frequency !== 'cron') {
      delete data.cron_expr;
    }

    if (isEditMode && onUpdate) {
      onUpdate(data);
    } else {
      createMutation.mutate(data);
    }
  };

  const isPending = isEditMode ? isUpdating : createMutation.isPending;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-xl shadow-xl max-w-lg w-full mx-4">
        <div className="flex items-center justify-between p-6 border-b">
          <h2 className="text-xl font-semibold">
            {isEditMode ? 'Edit Schedule' : 'Create Schedule'}
          </h2>
          <button onClick={onClose} className="p-2 hover:bg-gray-100 rounded-lg">
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div>
            <label htmlFor="schedule-name" className="block text-sm font-medium text-gray-700 mb-1">
              Name *
            </label>
            <input
              id="schedule-name"
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className="input"
              required
            />
          </div>

          <div>
            <label htmlFor="schedule-description" className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              id="schedule-description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              className="input"
              rows={2}
            />
          </div>

          <div>
            <label htmlFor="schedule-scenario" className="block text-sm font-medium text-gray-700 mb-1">
              Scenario *
            </label>
            <select
              id="schedule-scenario"
              value={formData.scenario_id}
              onChange={(e) => setFormData({ ...formData, scenario_id: e.target.value })}
              className="input"
              required
            >
              <option value="">Select a scenario</option>
              {scenarios.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label htmlFor="schedule-frequency" className="block text-sm font-medium text-gray-700 mb-1">
              Frequency *
            </label>
            <select
              id="schedule-frequency"
              value={formData.frequency}
              onChange={(e) =>
                setFormData({ ...formData, frequency: e.target.value as ScheduleFrequency })
              }
              className="input"
            >
              {Object.entries(frequencyLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </div>

          {formData.frequency === 'cron' && (
            <div>
              <label htmlFor="schedule-cron" className="block text-sm font-medium text-gray-700 mb-1">
                Cron Expression *
              </label>
              <input
                id="schedule-cron"
                type="text"
                value={formData.cron_expr}
                onChange={(e) => setFormData({ ...formData, cron_expr: e.target.value })}
                className="input"
                placeholder="0 0 * * *"
                required={formData.frequency === 'cron'}
              />
              <p className="text-xs text-gray-500 mt-1">
                Format: minute hour day-of-month month day-of-week
              </p>
            </div>
          )}

          {!isEditMode && (
            <div>
              <label htmlFor="schedule-start-at" className="block text-sm font-medium text-gray-700 mb-1">
                Start At (optional)
              </label>
              <input
                id="schedule-start-at"
                type="datetime-local"
                value={formData.start_at}
                onChange={(e) => {
                  const value = e.target.value;
                  setFormData({
                    ...formData,
                    start_at: value ? new Date(value).toISOString() : '',
                  });
                }}
                className="input"
              />
            </div>
          )}

          <div>
            <label htmlFor="schedule-agent" className="block text-sm font-medium text-gray-700 mb-1">
              Agent (optional)
            </label>
            <input
              id="schedule-agent"
              type="text"
              value={formData.agent_paw}
              onChange={(e) => setFormData({ ...formData, agent_paw: e.target.value })}
              className="input"
              placeholder="Leave empty for all agents"
            />
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="safe_mode"
              checked={formData.safe_mode}
              onChange={(e) => setFormData({ ...formData, safe_mode: e.target.checked })}
              className="h-4 w-4 text-primary-600 rounded border-gray-300"
            />
            <label htmlFor="safe_mode" className="text-sm font-medium text-gray-700">
              Safe Mode
            </label>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={onClose} className="btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={isPending}>
              {getSubmitButtonText(isPending ?? false, isEditMode)}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
