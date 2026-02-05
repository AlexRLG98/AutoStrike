import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';
import { Line, Bar } from 'react-chartjs-2';
import {
  ArrowTrendingUpIcon,
  ArrowTrendingDownIcon,
  MinusIcon,
  CalendarDaysIcon,
  ExclamationTriangleIcon,
  ArrowPathIcon,
} from '@heroicons/react/24/outline';
import { analyticsApi, ScoreComparison, ScoreTrend, ExecutionSummary } from '../lib/api';
import { LoadingState } from '../components/LoadingState';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  Filler
);

function formatScoreChange(change?: number): string {
  if (!change) return '0';
  const prefix = change > 0 ? '+' : '';
  return `${prefix}${change.toFixed(1)}`;
}

/**
 * Analytics page component.
 * Displays security score trends, comparisons, and execution analytics.
 *
 * @returns The Analytics page component
 */
export default function Analytics() {
  const [period, setPeriod] = useState<7 | 30 | 90>(30);

  const {
    data: comparison,
    isLoading: comparisonLoading,
    error: comparisonError,
    refetch: refetchComparison,
  } = useQuery<ScoreComparison>({
    queryKey: ['analytics', 'compare', period],
    queryFn: () => analyticsApi.compare(period).then((res) => res.data),
  });

  const {
    data: trend,
    isLoading: trendLoading,
    error: trendError,
    refetch: refetchTrend,
  } = useQuery<ScoreTrend>({
    queryKey: ['analytics', 'trend', period],
    queryFn: () => analyticsApi.trend(period).then((res) => res.data),
  });

  const {
    data: summary,
    isLoading: summaryLoading,
    error: summaryError,
    refetch: refetchSummary,
  } = useQuery<ExecutionSummary>({
    queryKey: ['analytics', 'summary', period],
    queryFn: () => analyticsApi.summary(period).then((res) => res.data),
  });

  const isLoading = comparisonLoading || trendLoading || summaryLoading;
  const hasError = comparisonError || trendError || summaryError;

  const handleRefresh = () => {
    refetchComparison();
    refetchTrend();
    refetchSummary();
  };

  if (isLoading) {
    return <LoadingState message="Loading analytics..." />;
  }

  if (hasError) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px]">
        <ExclamationTriangleIcon className="h-16 w-16 text-red-400 mb-4" />
        <h2 className="text-xl font-semibold text-gray-700 mb-2">Failed to load analytics</h2>
        <p className="text-gray-500 mb-6">
          {comparisonError?.message || trendError?.message || summaryError?.message || 'An error occurred while fetching data'}
        </p>
        <button
          onClick={handleRefresh}
          className="btn-primary flex items-center gap-2"
        >
          <ArrowPathIcon className="h-5 w-5" />
          Try Again
        </button>
      </div>
    );
  }

  const getTrendIcon = (trendType: string | undefined) => {
    switch (trendType) {
      case 'improving':
        return <ArrowTrendingUpIcon className="h-5 w-5 text-green-500" />;
      case 'declining':
        return <ArrowTrendingDownIcon className="h-5 w-5 text-red-500" />;
      default:
        return <MinusIcon className="h-5 w-5 text-gray-400" />;
    }
  };

  const getTrendColor = (trendType: string | undefined) => {
    switch (trendType) {
      case 'improving':
        return 'text-green-500';
      case 'declining':
        return 'text-red-500';
      default:
        return 'text-gray-500';
    }
  };

  const trendChartData = {
    labels: trend?.data_points.map((p) => p.date) || [],
    datasets: [
      {
        label: 'Average Score',
        data: trend?.data_points.map((p) => p.average_score) || [],
        borderColor: '#3b82f6',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        fill: true,
        tension: 0.4,
      },
    ],
  };

  const executionChartData = {
    labels: trend?.data_points.map((p) => p.date) || [],
    datasets: [
      {
        label: 'Blocked',
        data: trend?.data_points.map((p) => p.blocked) || [],
        backgroundColor: '#22c55e',
      },
      {
        label: 'Detected',
        data: trend?.data_points.map((p) => p.detected) || [],
        backgroundColor: '#f59e0b',
      },
      {
        label: 'Successful',
        data: trend?.data_points.map((p) => p.successful) || [],
        backgroundColor: '#ef4444',
      },
    ],
  };

  const statusData = summary?.executions_by_status || {};
  const statusChartData = {
    labels: Object.keys(statusData),
    datasets: [
      {
        label: 'Executions',
        data: Object.values(statusData),
        backgroundColor: [
          '#22c55e', // completed - green
          '#ef4444', // failed - red
          '#f59e0b', // pending - amber
          '#3b82f6', // running - blue
          '#6b7280', // cancelled - gray
        ],
      },
    ],
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'bottom' as const,
      },
    },
    scales: {
      y: {
        beginAtZero: true,
      },
    },
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold">Analytics</h1>
        <div className="flex items-center gap-2">
          <CalendarDaysIcon className="h-5 w-5 text-gray-400" />
          <select
            value={period}
            onChange={(e) => setPeriod(Number(e.target.value) as 7 | 30 | 90)}
            className="border border-gray-300 rounded-lg px-3 py-2"
          >
            <option value={7}>Last 7 days</option>
            <option value={30}>Last 30 days</option>
            <option value={90}>Last 90 days</option>
          </select>
        </div>
      </div>

      {/* Period Comparison */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="card">
          <div className="flex items-center justify-between mb-2">
            <p className="text-sm text-gray-500">Average Score</p>
            {getTrendIcon(comparison?.score_trend)}
          </div>
          <p className="text-3xl font-bold text-primary-600">
            {comparison?.current.average_score.toFixed(1) || 0}%
          </p>
          <p className={`text-sm ${getTrendColor(comparison?.score_trend)}`}>
            {formatScoreChange(comparison?.score_change)}% vs previous period
          </p>
        </div>

        <div className="card">
          <p className="text-sm text-gray-500 mb-2">Executions</p>
          <p className="text-3xl font-bold">
            {comparison?.current.execution_count || 0}
          </p>
          <p className="text-sm text-gray-400">
            {comparison?.previous.execution_count || 0} previous period
          </p>
        </div>

        <div className="card">
          <p className="text-sm text-gray-500 mb-2">Blocked Attacks</p>
          <p className="text-3xl font-bold text-green-600">
            {comparison?.current.total_blocked || 0}
          </p>
          <p className="text-sm text-gray-400">
            {comparison?.blocked_change && comparison.blocked_change > 0 ? '+' : ''}
            {comparison?.blocked_change || 0} vs previous
          </p>
        </div>

        <div className="card">
          <p className="text-sm text-gray-500 mb-2">Detected Attacks</p>
          <p className="text-3xl font-bold text-amber-600">
            {comparison?.current.total_detected || 0}
          </p>
          <p className="text-sm text-gray-400">
            {comparison?.detected_change && comparison.detected_change > 0 ? '+' : ''}
            {comparison?.detected_change || 0} vs previous
          </p>
        </div>
      </div>

      {/* Score Trend */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
        <div className="card">
          <h2 className="text-lg font-semibold mb-4">Score Trend</h2>
          <div className="h-64">
            <Line data={trendChartData} options={chartOptions} />
          </div>
          {trend?.summary && (
            <div className="mt-4 pt-4 border-t border-gray-100">
              <div className="flex justify-between text-sm">
                <span className="text-gray-500">Min Score</span>
                <span className="font-medium">{trend.summary.min_score.toFixed(1)}%</span>
              </div>
              <div className="flex justify-between text-sm mt-1">
                <span className="text-gray-500">Max Score</span>
                <span className="font-medium">{trend.summary.max_score.toFixed(1)}%</span>
              </div>
              <div className="flex justify-between text-sm mt-1">
                <span className="text-gray-500">Average</span>
                <span className="font-medium">{trend.summary.average_score.toFixed(1)}%</span>
              </div>
            </div>
          )}
        </div>

        <div className="card">
          <h2 className="text-lg font-semibold mb-4">Detection Results Over Time</h2>
          <div className="h-64">
            <Bar
              data={executionChartData}
              options={{
                ...chartOptions,
                scales: {
                  x: { stacked: true },
                  y: { stacked: true, beginAtZero: true },
                },
              }}
            />
          </div>
        </div>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="card">
          <h2 className="text-lg font-semibold mb-4">Execution Summary</h2>
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-gray-500">Total Executions</span>
              <span className="font-semibold">{summary?.total_executions || 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Completed</span>
              <span className="font-semibold text-green-600">
                {summary?.completed_executions || 0}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Best Score</span>
              <span className="font-semibold">
                {summary?.best_score?.toFixed(1) || 0}%
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Worst Score</span>
              <span className="font-semibold">
                {summary?.worst_score?.toFixed(1) || 0}%
              </span>
            </div>
          </div>
        </div>

        <div className="card">
          <h2 className="text-lg font-semibold mb-4">Executions by Status</h2>
          <div className="h-48">
            <Bar
              data={statusChartData}
              options={{
                ...chartOptions,
                indexAxis: 'y' as const,
              }}
            />
          </div>
        </div>

        <div className="card">
          <h2 className="text-lg font-semibold mb-4">Performance by Scenario</h2>
          <div className="space-y-3 max-h-64 overflow-y-auto">
            {Object.entries(summary?.scores_by_scenario || {}).map(([scenarioId, score]) => (
              <div key={scenarioId} className="flex justify-between items-center">
                <span className="text-gray-700 truncate max-w-[60%]" title={scenarioId}>
                  {scenarioId}
                </span>
                <div className="flex items-center gap-2">
                  <div className="w-24 bg-gray-200 rounded-full h-2">
                    <div
                      className="bg-primary-600 h-2 rounded-full"
                      style={{ width: `${Math.min(score, 100)}%` }}
                    />
                  </div>
                  <span className="font-semibold w-12 text-right">
                    {score.toFixed(0)}%
                  </span>
                </div>
              </div>
            ))}
            {Object.keys(summary?.scores_by_scenario || {}).length === 0 && (
              <p className="text-gray-500 text-sm">No scenario data available</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
