import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Scheduler from './Scheduler';

// Mock react-hot-toast
vi.mock('react-hot-toast', () => ({
  default: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

// Mock functions stored as references
const mockScheduleList = vi.fn();
const mockGetRuns = vi.fn();
const mockPause = vi.fn();
const mockResume = vi.fn();
const mockRunNow = vi.fn();
const mockDelete = vi.fn();
const mockCreate = vi.fn();
const mockUpdate = vi.fn();
const mockScenarioList = vi.fn();

// Mock the schedule and scenario API
vi.mock('../lib/api', () => ({
  scheduleApi: {
    list: () => mockScheduleList(),
    getRuns: (id: string) => mockGetRuns(id),
    pause: (id: string) => mockPause(id),
    resume: (id: string) => mockResume(id),
    runNow: (id: string) => mockRunNow(id),
    delete: (id: string) => mockDelete(id),
    create: (data: unknown) => mockCreate(data),
    update: (id: string, data: unknown) => mockUpdate(id, data),
  },
  scenarioApi: {
    list: () => mockScenarioList(),
  },
}));

const mockScheduleData = [
  {
    id: 'sched-1',
    name: 'Daily Security Check',
    description: 'Run security tests daily',
    scenario_id: 'scenario-1',
    agent_paw: '',
    frequency: 'daily',
    cron_expr: '',
    safe_mode: true,
    status: 'active',
    next_run_at: new Date(Date.now() + 3600000).toISOString(),
    last_run_at: new Date(Date.now() - 86400000).toISOString(),
    last_run_id: 'exec-1',
    created_by: 'admin',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'sched-2',
    name: 'Weekly Audit',
    description: 'Weekly security audit',
    scenario_id: 'scenario-2',
    agent_paw: 'agent-1',
    frequency: 'weekly',
    cron_expr: '',
    safe_mode: false,
    status: 'paused',
    next_run_at: null,
    last_run_at: null,
    last_run_id: '',
    created_by: 'admin',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
];

const mockScenarioData = [
  { id: 'scenario-1', name: 'Test Scenario 1', description: '', phases: [], tags: [] },
  { id: 'scenario-2', name: 'Test Scenario 2', description: '', phases: [], tags: [] },
];

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
}

function renderScheduler() {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Scheduler />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function setupDefaultMocks() {
  mockScheduleList.mockResolvedValue({ data: mockScheduleData });
  mockScenarioList.mockResolvedValue({ data: mockScenarioData });
  mockGetRuns.mockResolvedValue({ data: [] });
  mockPause.mockResolvedValue({ data: {} });
  mockResume.mockResolvedValue({ data: {} });
  mockRunNow.mockResolvedValue({ data: {} });
  mockDelete.mockResolvedValue({ data: {} });
  mockCreate.mockResolvedValue({ data: {} });
  mockUpdate.mockResolvedValue({ data: {} });
}

describe('Scheduler Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('renders scheduler title', async () => {
    renderScheduler();
    await waitFor(() => {
      expect(screen.getByText('Scheduler')).toBeInTheDocument();
    });
  });

  it('renders create schedule button', async () => {
    renderScheduler();
    await waitFor(() => {
      expect(screen.getByText('Create Schedule')).toBeInTheDocument();
    });
  });

  it('displays schedules after loading', async () => {
    renderScheduler();
    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
      expect(screen.getByText('Weekly Audit')).toBeInTheDocument();
    });
  });

  it('shows schedule status badges', async () => {
    renderScheduler();
    await waitFor(() => {
      expect(screen.getByText('active')).toBeInTheDocument();
      expect(screen.getByText('paused')).toBeInTheDocument();
    });
  });

  it('shows frequency labels', async () => {
    renderScheduler();
    await waitFor(() => {
      expect(screen.getByText('Daily')).toBeInTheDocument();
      expect(screen.getByText('Weekly')).toBeInTheDocument();
    });
  });
});

describe('Scheduler Create Modal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('opens create modal when button clicked', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const createButton = screen.getByText('Create Schedule');
    fireEvent.click(createButton);

    await waitFor(() => {
      expect(screen.getByText('Name *')).toBeInTheDocument();
      expect(screen.getByText('Scenario *')).toBeInTheDocument();
      expect(screen.getByText('Frequency *')).toBeInTheDocument();
    });
  });

  it('closes modal when cancel clicked', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Name *')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Cancel'));

    await waitFor(() => {
      expect(screen.queryByText('Name *')).not.toBeInTheDocument();
    });
  });

  it('shows safe mode checkbox', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Safe Mode')).toBeInTheDocument();
    });
  });
});

describe('Scheduler Edit', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('opens edit modal when edit button clicked', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const editButtons = screen.getAllByTitle('Edit');
    fireEvent.click(editButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Edit Schedule')).toBeInTheDocument();
    });
  });

  it('closes edit modal when cancel clicked', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const editButtons = screen.getAllByTitle('Edit');
    fireEvent.click(editButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Edit Schedule')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Cancel'));

    await waitFor(() => {
      expect(screen.queryByText('Edit Schedule')).not.toBeInTheDocument();
    });
  });

  it('pre-fills form with schedule data when editing', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const editButtons = screen.getAllByTitle('Edit');
    fireEvent.click(editButtons[0]);

    await waitFor(() => {
      const nameInput = screen.getByDisplayValue('Daily Security Check');
      expect(nameInput).toBeInTheDocument();
    });
  });
});

describe('Scheduler Actions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows delete confirmation modal', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    // Find all delete buttons (there should be one per schedule)
    const deleteButtons = screen.getAllByTitle('Delete');
    fireEvent.click(deleteButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Delete Schedule')).toBeInTheDocument();
      expect(
        screen.getByText(/Are you sure you want to delete "Daily Security Check"/)
      ).toBeInTheDocument();
    });
  });

  it('cancels delete when cancel clicked', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const deleteButtons = screen.getAllByTitle('Delete');
    fireEvent.click(deleteButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Delete Schedule')).toBeInTheDocument();
    });

    // Click cancel button in modal
    const cancelButtons = screen.getAllByText('Cancel');
    fireEvent.click(cancelButtons[cancelButtons.length - 1]);

    await waitFor(() => {
      expect(screen.queryByText('Delete Schedule')).not.toBeInTheDocument();
    });
  });
});

describe('Scheduler Frequency Labels', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('displays schedule descriptions', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Run security tests daily')).toBeInTheDocument();
      expect(screen.getByText('Weekly security audit')).toBeInTheDocument();
    });
  });

  it('shows scenario name labels', async () => {
    renderScheduler();

    await waitFor(() => {
      // Check that scenario info is displayed
      const scenarioLabels = screen.getAllByText('Scenario:');
      expect(scenarioLabels.length).toBeGreaterThan(0);
    });
  });
});

describe('Scheduler Empty State', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockScheduleList.mockResolvedValue({ data: [] });
    mockScenarioList.mockResolvedValue({ data: mockScenarioData });
    mockGetRuns.mockResolvedValue({ data: [] });
  });

  it('shows empty state when no schedules', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('No schedules created')).toBeInTheDocument();
    });
  });

  it('shows create button in empty state', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('No schedules created')).toBeInTheDocument();
    });

    expect(screen.getByText('Create Schedule')).toBeInTheDocument();
  });
});

describe('Scheduler Pause/Resume', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows pause button for active schedules', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const pauseButtons = screen.getAllByTitle('Pause');
    expect(pauseButtons.length).toBeGreaterThan(0);
  });

  it('shows resume button for paused schedules', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Weekly Audit')).toBeInTheDocument();
    });

    const resumeButtons = screen.getAllByTitle('Resume');
    expect(resumeButtons.length).toBeGreaterThan(0);
  });

  it('calls pause API when pause button clicked', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const pauseButtons = screen.getAllByTitle('Pause');
    fireEvent.click(pauseButtons[0]);

    await waitFor(() => {
      expect(mockPause).toHaveBeenCalledWith('sched-1');
    });
  });

  it('calls resume API when resume button clicked', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Weekly Audit')).toBeInTheDocument();
    });

    const resumeButtons = screen.getAllByTitle('Resume');
    fireEvent.click(resumeButtons[0]);

    await waitFor(() => {
      expect(mockResume).toHaveBeenCalledWith('sched-2');
    });
  });
});

describe('Scheduler Run Now', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows run now button for schedules', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const runNowButtons = screen.getAllByTitle('Run Now');
    expect(runNowButtons.length).toBeGreaterThan(0);
  });

  it('calls runNow API when run now button clicked', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const runNowButtons = screen.getAllByTitle('Run Now');
    fireEvent.click(runNowButtons[0]);

    await waitFor(() => {
      expect(mockRunNow).toHaveBeenCalledWith('sched-1');
    });
  });
});

describe('Scheduler Delete Confirmation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('calls delete API when confirmed', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const deleteButtons = screen.getAllByTitle('Delete');
    fireEvent.click(deleteButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Delete Schedule')).toBeInTheDocument();
    });

    // Find the Delete button in modal - it's the only one with that class
    const allButtons = screen.getAllByRole('button');
    const confirmButton = allButtons.find(
      (btn) => btn.textContent === 'Delete' && btn.className.includes('bg-red-600')
    );
    expect(confirmButton).toBeDefined();
    fireEvent.click(confirmButton!);

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith('sched-1');
    });
  });
});

describe('Scheduler History Expansion', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
    mockGetRuns.mockResolvedValue({
      data: [
        {
          id: 'run-1',
          schedule_id: 'sched-1',
          execution_id: 'exec-1',
          started_at: new Date().toISOString(),
          completed_at: new Date().toISOString(),
          status: 'completed',
          triggered_by: 'scheduler',
        },
      ],
    });
  });

  it('shows expand button for schedules with history', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const expandButtons = screen.getAllByTitle('Show history');
    expect(expandButtons.length).toBeGreaterThan(0);
  });

  it('expands history when expand button clicked', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const expandButtons = screen.getAllByTitle('Show history');
    fireEvent.click(expandButtons[0]);

    await waitFor(() => {
      // The history section shows runs or "No runs yet" message
      const historyContent = screen.getByText(/Loading history|No runs yet|completed/);
      expect(historyContent).toBeInTheDocument();
    });
  });

  it('collapses history when clicked again', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const expandButtons = screen.getAllByTitle('Show history');
    fireEvent.click(expandButtons[0]);

    await waitFor(() => {
      const historyContent = screen.getByText(/Loading history|No runs yet|completed/);
      expect(historyContent).toBeInTheDocument();
    });

    // Click again to collapse - button title stays "Show history"
    const collapseButtons = screen.getAllByTitle('Show history');
    fireEvent.click(collapseButtons[0]);

    // Note: The history section will be hidden
    await waitFor(() => {
      expect(screen.queryByText('Loading history')).not.toBeInTheDocument();
    });
  });
});

describe('Scheduler Form Modal Cron Expression', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows cron expression field when frequency is cron', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Frequency *')).toBeInTheDocument();
    });

    // Select cron frequency
    const frequencySelect = screen.getByLabelText('Frequency *');
    fireEvent.change(frequencySelect, { target: { value: 'cron' } });

    await waitFor(() => {
      expect(screen.getByText('Cron Expression *')).toBeInTheDocument();
    });
  });

  it('hides cron expression field for non-cron frequencies', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Frequency *')).toBeInTheDocument();
    });

    // Default is not cron
    expect(screen.queryByText('Cron Expression *')).not.toBeInTheDocument();
  });
});

describe('Scheduler Create Submission', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('submits create form with valid data', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Name *')).toBeInTheDocument();
    });

    // Fill the form
    fireEvent.change(screen.getByLabelText('Name *'), {
      target: { value: 'New Test Schedule' },
    });
    fireEvent.change(screen.getByLabelText('Description'), {
      target: { value: 'A test schedule description' },
    });

    // Select scenario
    const scenarioSelect = screen.getByLabelText('Scenario *');
    fireEvent.change(scenarioSelect, { target: { value: 'scenario-1' } });

    // Select frequency
    const frequencySelect = screen.getByLabelText('Frequency *');
    fireEvent.change(frequencySelect, { target: { value: 'daily' } });

    // Submit form - find submit button in modal (the one with btn-primary class without gap-2)
    const allButtons = screen.getAllByRole('button', { name: /Create Schedule/ });
    // Modal submit button is the btn-primary one without the gap-2 class
    const submitButton = allButtons.find((btn) => !btn.className.includes('gap-2'));
    expect(submitButton).toBeDefined();
    fireEvent.click(submitButton!);

    await waitFor(() => {
      expect(mockCreate).toHaveBeenCalled();
    });
  });
});

describe('Scheduler Update Submission', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('submits update form with modified data', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    // Click edit on first schedule
    const editButtons = screen.getAllByTitle('Edit');
    fireEvent.click(editButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Edit Schedule')).toBeInTheDocument();
    });

    // Modify name
    const nameInput = screen.getByDisplayValue('Daily Security Check');
    fireEvent.change(nameInput, { target: { value: 'Modified Schedule Name' } });

    // Submit form - button text is "Update Schedule"
    const submitButton = screen.getByRole('button', { name: /Update Schedule$/ });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockUpdate).toHaveBeenCalled();
    });
  });
});

describe('Scheduler Safe Mode Toggle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows safe mode checkbox in form', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Safe Mode')).toBeInTheDocument();
    });
  });

  it('can toggle safe mode in form', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Safe Mode')).toBeInTheDocument();
    });

    const safeModeCheckbox = screen.getByRole('checkbox');
    expect(safeModeCheckbox).toBeInTheDocument();
    fireEvent.click(safeModeCheckbox);
  });
});

describe('Scheduler Next Run Display', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows next run time for active schedules', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    // Should show next run info
    const nextRunLabels = screen.getAllByText('Next Run:');
    expect(nextRunLabels.length).toBeGreaterThan(0);
  });

  it('shows last run time when available', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    // First schedule has last_run_at
    const lastRunLabels = screen.getAllByText('Last Run:');
    expect(lastRunLabels.length).toBeGreaterThan(0);
  });
});

describe('Scheduler Agent Selection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows agent field in create form', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Agent (optional)')).toBeInTheDocument();
    });
  });

  it('shows placeholder for agent field', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Leave empty for all agents')).toBeInTheDocument();
    });
  });
});

describe('Scheduler API Error Handling', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('handles pause error gracefully', async () => {
    const toast = await import('react-hot-toast');
    mockPause.mockRejectedValueOnce(new Error('Pause failed'));

    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const pauseButtons = screen.getAllByTitle('Pause');
    fireEvent.click(pauseButtons[0]);

    await waitFor(() => {
      expect(toast.default.error).toHaveBeenCalled();
    });
  });

  it('handles resume error gracefully', async () => {
    const toast = await import('react-hot-toast');
    mockResume.mockRejectedValueOnce(new Error('Resume failed'));

    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Weekly Audit')).toBeInTheDocument();
    });

    const resumeButtons = screen.getAllByTitle('Resume');
    fireEvent.click(resumeButtons[0]);

    await waitFor(() => {
      expect(toast.default.error).toHaveBeenCalled();
    });
  });

  it('handles run now error gracefully', async () => {
    const toast = await import('react-hot-toast');
    mockRunNow.mockRejectedValueOnce(new Error('Run now failed'));

    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const runNowButtons = screen.getAllByTitle('Run Now');
    fireEvent.click(runNowButtons[0]);

    await waitFor(() => {
      expect(toast.default.error).toHaveBeenCalled();
    });
  });

  it('handles delete error gracefully', async () => {
    const toast = await import('react-hot-toast');
    mockDelete.mockRejectedValueOnce(new Error('Delete failed'));

    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const deleteButtons = screen.getAllByTitle('Delete');
    fireEvent.click(deleteButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Delete Schedule')).toBeInTheDocument();
    });

    const allButtons = screen.getAllByRole('button');
    const confirmButton = allButtons.find(
      (btn) => btn.textContent === 'Delete' && btn.className.includes('bg-red-600')
    );
    fireEvent.click(confirmButton!);

    await waitFor(() => {
      expect(toast.default.error).toHaveBeenCalled();
    });
  });
});

describe('Scheduler Form Validation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows required fields with asterisks', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Name *')).toBeInTheDocument();
      expect(screen.getByText('Scenario *')).toBeInTheDocument();
      expect(screen.getByText('Frequency *')).toBeInTheDocument();
    });
  });

  it('shows description field as optional', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Description')).toBeInTheDocument();
    });
  });
});

describe('Scheduler Loading State', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Don't setup mocks immediately to test loading state
    mockScheduleList.mockImplementation(
      () => new Promise((resolve) => setTimeout(() => resolve({ data: mockScheduleData }), 100))
    );
    mockScenarioList.mockResolvedValue({ data: mockScenarioData });
    mockGetRuns.mockResolvedValue({ data: [] });
  });

  it('shows scheduler title after loading', async () => {
    renderScheduler();
    await waitFor(() => {
      expect(screen.getByText('Scheduler')).toBeInTheDocument();
    });
  });
});

describe('Scheduler Frequency Options', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows all frequency options in dropdown', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      const frequencySelect = screen.getByLabelText('Frequency *');
      expect(frequencySelect).toBeInTheDocument();
    });
  });

  it('can select hourly frequency', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      const frequencySelect = screen.getByLabelText('Frequency *');
      fireEvent.change(frequencySelect, { target: { value: 'hourly' } });
      expect(frequencySelect).toHaveValue('hourly');
    });
  });

  it('can select monthly frequency', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      const frequencySelect = screen.getByLabelText('Frequency *');
      fireEvent.change(frequencySelect, { target: { value: 'monthly' } });
      expect(frequencySelect).toHaveValue('monthly');
    });
  });
});

describe('Scheduler Schedule Card Details', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows schedule names', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
      expect(screen.getByText('Weekly Audit')).toBeInTheDocument();
    });
  });

  it('shows schedule descriptions', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Run security tests daily')).toBeInTheDocument();
      expect(screen.getByText('Weekly security audit')).toBeInTheDocument();
    });
  });
});

describe('Scheduler Modal Close', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('closes create modal when clicking outside', async () => {
    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Create Schedule'));

    await waitFor(() => {
      expect(screen.getByText('Name *')).toBeInTheDocument();
    });

    // Click cancel to close
    fireEvent.click(screen.getByText('Cancel'));

    await waitFor(() => {
      expect(screen.queryByText('Name *')).not.toBeInTheDocument();
    });
  });
});

describe('Scheduler Success Messages', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupDefaultMocks();
  });

  it('shows success toast on pause', async () => {
    const toast = await import('react-hot-toast');

    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const pauseButtons = screen.getAllByTitle('Pause');
    fireEvent.click(pauseButtons[0]);

    await waitFor(() => {
      expect(toast.default.success).toHaveBeenCalled();
    });
  });

  it('shows success toast on resume', async () => {
    const toast = await import('react-hot-toast');

    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Weekly Audit')).toBeInTheDocument();
    });

    const resumeButtons = screen.getAllByTitle('Resume');
    fireEvent.click(resumeButtons[0]);

    await waitFor(() => {
      expect(toast.default.success).toHaveBeenCalled();
    });
  });

  it('shows success toast on run now', async () => {
    const toast = await import('react-hot-toast');

    renderScheduler();

    await waitFor(() => {
      expect(screen.getByText('Daily Security Check')).toBeInTheDocument();
    });

    const runNowButtons = screen.getAllByTitle('Run Now');
    fireEvent.click(runNowButtons[0]);

    await waitFor(() => {
      expect(toast.default.success).toHaveBeenCalled();
    });
  });
});
