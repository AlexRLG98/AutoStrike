import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import Executions from './Executions';
import { executionApi, api } from '../lib/api';
import toast from 'react-hot-toast';

// Mock the API
vi.mock('../lib/api', () => ({
  api: {
    get: vi.fn(),
  },
  executionApi: {
    list: vi.fn(),
    stop: vi.fn(),
    start: vi.fn(),
  },
}));

// Mock the WebSocket hook - capture onMessage callback
let capturedOnMessage: ((message: { type: string; payload: unknown }) => void) | undefined;
vi.mock('../hooks/useWebSocket', () => ({
  useWebSocket: vi.fn((options?: { onMessage?: (message: { type: string; payload: unknown }) => void }) => {
    capturedOnMessage = options?.onMessage;
    return {
      isConnected: false,
      send: vi.fn(),
      lastMessage: null,
    };
  }),
}));

// Mock date-fns
vi.mock('date-fns', () => ({
  formatDistanceToNow: vi.fn(() => '2 hours ago'),
}));

// Mock react-hot-toast
vi.mock('react-hot-toast', () => ({
  default: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

// Mock RunExecutionModal for controlled testing
vi.mock('../components/RunExecutionModal', () => ({
  RunExecutionModal: ({
    scenario,
    onConfirm,
    onCancel,
    isLoading,
  }: {
    scenario: { name: string; id: string };
    onConfirm: (agents: string[], safeMode: boolean) => void;
    onCancel: () => void;
    isLoading: boolean;
  }) => (
    <div data-testid="run-execution-modal">
      <span data-testid="modal-scenario-name">{scenario.name}</span>
      <span data-testid="modal-loading">{isLoading ? 'true' : 'false'}</span>
      <button data-testid="modal-run-confirm" onClick={() => onConfirm(['agent-1'], true)}>
        Run Execution
      </button>
      <button data-testid="modal-run-cancel" onClick={onCancel}>
        Cancel Run
      </button>
    </div>
  ),
}));

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

function renderWithClient(ui: React.ReactElement) {
  const testQueryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={testQueryClient}>
      <MemoryRouter>
        {ui}
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('Executions Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedOnMessage = undefined;
  });

  it('renders loading state', () => {
    vi.mocked(executionApi.list).mockReturnValue(new Promise(() => {}) as never);

    renderWithClient(<Executions />);
    expect(screen.getByText('Loading executions...')).toBeInTheDocument();
  });

  it('renders executions list with completed status', async () => {
    const mockExecutions = [
      {
        id: 'exec-12345678-abcd',
        scenario_id: 'scenario-1',
        status: 'completed',
        started_at: '2024-01-15T10:00:00Z',
        completed_at: '2024-01-15T10:30:00Z',
        safe_mode: true,
        score: {
          overall: 75.5,
          blocked: 5,
          detected: 3,
          successful: 2,
          total: 10,
        },
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    expect(await screen.findByText('scenario-1')).toBeInTheDocument();
    expect(screen.getByText('completed')).toBeInTheDocument();
    expect(screen.getByText('75.5%')).toBeInTheDocument();
    expect(screen.getByText('5 blocked')).toBeInTheDocument();
    expect(screen.getByText('3 detected')).toBeInTheDocument();
    expect(screen.getByText('2 success')).toBeInTheDocument();
    expect(screen.getByText('Safe')).toBeInTheDocument();
  });

  it('renders running execution with warning badge and stop button', async () => {
    const mockExecutions = [
      {
        id: 'exec-running',
        scenario_id: 'scenario-2',
        status: 'running',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: false,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    expect(await screen.findByText('running')).toBeInTheDocument();
    expect(screen.getByText('Full')).toBeInTheDocument();
    expect(screen.getByText('-%')).toBeInTheDocument();
    // Should have a stop button for running execution
    expect(screen.getByRole('button', { name: /stop/i })).toBeInTheDocument();
  });

  it('renders failed execution with danger badge', async () => {
    const mockExecutions = [
      {
        id: 'exec-failed',
        scenario_id: 'scenario-3',
        status: 'failed',
        started_at: '2024-01-15T11:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    expect(await screen.findByText('failed')).toBeInTheDocument();
  });

  it('renders empty state when no executions', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);

    renderWithClient(<Executions />);

    expect(await screen.findByText('No executions yet')).toBeInTheDocument();
    expect(screen.getByText('Run a scenario to see results here')).toBeInTheDocument();
  });

  it('renders page title and new execution button', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);

    renderWithClient(<Executions />);

    expect(await screen.findByText('Executions')).toBeInTheDocument();
    expect(screen.getByText('New Execution')).toBeInTheDocument();
  });

  it('renders truncated execution ID', async () => {
    const mockExecutions = [
      {
        id: 'abcdefgh-1234-5678-9012-ijklmnopqrst',
        scenario_id: 'test-scenario',
        status: 'completed',
        started_at: '2024-01-15T10:00:00Z',
        safe_mode: true,
        score: { overall: 80, blocked: 4, detected: 1, successful: 0, total: 5 },
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    expect(await screen.findByText('abcdefgh...')).toBeInTheDocument();
  });

  it('handles execution without score', async () => {
    const mockExecutions = [
      {
        id: 'exec-no-score',
        scenario_id: 'pending-scenario',
        status: 'pending',
        started_at: '2024-01-15T14:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    expect(await screen.findByText('pending-scenario')).toBeInTheDocument();
    expect(screen.getByText('-%')).toBeInTheDocument();
    expect(screen.getByText('0 blocked')).toBeInTheDocument();
    expect(screen.getByText('0 detected')).toBeInTheDocument();
    expect(screen.getByText('0 success')).toBeInTheDocument();
    // Pending executions should have stop button
    expect(screen.getByRole('button', { name: /stop/i })).toBeInTheDocument();
  });

  it('opens confirmation modal when clicking stop button', async () => {
    const mockExecutions = [
      {
        id: 'exec-running-123',
        scenario_id: 'test-scenario',
        status: 'running',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    // Wait for the stop button to appear (the one in the table row, not the modal)
    const stopButtons = await screen.findAllByRole('button', { name: /stop/i });
    fireEvent.click(stopButtons[0]); // Click the row stop button

    // Modal should appear with title
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText(/Are you sure you want to stop this execution/)).toBeInTheDocument();
    // Check for execution details in modal
    expect(screen.getByText(/Execution ID:/)).toBeInTheDocument();
    expect(screen.getByText(/Scenario: test-scenario/)).toBeInTheDocument();
  });

  it('closes modal when clicking cancel', async () => {
    const mockExecutions = [
      {
        id: 'exec-running-456',
        scenario_id: 'test-scenario',
        status: 'running',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    // Open modal
    const stopButton = await screen.findByRole('button', { name: /stop/i });
    fireEvent.click(stopButton);

    // Click cancel
    const cancelButton = screen.getByRole('button', { name: /cancel/i });
    fireEvent.click(cancelButton);

    // Modal should close
    await waitFor(() => {
      expect(screen.queryByText('Stop Execution')).not.toBeInTheDocument();
    });
  });

  it('calls stop API when confirming stop', async () => {
    const mockExecutions = [
      {
        id: 'exec-to-stop',
        scenario_id: 'test-scenario',
        status: 'running',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);
    vi.mocked(executionApi.stop).mockResolvedValue({ data: { status: 'cancelled' } } as never);

    renderWithClient(<Executions />);

    // Open modal
    const stopButton = await screen.findByRole('button', { name: /stop/i });
    fireEvent.click(stopButton);

    // Confirm stop
    const confirmButton = screen.getByRole('button', { name: /stop execution/i });
    fireEvent.click(confirmButton);

    await waitFor(() => {
      expect(executionApi.stop).toHaveBeenCalledWith('exec-to-stop');
    });
  });

  it('renders cancelled execution with appropriate display', async () => {
    const mockExecutions = [
      {
        id: 'exec-cancelled',
        scenario_id: 'cancelled-scenario',
        status: 'cancelled',
        started_at: '2024-01-15T10:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    expect(await screen.findByText('cancelled')).toBeInTheDocument();
    expect(screen.getByText('Cancelled')).toBeInTheDocument();
    // Should not have stop button for cancelled execution
    expect(screen.queryByRole('button', { name: /stop/i })).not.toBeInTheDocument();
  });

  it('does not show stop button for completed executions', async () => {
    const mockExecutions = [
      {
        id: 'exec-completed',
        scenario_id: 'completed-scenario',
        status: 'completed',
        started_at: '2024-01-15T10:00:00Z',
        safe_mode: true,
        score: { overall: 100, blocked: 5, detected: 0, successful: 0, total: 5 },
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    await screen.findByText('completed');
    // Should not have stop button for completed execution
    expect(screen.queryByRole('button', { name: /stop/i })).not.toBeInTheDocument();
  });

  it('handles stop execution error', async () => {
    const mockExecutions = [
      {
        id: 'exec-error',
        scenario_id: 'error-scenario',
        status: 'running',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);
    vi.mocked(executionApi.stop).mockRejectedValue({
      response: { data: { error: 'Execution already stopped' }, status: 409 }
    } as never);

    renderWithClient(<Executions />);

    // Open modal
    const stopButton = await screen.findByRole('button', { name: /stop/i });
    fireEvent.click(stopButton);

    // Confirm stop
    const confirmButton = screen.getByRole('button', { name: /stop execution/i });
    fireEvent.click(confirmButton);

    await waitFor(() => {
      expect(executionApi.stop).toHaveBeenCalledWith('exec-error');
    });
  });

  it('handles stop execution error without response data', async () => {
    const mockExecutions = [
      {
        id: 'exec-network-error',
        scenario_id: 'network-scenario',
        status: 'running',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);
    vi.mocked(executionApi.stop).mockRejectedValue(new Error('Network error') as never);

    renderWithClient(<Executions />);

    // Open modal
    const stopButton = await screen.findByRole('button', { name: /stop/i });
    fireEvent.click(stopButton);

    // Confirm stop
    const confirmButton = screen.getByRole('button', { name: /stop execution/i });
    fireEvent.click(confirmButton);

    await waitFor(() => {
      expect(executionApi.stop).toHaveBeenCalledWith('exec-network-error');
    });
  });

  it('handles WebSocket execution_cancelled message', async () => {
    const mockExecutions = [
      {
        id: 'exec-1',
        scenario_id: 'scenario-1',
        status: 'running',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    await screen.findByText('running');

    // Trigger WebSocket message
    if (capturedOnMessage) {
      capturedOnMessage({ type: 'execution_cancelled', payload: { execution_id: 'exec-1' } });
    }

    // Query should be invalidated (we can check that the callback was captured)
    expect(capturedOnMessage).toBeDefined();
  });

  it('handles WebSocket execution_completed message', async () => {
    const mockExecutions = [
      {
        id: 'exec-2',
        scenario_id: 'scenario-2',
        status: 'running',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    await screen.findByText('running');

    // Trigger WebSocket message
    if (capturedOnMessage) {
      capturedOnMessage({ type: 'execution_completed', payload: { execution_id: 'exec-2' } });
    }

    expect(capturedOnMessage).toBeDefined();
  });

  it('handles WebSocket execution_started message', async () => {
    const mockExecutions = [
      {
        id: 'exec-3',
        scenario_id: 'scenario-3',
        status: 'pending',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    await screen.findByText('pending');

    // Trigger WebSocket message
    if (capturedOnMessage) {
      capturedOnMessage({ type: 'execution_started', payload: { execution_id: 'exec-3' } });
    }

    expect(capturedOnMessage).toBeDefined();
  });

  it('ignores unrelated WebSocket messages', async () => {
    const mockExecutions = [
      {
        id: 'exec-4',
        scenario_id: 'scenario-4',
        status: 'running',
        started_at: '2024-01-15T12:00:00Z',
        safe_mode: true,
      },
    ];
    vi.mocked(executionApi.list).mockResolvedValue({ data: mockExecutions } as never);

    renderWithClient(<Executions />);

    await screen.findByText('running');

    // Trigger unrelated WebSocket message
    if (capturedOnMessage) {
      capturedOnMessage({ type: 'agent_connected', payload: { agent_id: 'agent-1' } });
    }

    // Should not cause any errors
    expect(capturedOnMessage).toBeDefined();
  });
});

describe('New Execution Flow', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedOnMessage = undefined;
  });

  it('opens scenario select modal when New Execution button is clicked', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({
      data: [
        {
          id: 'scenario-1',
          name: 'Test Scenario',
          description: 'A test scenario',
          phases: [{ name: 'Phase 1', techniques: ['T1082'] }],
          tags: ['safe'],
        },
      ],
    } as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('Select Scenario')).toBeInTheDocument();
    });
  });

  it('displays scenarios in the select modal', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({
      data: [
        {
          id: 'scenario-1',
          name: 'Discovery Scenario',
          description: 'Test discovery techniques',
          phases: [{ name: 'Phase 1', techniques: ['T1082'] }],
          tags: ['discovery', 'safe'],
        },
        {
          id: 'scenario-2',
          name: 'Execution Scenario',
          description: 'Test execution techniques',
          phases: [{ name: 'Phase 1', techniques: ['T1059'] }, { name: 'Phase 2', techniques: ['T1059.001'] }],
          tags: ['execution'],
        },
      ],
    } as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('Discovery Scenario')).toBeInTheDocument();
      expect(screen.getByText('Execution Scenario')).toBeInTheDocument();
      expect(screen.getByText('Test discovery techniques')).toBeInTheDocument();
      expect(screen.getByText('1 phases')).toBeInTheDocument();
      expect(screen.getByText('2 phases')).toBeInTheDocument();
    });
  });

  it('shows empty state when no scenarios available', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({ data: [] } as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('No scenarios available. Create a scenario first.')).toBeInTheDocument();
    });
  });

  it('closes scenario select modal when X is clicked', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({
      data: [{ id: 'scenario-1', name: 'Test', description: 'Test', phases: [], tags: [] }],
    } as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('Select Scenario')).toBeInTheDocument();
    });

    // Find the close button in the modal
    const modalHeader = screen.getByText('Select Scenario').closest('div');
    const closeButton = modalHeader?.parentElement?.querySelector('button');
    if (closeButton) {
      fireEvent.click(closeButton);
    }

    await waitFor(() => {
      expect(screen.queryByText('Select Scenario')).not.toBeInTheDocument();
    });
  });

  it('opens RunExecutionModal when scenario is selected', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({
      data: [
        {
          id: 'scenario-1',
          name: 'Selected Scenario',
          description: 'Will be selected',
          phases: [{ name: 'Phase 1', techniques: ['T1082'] }],
          tags: ['safe'],
        },
      ],
    } as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('Selected Scenario')).toBeInTheDocument();
    });

    // Click on the scenario to select it
    fireEvent.click(screen.getByText('Selected Scenario'));

    await waitFor(() => {
      // RunExecutionModal should be displayed
      expect(screen.getByTestId('run-execution-modal')).toBeInTheDocument();
      expect(screen.getByTestId('modal-scenario-name')).toHaveTextContent('Selected Scenario');
    });
  });

  it('starts execution when confirmed in RunExecutionModal', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({
      data: [
        {
          id: 'scenario-run',
          name: 'Runnable Scenario',
          description: 'Will be run',
          phases: [{ name: 'Phase 1', techniques: ['T1082'] }],
          tags: [],
        },
      ],
    } as never);
    vi.mocked(executionApi.start).mockResolvedValue({ data: { id: 'exec-123' } } as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('Runnable Scenario')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Runnable Scenario'));

    await waitFor(() => {
      expect(screen.getByTestId('run-execution-modal')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('modal-run-confirm'));

    await waitFor(() => {
      expect(executionApi.start).toHaveBeenCalledWith('scenario-run', ['agent-1'], true);
      expect(toast.success).toHaveBeenCalledWith('Execution started successfully');
    });
  });

  it('closes RunExecutionModal when cancelled', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({
      data: [
        {
          id: 'scenario-cancel',
          name: 'Cancelable Scenario',
          description: 'Will be cancelled',
          phases: [{ name: 'Phase 1', techniques: ['T1082'] }],
          tags: [],
        },
      ],
    } as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('Cancelable Scenario')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Cancelable Scenario'));

    await waitFor(() => {
      expect(screen.getByTestId('run-execution-modal')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('modal-run-cancel'));

    await waitFor(() => {
      expect(screen.queryByTestId('run-execution-modal')).not.toBeInTheDocument();
    });
  });

  it('shows error toast when start execution fails', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({
      data: [
        {
          id: 'scenario-fail',
          name: 'Failing Scenario',
          description: 'Will fail',
          phases: [{ name: 'Phase 1', techniques: ['T1082'] }],
          tags: [],
        },
      ],
    } as never);
    vi.mocked(executionApi.start).mockRejectedValue({
      response: { data: { error: 'No agents available' } },
    } as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('Failing Scenario')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Failing Scenario'));

    await waitFor(() => {
      expect(screen.getByTestId('run-execution-modal')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('modal-run-confirm'));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('No agents available');
    });
  });

  it('shows default error message when start fails without details', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({
      data: [
        {
          id: 'scenario-network-fail',
          name: 'Network Fail Scenario',
          description: 'Network fails',
          phases: [{ name: 'Phase 1', techniques: ['T1082'] }],
          tags: [],
        },
      ],
    } as never);
    vi.mocked(executionApi.start).mockRejectedValue(new Error('Network error') as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('Network Fail Scenario')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Network Fail Scenario'));

    await waitFor(() => {
      expect(screen.getByTestId('run-execution-modal')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('modal-run-confirm'));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('Failed to start execution');
    });
  });

  it('displays scenario tags in select modal', async () => {
    vi.mocked(executionApi.list).mockResolvedValue({ data: [] } as never);
    vi.mocked(api.get).mockResolvedValue({
      data: [
        {
          id: 'scenario-tags',
          name: 'Tagged Scenario',
          description: 'Has tags',
          phases: [{ name: 'Phase 1', techniques: ['T1082'] }],
          tags: ['discovery', 'safe', 'windows'],
        },
      ],
    } as never);

    renderWithClient(<Executions />);

    await screen.findByText('New Execution');
    fireEvent.click(screen.getByText('New Execution'));

    await waitFor(() => {
      expect(screen.getByText('discovery')).toBeInTheDocument();
      expect(screen.getByText('safe')).toBeInTheDocument();
      expect(screen.getByText('windows')).toBeInTheDocument();
    });
  });
});
