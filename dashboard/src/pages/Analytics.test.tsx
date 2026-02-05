import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Analytics from './Analytics';

// Mock Chart.js
vi.mock('react-chartjs-2', () => ({
  Line: () => <div data-testid="line-chart">Line Chart</div>,
  Bar: () => <div data-testid="bar-chart">Bar Chart</div>,
}));

vi.mock('chart.js', () => ({
  Chart: {
    register: vi.fn(),
  },
  CategoryScale: vi.fn(),
  LinearScale: vi.fn(),
  PointElement: vi.fn(),
  LineElement: vi.fn(),
  BarElement: vi.fn(),
  Title: vi.fn(),
  Tooltip: vi.fn(),
  Legend: vi.fn(),
  Filler: vi.fn(),
}));

// Mock the API
const mockCompare = vi.fn();
const mockTrend = vi.fn();
const mockSummary = vi.fn();

vi.mock('../lib/api', () => ({
  analyticsApi: {
    compare: (...args: unknown[]) => mockCompare(...args),
    trend: (...args: unknown[]) => mockTrend(...args),
    summary: (...args: unknown[]) => mockSummary(...args),
  },
}));

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
}

function renderAnalytics() {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <Analytics />
    </QueryClientProvider>
  );
}

const mockComparisonData = {
  current: {
    period: '30_days',
    start_date: '2024-01-01',
    end_date: '2024-01-30',
    execution_count: 25,
    average_score: 78.5,
    total_blocked: 150,
    total_detected: 75,
    total_successful: 25,
    total_techniques: 50,
  },
  previous: {
    period: '30_days',
    start_date: '2023-12-01',
    end_date: '2023-12-30',
    execution_count: 20,
    average_score: 65.0,
    total_blocked: 100,
    total_detected: 60,
    total_successful: 40,
    total_techniques: 50,
  },
  score_change: 13.5,
  score_trend: 'improving' as const,
  blocked_change: 50,
  detected_change: 15,
};

const mockTrendData = {
  period: '30_days',
  data_points: [
    {
      date: '2024-01-01',
      average_score: 70,
      execution_count: 5,
      blocked: 30,
      detected: 15,
      successful: 5,
    },
    {
      date: '2024-01-15',
      average_score: 75,
      execution_count: 8,
      blocked: 50,
      detected: 25,
      successful: 10,
    },
    {
      date: '2024-01-30',
      average_score: 80,
      execution_count: 12,
      blocked: 70,
      detected: 35,
      successful: 10,
    },
  ],
  summary: {
    start_score: 70,
    end_score: 80,
    average_score: 75,
    max_score: 85,
    min_score: 65,
    total_executions: 25,
    overall_trend: 'improving' as const,
    percentage_change: 14.3,
  },
};

const mockSummaryData = {
  total_executions: 100,
  completed_executions: 85,
  average_score: 72.5,
  best_score: 95.0,
  worst_score: 45.0,
  scores_by_scenario: {
    'Discovery Test': 85,
    'Lateral Movement': 70,
    'Defense Evasion': 65,
  },
  executions_by_status: {
    completed: 85,
    failed: 10,
    cancelled: 5,
  },
};

describe('Analytics Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCompare.mockResolvedValue({ data: mockComparisonData });
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockResolvedValue({ data: mockSummaryData });
  });

  it('renders loading state initially', () => {
    // Make the promises never resolve to keep loading state
    mockCompare.mockReturnValue(new Promise(() => {}));
    mockTrend.mockReturnValue(new Promise(() => {}));
    mockSummary.mockReturnValue(new Promise(() => {}));

    renderAnalytics();
    expect(screen.getByText('Loading analytics...')).toBeInTheDocument();
  });

  it('renders page title after loading', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Analytics')).toBeInTheDocument();
    });
  });

  it('displays period selector with default 30 days', async () => {
    renderAnalytics();

    await waitFor(() => {
      const select = screen.getByRole('combobox');
      expect(select).toHaveValue('30');
    });
  });

  it('displays all period options', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Last 7 days')).toBeInTheDocument();
      expect(screen.getByText('Last 30 days')).toBeInTheDocument();
      expect(screen.getByText('Last 90 days')).toBeInTheDocument();
    });
  });

  it('changes period when selector changes', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Analytics')).toBeInTheDocument();
    });

    const select = screen.getByRole('combobox');
    fireEvent.change(select, { target: { value: '7' } });

    await waitFor(() => {
      expect(mockCompare).toHaveBeenCalledWith(7);
      expect(mockTrend).toHaveBeenCalledWith(7);
      expect(mockSummary).toHaveBeenCalledWith(7);
    });
  });

  it('displays average score from comparison data', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('78.5%')).toBeInTheDocument();
    });
  });

  it('displays execution count', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('25')).toBeInTheDocument();
      expect(screen.getByText('20 previous period')).toBeInTheDocument();
    });
  });

  it('displays blocked attacks count', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Blocked Attacks')).toBeInTheDocument();
      expect(screen.getByText('150')).toBeInTheDocument();
    });
  });

  it('displays detected attacks count', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Detected Attacks')).toBeInTheDocument();
      expect(screen.getByText('75')).toBeInTheDocument();
    });
  });

  it('displays score change with positive prefix', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('+13.5% vs previous period')).toBeInTheDocument();
    });
  });

  it('displays trend summary statistics', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Min Score')).toBeInTheDocument();
      expect(screen.getByText('65.0%')).toBeInTheDocument();
      expect(screen.getByText('Max Score')).toBeInTheDocument();
      expect(screen.getByText('85.0%')).toBeInTheDocument();
    });
  });

  it('displays execution summary totals', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Execution Summary')).toBeInTheDocument();
      expect(screen.getByText('Total Executions')).toBeInTheDocument();
      expect(screen.getByText('100')).toBeInTheDocument();
    });
  });

  it('displays completed executions', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Completed')).toBeInTheDocument();
      expect(screen.getByText('85')).toBeInTheDocument();
    });
  });

  it('displays best and worst scores', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Best Score')).toBeInTheDocument();
      expect(screen.getByText('95.0%')).toBeInTheDocument();
      expect(screen.getByText('Worst Score')).toBeInTheDocument();
      expect(screen.getByText('45.0%')).toBeInTheDocument();
    });
  });

  it('renders charts', async () => {
    renderAnalytics();

    await waitFor(() => {
      const lineCharts = screen.getAllByTestId('line-chart');
      const barCharts = screen.getAllByTestId('bar-chart');
      expect(lineCharts.length).toBeGreaterThan(0);
      expect(barCharts.length).toBeGreaterThan(0);
    });
  });

  it('displays performance by scenario section', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Performance by Scenario')).toBeInTheDocument();
      expect(screen.getByText('Discovery Test')).toBeInTheDocument();
      expect(screen.getByText('Lateral Movement')).toBeInTheDocument();
      expect(screen.getByText('Defense Evasion')).toBeInTheDocument();
    });
  });

  it('displays scenario scores', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('85%')).toBeInTheDocument();
      expect(screen.getByText('70%')).toBeInTheDocument();
    });
  });
});

describe('Analytics Error State', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('displays error state when compare fails', async () => {
    mockCompare.mockRejectedValue(new Error('Network error'));
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockResolvedValue({ data: mockSummaryData });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Failed to load analytics')).toBeInTheDocument();
      expect(screen.getByText('Network error')).toBeInTheDocument();
    });
  });

  it('displays error state when trend fails', async () => {
    mockCompare.mockResolvedValue({ data: mockComparisonData });
    mockTrend.mockRejectedValue(new Error('Trend fetch failed'));
    mockSummary.mockResolvedValue({ data: mockSummaryData });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Failed to load analytics')).toBeInTheDocument();
      expect(screen.getByText('Trend fetch failed')).toBeInTheDocument();
    });
  });

  it('displays error state when summary fails', async () => {
    mockCompare.mockResolvedValue({ data: mockComparisonData });
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockRejectedValue(new Error('Summary error'));

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Failed to load analytics')).toBeInTheDocument();
      expect(screen.getByText('Summary error')).toBeInTheDocument();
    });
  });

  it('shows try again button on error', async () => {
    mockCompare.mockRejectedValue(new Error('Network error'));
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockResolvedValue({ data: mockSummaryData });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Try Again')).toBeInTheDocument();
    });
  });

  it('refetches data when try again is clicked', async () => {
    mockCompare.mockRejectedValueOnce(new Error('Network error'));
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockResolvedValue({ data: mockSummaryData });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Try Again')).toBeInTheDocument();
    });

    // Reset mocks to succeed
    mockCompare.mockResolvedValue({ data: mockComparisonData });

    const retryButton = screen.getByText('Try Again');
    fireEvent.click(retryButton);

    await waitFor(() => {
      expect(mockCompare).toHaveBeenCalledTimes(2);
    });
  });

  it('shows fallback error message when no specific message', async () => {
    mockCompare.mockRejectedValue({});
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockResolvedValue({ data: mockSummaryData });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('An error occurred while fetching data')).toBeInTheDocument();
    });
  });
});

describe('Analytics Trend Icons', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockResolvedValue({ data: mockSummaryData });
  });

  it('shows improving trend indicator', async () => {
    mockCompare.mockResolvedValue({
      data: { ...mockComparisonData, score_trend: 'improving' },
    });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Average Score')).toBeInTheDocument();
    });
  });

  it('handles declining trend', async () => {
    mockCompare.mockResolvedValue({
      data: { ...mockComparisonData, score_trend: 'declining', score_change: -5.5 },
    });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('-5.5% vs previous period')).toBeInTheDocument();
    });
  });

  it('handles stable trend', async () => {
    mockCompare.mockResolvedValue({
      data: { ...mockComparisonData, score_trend: 'stable', score_change: 0 },
    });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('0% vs previous period')).toBeInTheDocument();
    });
  });
});

describe('Analytics Empty Data', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCompare.mockResolvedValue({ data: mockComparisonData });
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockResolvedValue({ data: mockSummaryData });
  });

  it('shows message when no scenario data', async () => {
    mockSummary.mockResolvedValue({
      data: { ...mockSummaryData, scores_by_scenario: {} },
    });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('No scenario data available')).toBeInTheDocument();
    });
  });

  it('handles zero values gracefully', async () => {
    mockCompare.mockResolvedValue({
      data: {
        ...mockComparisonData,
        current: {
          ...mockComparisonData.current,
          average_score: 0,
          execution_count: 0,
          total_blocked: 0,
          total_detected: 0,
        },
      },
    });

    renderAnalytics();

    await waitFor(() => {
      // Page still renders correctly with zero values
      expect(screen.getByText('Average Score')).toBeInTheDocument();
      expect(screen.getByText('Executions')).toBeInTheDocument();
      expect(screen.getByText('Blocked Attacks')).toBeInTheDocument();
    });
  });

  it('handles null/undefined score change', async () => {
    mockCompare.mockResolvedValue({
      data: {
        ...mockComparisonData,
        score_change: undefined,
      },
    });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('0% vs previous period')).toBeInTheDocument();
    });
  });

  it('handles empty data points in trend', async () => {
    mockTrend.mockResolvedValue({
      data: {
        ...mockTrendData,
        data_points: [],
        summary: undefined,
      },
    });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Score Trend')).toBeInTheDocument();
    });
  });
});

describe('Analytics Period Changes', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCompare.mockResolvedValue({ data: mockComparisonData });
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockResolvedValue({ data: mockSummaryData });
  });

  it('fetches with 7 days period', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Analytics')).toBeInTheDocument();
    });

    const select = screen.getByRole('combobox');
    fireEvent.change(select, { target: { value: '7' } });

    await waitFor(() => {
      expect(mockCompare).toHaveBeenLastCalledWith(7);
    });
  });

  it('fetches with 90 days period', async () => {
    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('Analytics')).toBeInTheDocument();
    });

    const select = screen.getByRole('combobox');
    fireEvent.change(select, { target: { value: '90' } });

    await waitFor(() => {
      expect(mockCompare).toHaveBeenLastCalledWith(90);
    });
  });
});

describe('Analytics Blocked/Detected Changes', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTrend.mockResolvedValue({ data: mockTrendData });
    mockSummary.mockResolvedValue({ data: mockSummaryData });
  });

  it('shows positive blocked change with plus', async () => {
    mockCompare.mockResolvedValue({
      data: { ...mockComparisonData, blocked_change: 50 },
    });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('+50 vs previous')).toBeInTheDocument();
    });
  });

  it('shows positive detected change with plus', async () => {
    mockCompare.mockResolvedValue({
      data: { ...mockComparisonData, detected_change: 15 },
    });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('+15 vs previous')).toBeInTheDocument();
    });
  });

  it('shows negative blocked change without plus', async () => {
    mockCompare.mockResolvedValue({
      data: { ...mockComparisonData, blocked_change: -10 },
    });

    renderAnalytics();

    await waitFor(() => {
      expect(screen.getByText('-10 vs previous')).toBeInTheDocument();
    });
  });

  it('shows zero change without prefix', async () => {
    mockCompare.mockResolvedValue({
      data: { ...mockComparisonData, blocked_change: 0, detected_change: 0 },
    });

    renderAnalytics();

    await waitFor(() => {
      const zeroTexts = screen.getAllByText('0 vs previous');
      expect(zeroTexts.length).toBe(2);
    });
  });
});
