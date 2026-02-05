import { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { PlayIcon, StopIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { executionApi, api } from '../lib/api';
import { formatDistanceToNow } from 'date-fns';
import { Execution, ExecutionStatus, Scenario } from '../types';
import { LoadingState } from '../components/LoadingState';
import { EmptyState } from '../components/EmptyState';
import { Modal } from '../components/Modal';
import { RunExecutionModal } from '../components/RunExecutionModal';
import { useWebSocket, WebSocketMessage } from '../hooks/useWebSocket';
import toast from 'react-hot-toast';

/**
 * Returns the appropriate badge class for an execution status.
 */
function getStatusBadgeClass(status: string): string {
  switch (status) {
    case 'completed':
      return 'badge-success';
    case 'running':
      return 'badge-warning';
    case 'pending':
      return 'badge-warning';
    case 'cancelled':
      return 'badge-danger';
    default:
      return 'badge-danger';
  }
}

/**
 * Check if an execution can be stopped
 */
function canStopExecution(status: ExecutionStatus): boolean {
  return status === 'running' || status === 'pending';
}

/**
 * Confirmation Modal component for stopping executions
 */
interface StopConfirmModalProps {
  readonly execution: Execution;
  readonly onConfirm: () => void;
  readonly onCancel: () => void;
  readonly isLoading: boolean;
}

function StopConfirmModal({ execution, onConfirm, onCancel, isLoading }: Readonly<StopConfirmModalProps>) {
  return (
    <dialog open className="fixed inset-0 z-50 overflow-y-auto bg-transparent" aria-labelledby="modal-title" aria-modal="true">
      <div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
        {/* Background overlay */}
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" aria-hidden="true" onClick={onCancel}></div>

        {/* Modal panel */}
        <div className="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
          <div className="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
            <div className="sm:flex sm:items-start">
              <div className="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-red-100 sm:mx-0 sm:h-10 sm:w-10">
                <StopIcon className="h-6 w-6 text-red-600" aria-hidden="true" />
              </div>
              <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
                <h3 className="text-lg leading-6 font-medium text-gray-900" id="modal-title">
                  Stop Execution
                </h3>
                <div className="mt-2">
                  <p className="text-sm text-gray-500">
                    Are you sure you want to stop this execution? This will cancel all pending tasks and mark the execution as cancelled.
                  </p>
                  <div className="mt-3 p-3 bg-gray-50 rounded-md">
                    <p className="text-sm font-medium text-gray-900">Execution ID: <span className="font-mono">{execution.id.slice(0, 8)}...</span></p>
                    <p className="text-sm text-gray-500">Scenario: {execution.scenario_id}</p>
                    <p className="text-sm text-gray-500">Status: {execution.status}</p>
                  </div>
                  <p className="mt-2 text-sm text-amber-600">
                    Note: Partial results will be preserved and remain accessible.
                  </p>
                </div>
              </div>
            </div>
          </div>
          <div className="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
            <button
              type="button"
              disabled={isLoading}
              onClick={onConfirm}
              className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-red-600 text-base font-medium text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50"
            >
              {isLoading ? 'Stopping...' : 'Stop Execution'}
            </button>
            <button
              type="button"
              disabled={isLoading}
              onClick={onCancel}
              className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </dialog>
  );
}

/**
 * Scenario Selection Modal for new executions
 */
interface ScenarioSelectModalProps {
  readonly scenarios: Scenario[] | undefined;
  readonly isLoading: boolean;
  readonly onSelect: (scenario: Scenario) => void;
  readonly onCancel: () => void;
}

/**
 * Renders the content of the scenario selection modal based on loading state and scenarios.
 */
function ScenarioSelectContent({
  scenarios,
  isLoading,
  onSelect,
}: {
  readonly scenarios: Scenario[] | undefined;
  readonly isLoading: boolean;
  readonly onSelect: (scenario: Scenario) => void;
}) {
  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (!scenarios || scenarios.length === 0) {
    return (
      <p className="text-gray-500 text-center py-8">No scenarios available. Create a scenario first.</p>
    );
  }

  return (
    <div className="space-y-3">
      {scenarios.map((scenario) => (
        <button
          key={scenario.id}
          onClick={() => onSelect(scenario)}
          className="w-full text-left p-4 border rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
        >
          <div className="flex justify-between items-start">
            <div>
              <h3 className="font-semibold">{scenario.name}</h3>
              <p className="text-sm text-gray-500 mt-1">{scenario.description}</p>
            </div>
            <span className="text-xs text-gray-400">
              {scenario.phases.length} phases
            </span>
          </div>
          <div className="mt-2 flex gap-1 flex-wrap">
            {scenario.tags?.map((tag) => (
              <span key={tag} className="badge bg-gray-100 text-gray-700 text-xs">
                {tag}
              </span>
            ))}
          </div>
        </button>
      ))}
    </div>
  );
}

function ScenarioSelectModal({ scenarios, isLoading, onSelect, onCancel }: Readonly<ScenarioSelectModalProps>) {
  return (
    <Modal title="Select Scenario" onClose={onCancel} maxWidth="max-w-2xl">
      <ScenarioSelectContent scenarios={scenarios} isLoading={isLoading} onSelect={onSelect} />
    </Modal>
  );
}

/**
 * Executions page component.
 * Displays a table of scenario executions with their results and scores.
 *
 * @returns The Executions page component
 */
export default function Executions() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [executionToStop, setExecutionToStop] = useState<Execution | null>(null);
  const [showScenarioSelect, setShowScenarioSelect] = useState(false);
  const [scenarioToRun, setScenarioToRun] = useState<Scenario | null>(null);

  // Handle WebSocket messages for real-time updates
  const handleWebSocketMessage = useCallback((message: WebSocketMessage) => {
    if (message.type === 'execution_cancelled' ||
        message.type === 'execution_completed' ||
        message.type === 'execution_started') {
      // Invalidate the executions query to refresh the list
      queryClient.invalidateQueries({ queryKey: ['executions'] });
    }
  }, [queryClient]);

  // WebSocket connection for real-time updates
  useWebSocket({
    onMessage: handleWebSocketMessage,
  });

  const { data: executions, isLoading } = useQuery<Execution[]>({
    queryKey: ['executions'],
    queryFn: () => executionApi.list().then(res => res.data),
    refetchInterval: 5000, // Fallback polling every 5 seconds
  });

  const { data: scenarios, isLoading: isScenariosLoading } = useQuery<Scenario[]>({
    queryKey: ['scenarios'],
    queryFn: () => api.get('/scenarios').then(res => res.data),
  });

  // Mutation for stopping an execution
  const stopMutation = useMutation({
    mutationFn: (executionId: string) => executionApi.stop(executionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['executions'] });
      setExecutionToStop(null);
      toast.success('Execution cancelled successfully');
    },
    onError: (error: { response?: { data?: { error?: string }; status?: number } }) => {
      const message = error.response?.data?.error || 'Failed to stop execution';
      toast.error(message);
      setExecutionToStop(null);
    },
  });

  // Mutation for starting an execution
  const startMutation = useMutation({
    mutationFn: ({ scenarioId, agentPaws, safeMode }: { scenarioId: string; agentPaws: string[]; safeMode: boolean }) =>
      executionApi.start(scenarioId, agentPaws, safeMode),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['executions'] });
      toast.success('Execution started successfully');
      setScenarioToRun(null);
    },
    onError: (error: { response?: { data?: { error?: string } } }) => {
      const message = error.response?.data?.error || 'Failed to start execution';
      toast.error(message);
    },
  });

  const handleStopClick = (execution: Execution, e: React.MouseEvent) => {
    e.stopPropagation();
    setExecutionToStop(execution);
  };

  const handleConfirmStop = () => {
    if (executionToStop) {
      stopMutation.mutate(executionToStop.id);
    }
  };

  const handleCancelStop = () => {
    setExecutionToStop(null);
  };

  const handleNewExecution = () => {
    setShowScenarioSelect(true);
  };

  const handleScenarioSelect = (scenario: Scenario) => {
    setShowScenarioSelect(false);
    setScenarioToRun(scenario);
  };

  const handleConfirmRun = (agentPaws: string[], safeMode: boolean) => {
    if (scenarioToRun) {
      startMutation.mutate({
        scenarioId: scenarioToRun.id,
        agentPaws,
        safeMode,
      });
    }
  };

  const handleCancelRun = () => {
    setScenarioToRun(null);
  };

  if (isLoading) {
    return <LoadingState message="Loading executions..." />;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold">Executions</h1>
        <button onClick={handleNewExecution} className="btn-primary flex items-center gap-2">
          <PlayIcon className="h-5 w-5" />
          New Execution
        </button>
      </div>

      <div className="card overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Scenario
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Score
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Results
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Started
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Mode
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {executions?.map((execution) => (
              <tr
                key={execution.id}
                className="hover:bg-gray-50 cursor-pointer"
                onClick={() => navigate(`/executions/${execution.id}`)}
              >
                <td className="px-6 py-4 whitespace-nowrap">
                  <p className="font-medium">{execution.scenario_id}</p>
                  <p className="text-xs text-gray-400">{execution.id.slice(0, 8)}...</p>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className={`badge ${getStatusBadgeClass(execution.status)}`}>
                    {execution.status}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className="text-2xl font-bold">
                    {execution.score?.overall?.toFixed(1) || '-'}%
                  </span>
                </td>
                <td className="px-6 py-4">
                  <div className="flex gap-2 text-sm">
                    <span className="text-success-600">
                      {execution.score?.blocked || 0} blocked
                    </span>
                    <span className="text-warning-600">
                      {execution.score?.detected || 0} detected
                    </span>
                    <span className="text-danger-600">
                      {execution.score?.successful || 0} success
                    </span>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {formatDistanceToNow(new Date(execution.started_at), { addSuffix: true })}
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className={`badge ${execution.safe_mode ? 'badge-success' : 'badge-danger'}`}>
                    {execution.safe_mode ? 'Safe' : 'Full'}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  {canStopExecution(execution.status) && (
                    <button
                      onClick={(e) => handleStopClick(execution, e)}
                      className="inline-flex items-center px-3 py-1.5 border border-transparent text-xs font-medium rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
                      title="Stop execution"
                    >
                      <StopIcon className="h-4 w-4 mr-1" />
                      Stop
                    </button>
                  )}
                  {execution.status === 'cancelled' && (
                    <span className="inline-flex items-center text-xs text-gray-500">
                      <XMarkIcon className="h-4 w-4 mr-1" />
                      Cancelled
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {executions?.length === 0 && (
        <EmptyState
          icon={PlayIcon}
          title="No executions yet"
          description="Run a scenario to see results here"
        />
      )}

      {/* Stop Confirmation Modal */}
      {executionToStop && (
        <StopConfirmModal
          execution={executionToStop}
          onConfirm={handleConfirmStop}
          onCancel={handleCancelStop}
          isLoading={stopMutation.isPending}
        />
      )}

      {/* Scenario Selection Modal */}
      {showScenarioSelect && (
        <ScenarioSelectModal
          scenarios={scenarios}
          isLoading={isScenariosLoading}
          onSelect={handleScenarioSelect}
          onCancel={() => setShowScenarioSelect(false)}
        />
      )}

      {/* Run Execution Modal */}
      {scenarioToRun && (
        <RunExecutionModal
          scenario={scenarioToRun}
          onConfirm={handleConfirmRun}
          onCancel={handleCancelRun}
          isLoading={startMutation.isPending}
        />
      )}
    </div>
  );
}
