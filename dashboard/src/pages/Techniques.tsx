import { useState, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ShieldExclamationIcon,
  ArrowUpTrayIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline';
import { api, techniqueApi, Technique as TechniqueType } from '../lib/api';
import { getTacticBadgeColor, formatTacticName } from '../lib/tacticColors';
import { Technique } from '../types';
import { LoadingState } from '../components/LoadingState';
import { EmptyState } from '../components/EmptyState';
import { TableHeader, TableBody, TableRow, TABLE_CELL_CLASS, TABLE_CELL_NOWRAP_CLASS } from '../components/Table';
import { Modal } from '../components/Modal';
import toast from 'react-hot-toast';

interface ImportResult {
  imported: number;
  failed: number;
  errors?: string[];
}

/**
 * Returns the appropriate icon for an import result.
 */
function ImportResultIcon({ importResult }: { readonly importResult: ImportResult }) {
  if (importResult.failed === 0) {
    return <CheckCircleIcon className="h-8 w-8 text-green-500" />;
  }
  if (importResult.imported === 0) {
    return <ExclamationTriangleIcon className="h-8 w-8 text-red-500" />;
  }
  return <ExclamationTriangleIcon className="h-8 w-8 text-yellow-500" />;
}

/**
 * Returns the appropriate title for an import result.
 */
function getImportResultTitle(importResult: ImportResult): string {
  if (importResult.failed === 0) {
    return 'Import Successful';
  }
  if (importResult.imported === 0) {
    return 'Import Failed';
  }
  return 'Partial Import';
}

/**
 * Techniques page component.
 * Displays a table of available MITRE ATT&CK techniques.
 *
 * @returns The Techniques page component
 */
export default function Techniques() {
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [showImportModal, setShowImportModal] = useState(false);
  const [importResult, setImportResult] = useState<ImportResult | null>(null);

  const { data: techniques, isLoading } = useQuery<Technique[]>({
    queryKey: ['techniques'],
    queryFn: () => api.get('/techniques').then(res => res.data),
  });

  const importMutation = useMutation({
    mutationFn: (techniques: TechniqueType[]) => techniqueApi.import(techniques),
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: ['techniques'] });
      setImportResult({
        imported: response.data.imported,
        failed: response.data.failed,
        errors: response.data.errors,
      });
      if (response.data.imported > 0 && response.data.failed === 0) {
        toast.success(`Imported ${response.data.imported} technique(s) successfully`);
      }
    },
    onError: (error: { response?: { data?: { error?: string } } }) => {
      toast.error(error.response?.data?.error || 'Failed to import techniques');
    },
  });

  const handleImportClick = () => {
    setImportResult(null);
    setShowImportModal(true);
  };

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    try {
      const content = await file.text();
      let techniques: TechniqueType[];

      // Support both JSON and YAML-like formats
      if (file.name.endsWith('.json')) {
        const data = JSON.parse(content);
        techniques = Array.isArray(data) ? data : data.techniques || [];
      } else {
        // For YAML files, we need to parse them
        // Simple YAML array parser for technique format
        toast.error('Please use JSON format for importing techniques');
        return;
      }

      if (!Array.isArray(techniques) || techniques.length === 0) {
        toast.error('Invalid format: expected techniques array');
        return;
      }

      importMutation.mutate(techniques);
    } catch {
      toast.error('Failed to parse file');
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
    return <LoadingState message="Loading techniques..." />;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold">Techniques</h1>
        <button onClick={handleImportClick} className="btn-primary flex items-center gap-2">
          <ArrowUpTrayIcon className="h-5 w-5" />
          Import Techniques
        </button>
      </div>

      {/* Hidden file input for import */}
      <input
        ref={fileInputRef}
        type="file"
        accept=".json"
        onChange={handleFileSelect}
        className="hidden"
      />

      <div className="card overflow-hidden">
        <table className="w-full">
          <TableHeader columns={['ID', 'Name', 'Tactic', 'Platforms', 'Safe']} />
          <TableBody>
            {techniques?.map((technique) => (
              <TableRow key={technique.id}>
                <td className={TABLE_CELL_NOWRAP_CLASS}>
                  <span className="font-mono text-sm">{technique.id}</span>
                </td>
                <td className={TABLE_CELL_CLASS}>
                  <p className="font-medium">{technique.name}</p>
                  <p className="text-sm text-gray-500 truncate max-w-md">
                    {technique.description}
                  </p>
                </td>
                <td className={TABLE_CELL_NOWRAP_CLASS}>
                  <span className={`badge ${getTacticBadgeColor(technique.tactic)}`}>
                    {formatTacticName(technique.tactic)}
                  </span>
                </td>
                <td className={TABLE_CELL_CLASS}>
                  <div className="flex gap-1">
                    {technique.platforms.map((platform) => (
                      <span key={platform} className="badge bg-gray-100 text-gray-700">
                        {platform}
                      </span>
                    ))}
                  </div>
                </td>
                <td className={TABLE_CELL_NOWRAP_CLASS}>
                  <span className={`badge ${technique.is_safe ? 'badge-success' : 'badge-danger'}`}>
                    {technique.is_safe ? 'Safe' : 'Unsafe'}
                  </span>
                </td>
              </TableRow>
            ))}
          </TableBody>
        </table>
      </div>

      {techniques?.length === 0 && (
        <EmptyState
          icon={ShieldExclamationIcon}
          title="No techniques loaded"
          description="Import techniques from Atomic Red Team"
        />
      )}

      {/* Import Modal */}
      {showImportModal && (
        <Modal
          title="Import Techniques"
          onClose={closeImportModal}
          footer={importResult ? (
            <>
              <button onClick={() => setImportResult(null)} className="btn-secondary">
                Import More
              </button>
              <button onClick={closeImportModal} className="btn-primary">
                Done
              </button>
            </>
          ) : undefined}
        >
          {importResult ? (
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <ImportResultIcon importResult={importResult} />
                <div>
                  <p className="font-medium">{getImportResultTitle(importResult)}</p>
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
            </div>
          ) : (
            <div className="space-y-4">
              <p className="text-sm text-gray-600">
                Upload a JSON file containing MITRE ATT&CK techniques to import.
              </p>
              <div className="bg-gray-50 rounded-lg p-3 text-xs text-gray-600">
                <p className="font-medium mb-1">Expected format:</p>
                <pre className="overflow-x-auto">{`[{
  "id": "T1082",
  "name": "System Info",
  "tactic": "discovery",
  "platforms": ["windows", "linux"],
  "is_safe": true,
  ...
}]`}</pre>
              </div>
              <button
                type="button"
                className="w-full border-2 border-dashed border-gray-300 rounded-lg p-8 text-center cursor-pointer hover:border-primary-500 transition-colors bg-transparent"
                onClick={() => fileInputRef.current?.click()}
              >
                <ArrowUpTrayIcon className="h-10 w-10 mx-auto text-gray-400 mb-3" />
                <p className="text-sm text-gray-600">Click to select a JSON file</p>
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
        </Modal>
      )}
    </div>
  );
}
