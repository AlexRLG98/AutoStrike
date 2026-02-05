import { useState, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import {
  DocumentTextIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  XMarkIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
} from '@heroicons/react/24/outline';
import { api, executionApi, scenarioApi, ImportScenariosRequest, ScenarioPhase } from '../lib/api';
import { Scenario } from '../types';
import { LoadingState } from '../components/LoadingState';
import { EmptyState } from '../components/EmptyState';
import { RunExecutionModal } from '../components/RunExecutionModal';
import toast from 'react-hot-toast';

interface ImportResult {
  imported: number;
  failed: number;
  errors?: string[];
}

function ImportResultIcon({ importResult }: { readonly importResult: ImportResult }) {
  if (importResult.failed === 0) {
    return <CheckCircleIcon className="h-8 w-8 text-green-500" />;
  }
  if (importResult.imported === 0) {
    return <ExclamationTriangleIcon className="h-8 w-8 text-red-500" />;
  }
  return <ExclamationTriangleIcon className="h-8 w-8 text-yellow-500" />;
}

function ImportResultTitle({ importResult }: { readonly importResult: ImportResult }) {
  if (importResult.failed === 0) {
    return <>Import Successful</>;
  }
  if (importResult.imported === 0) {
    return <>Import Failed</>;
  }
  return <>Partial Import</>;
}

/**
 * Scenarios page component.
 * Displays a grid of attack scenarios with their phases and configuration.
 *
 * @returns The Scenarios page component
 */
export default function Scenarios() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [scenarioToRun, setScenarioToRun] = useState<Scenario | null>(null);
  const [showImportModal, setShowImportModal] = useState(false);
  const [importResult, setImportResult] = useState<{
    imported: number;
    failed: number;
    errors?: string[];
  } | null>(null);

  const { data: scenarios, isLoading } = useQuery<Scenario[]>({
    queryKey: ['scenarios'],
    queryFn: () => api.get('/scenarios').then(res => res.data),
  });

  const startMutation = useMutation({
    mutationFn: ({ scenarioId, agentPaws, safeMode }: { scenarioId: string; agentPaws: string[]; safeMode: boolean }) =>
      executionApi.start(scenarioId, agentPaws, safeMode),
    onSuccess: () => {
      toast.success('Execution started successfully');
      setScenarioToRun(null);
      navigate('/executions');
    },
    onError: (error: { response?: { data?: { error?: string } } }) => {
      const message = error.response?.data?.error || 'Failed to start execution';
      toast.error(message);
    },
  });

  const importMutation = useMutation({
    mutationFn: (data: ImportScenariosRequest) => scenarioApi.import(data),
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: ['scenarios'] });
      setImportResult({
        imported: response.data.imported,
        failed: response.data.failed,
        errors: response.data.errors,
      });
      if (response.data.imported > 0 && response.data.failed === 0) {
        toast.success(`Imported ${response.data.imported} scenario(s) successfully`);
      }
    },
    onError: (error: { response?: { data?: { error?: string; imported?: number; failed?: number; errors?: string[] } } }) => {
      const data = error.response?.data;
      if (data?.imported === undefined) {
        toast.error(data?.error || 'Failed to import scenarios');
      } else {
        setImportResult({
          imported: data.imported,
          failed: data.failed || 0,
          errors: data.errors,
        });
      }
    },
  });

  const handleRunClick = (scenario: Scenario) => {
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

  const handleExport = async () => {
    try {
      const response = await scenarioApi.exportAll();
      const blob = new Blob([JSON.stringify(response.data, null, 2)], {
        type: 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `autostrike-scenarios-${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
      toast.success('Scenarios exported successfully');
    } catch {
      toast.error('Failed to export scenarios');
    }
  };

  const handleImportClick = () => {
    setImportResult(null);
    setShowImportModal(true);
  };

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    try {
      const content = await file.text();
      const data = JSON.parse(content);

      // Handle both export format and direct array format
      const scenarios = data.scenarios || data;
      if (!Array.isArray(scenarios)) {
        toast.error('Invalid format: expected scenarios array');
        return;
      }

      importMutation.mutate({
        version: data.version || '1.0',
        scenarios: scenarios.map((s: { name: string; description?: string; phases: unknown[]; tags?: string[] }) => ({
          name: s.name,
          description: s.description,
          phases: s.phases as ScenarioPhase[],
          tags: s.tags,
        })),
      });
    } catch {
      toast.error('Failed to parse JSON file');
    }

    // Reset file input
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const closeImportModal = () => {
    setShowImportModal(false);
    setImportResult(null);
  };

  if (isLoading) {
    return <LoadingState message="Loading scenarios..." />;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold">Scenarios</h1>
        <div className="flex gap-2">
          <button
            onClick={handleImportClick}
            className="btn-secondary flex items-center gap-2"
          >
            <ArrowUpTrayIcon className="h-5 w-5" />
            Import
          </button>
          <button
            onClick={handleExport}
            className="btn-secondary flex items-center gap-2"
            disabled={!scenarios?.length}
          >
            <ArrowDownTrayIcon className="h-5 w-5" />
            Export
          </button>
          <button className="btn-primary">Create Scenario</button>
        </div>
      </div>

      {/* Hidden file input for import */}
      <input
        ref={fileInputRef}
        type="file"
        accept=".json"
        onChange={handleFileSelect}
        className="hidden"
      />

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {scenarios?.map((scenario) => (
          <div key={scenario.id} className="card">
            <div className="flex items-start justify-between">
              <div>
                <h3 className="font-semibold text-lg">{scenario.name}</h3>
                <p className="text-sm text-gray-500 mt-1">{scenario.description}</p>
              </div>
              <button
                onClick={() => handleRunClick(scenario)}
                className="btn-primary text-sm"
              >
                Run
              </button>
            </div>

            <div className="mt-4">
              <p className="text-sm text-gray-500 mb-2">Phases</p>
              <div className="space-y-2">
                {scenario.phases.map((phase, idx) => (
                  <div key={`${phase.name}-${idx}`} className="flex items-center gap-2">
                    <span className="w-6 h-6 rounded-full bg-primary-100 text-primary-600 flex items-center justify-center text-xs font-medium">
                      {idx + 1}
                    </span>
                    <span className="text-sm">{phase.name}</span>
                    <span className="text-xs text-gray-400">
                      ({phase.techniques.length} techniques)
                    </span>
                  </div>
                ))}
              </div>
            </div>

            <div className="mt-4 pt-4 border-t border-gray-100">
              <div className="flex gap-1 flex-wrap">
                {scenario.tags?.map((tag) => (
                  <span key={tag} className="badge bg-gray-100 text-gray-700">
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>

      {scenarios?.length === 0 && (
        <EmptyState
          icon={DocumentTextIcon}
          title="No scenarios created"
          description="Create an attack scenario to test your defenses"
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

      {/* Import Modal */}
      {showImportModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl shadow-xl max-w-md w-full mx-4">
            <div className="flex items-center justify-between p-6 border-b">
              <h2 className="text-xl font-semibold">Import Scenarios</h2>
              <button
                onClick={closeImportModal}
                className="p-2 hover:bg-gray-100 rounded-lg"
              >
                <XMarkIcon className="h-5 w-5" />
              </button>
            </div>
            <div className="p-6">
              {importResult ? (
                <div className="space-y-4">
                  {/* Import Results */}
                  <div className="flex items-center gap-3">
                    <ImportResultIcon importResult={importResult} />
                    <div>
                      <p className="font-medium">
                        <ImportResultTitle importResult={importResult} />
                      </p>
                      <p className="text-sm text-gray-600">
                        {importResult.imported} imported, {importResult.failed} failed
                      </p>
                    </div>
                  </div>

                  {importResult.errors && importResult.errors.length > 0 && (
                    <div className="bg-red-50 border border-red-200 rounded-lg p-3 max-h-40 overflow-y-auto">
                      <p className="text-sm font-medium text-red-700 mb-2">Errors:</p>
                      <ul className="text-xs text-red-600 space-y-1">
                        {importResult.errors.map((error, idx) => (
                          <li key={`error-${idx}-${error.slice(0, 20)}`}>{error}</li>
                        ))}
                      </ul>
                    </div>
                  )}

                  <div className="flex justify-end gap-3">
                    <button
                      onClick={() => setImportResult(null)}
                      className="btn-secondary"
                    >
                      Import More
                    </button>
                    <button onClick={closeImportModal} className="btn-primary">
                      Done
                    </button>
                  </div>
                </div>
              ) : (
                <div className="space-y-4">
                  <p className="text-sm text-gray-600">
                    Upload a JSON file containing scenarios to import. The file should be in AutoStrike export format.
                  </p>
                  <button
                    type="button"
                    className="w-full border-2 border-dashed border-gray-300 rounded-lg p-8 text-center cursor-pointer hover:border-primary-500 transition-colors bg-transparent"
                    onClick={() => fileInputRef.current?.click()}
                  >
                    <ArrowUpTrayIcon className="h-10 w-10 mx-auto text-gray-400 mb-3" />
                    <p className="text-sm text-gray-600">
                      Click to select a JSON file
                    </p>
                    <p className="text-xs text-gray-400 mt-1">
                      or drag and drop
                    </p>
                  </button>
                  {importMutation.isPending && (
                    <div className="flex items-center justify-center gap-2 text-gray-600">
                      <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                      </svg>
                      <span>Importing...</span>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
