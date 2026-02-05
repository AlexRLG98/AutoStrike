import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ComputerDesktopIcon, ClipboardIcon, CheckIcon } from '@heroicons/react/24/outline';
import { api } from '../lib/api';
import { formatDistanceToNow } from 'date-fns';
import { Agent } from '../types';
import { LoadingState } from '../components/LoadingState';
import { EmptyState } from '../components/EmptyState';
import { Modal } from '../components/Modal';

/**
 * Agents page component.
 * Displays a grid of connected agents with their status and details.
 *
 * @returns The Agents page component
 */
export default function Agents() {
  const [showDeployModal, setShowDeployModal] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState<string | null>(null);

  const { data: agents, isLoading } = useQuery<Agent[]>({
    queryKey: ['agents'],
    queryFn: () => api.get('/agents').then(res => res.data),
  });

  const serverUrl = globalThis.location.origin.replace('http:', 'https:');
  const commands = {
    linux: `./dist/autostrike-agent --server ${serverUrl}`,
    windows: String.raw`.\\dist\\autostrike-agent.exe --server ${serverUrl}`,
    docker: `docker run autostrike-agent --server ${serverUrl}`,
  };

  const copyToClipboard = async (text: string, key: string) => {
    await navigator.clipboard.writeText(text);
    setCopiedCommand(key);
    setTimeout(() => setCopiedCommand(null), 2000);
  };

  if (isLoading) {
    return <LoadingState message="Loading agents..." />;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold">Agents</h1>
        <button className="btn-primary" onClick={() => setShowDeployModal(true)}>
          Add Agent
        </button>
      </div>

      {/* Deploy Agent Modal */}
      {showDeployModal && (
        <Modal
          title="Deploy Agent"
          onClose={() => setShowDeployModal(false)}
          maxWidth="max-w-lg"
          footer={<button onClick={() => setShowDeployModal(false)} className="btn-primary">Close</button>}
        >
          <div className="space-y-4">
            <p className="text-gray-600">
              Download the agent binary for your platform and run:
            </p>
            <div className="space-y-3">
              <div>
                <span className="text-sm font-medium text-gray-700">Linux / macOS</span>
                <div className="mt-1 flex items-center gap-2">
                  <code className="flex-1 bg-gray-100 px-3 py-2 rounded text-sm font-mono overflow-x-auto">
                    {commands.linux}
                  </code>
                  <button onClick={() => copyToClipboard(commands.linux, 'linux')} className="p-2 hover:bg-gray-100 rounded" title="Copy">
                    {copiedCommand === 'linux' ? <CheckIcon className="h-5 w-5 text-green-500" /> : <ClipboardIcon className="h-5 w-5 text-gray-400" />}
                  </button>
                </div>
              </div>
              <div>
                <span className="text-sm font-medium text-gray-700">Windows</span>
                <div className="mt-1 flex items-center gap-2">
                  <code className="flex-1 bg-gray-100 px-3 py-2 rounded text-sm font-mono overflow-x-auto">
                    {commands.windows}
                  </code>
                  <button onClick={() => copyToClipboard(commands.windows, 'windows')} className="p-2 hover:bg-gray-100 rounded" title="Copy">
                    {copiedCommand === 'windows' ? <CheckIcon className="h-5 w-5 text-green-500" /> : <ClipboardIcon className="h-5 w-5 text-gray-400" />}
                  </button>
                </div>
              </div>
              <div>
                <span className="text-sm font-medium text-gray-700">Docker</span>
                <div className="mt-1 flex items-center gap-2">
                  <code className="flex-1 bg-gray-100 px-3 py-2 rounded text-sm font-mono overflow-x-auto">
                    {commands.docker}
                  </code>
                  <button onClick={() => copyToClipboard(commands.docker, 'docker')} className="p-2 hover:bg-gray-100 rounded" title="Copy">
                    {copiedCommand === 'docker' ? <CheckIcon className="h-5 w-5 text-green-500" /> : <ClipboardIcon className="h-5 w-5 text-gray-400" />}
                  </button>
                </div>
              </div>
            </div>
            <p className="text-sm text-gray-500 mt-4">
              The agent will automatically register with the server once started.
            </p>
          </div>
        </Modal>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {agents?.map((agent) => (
          <div key={agent.paw} className="card">
            <div className="flex items-start gap-4">
              <div className="p-3 bg-gray-100 rounded-lg">
                <ComputerDesktopIcon className="h-6 w-6 text-gray-600" />
              </div>
              <div className="flex-1">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold">{agent.hostname}</h3>
                  <span
                    className={`badge ${
                      agent.status === 'online' ? 'badge-success' : 'badge-danger'
                    }`}
                  >
                    {agent.status}
                  </span>
                </div>
                <p className="text-sm text-gray-500 mt-1">{agent.username}</p>
                <p className="text-sm text-gray-400 mt-1">PAW: {agent.paw.slice(0, 8)}...</p>
              </div>
            </div>

            <div className="mt-4 pt-4 border-t border-gray-100">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <p className="text-gray-500">Platform</p>
                  <p className="font-medium">{agent.platform}</p>
                </div>
                <div>
                  <p className="text-gray-500">Last Seen</p>
                  <p className="font-medium">
                    {formatDistanceToNow(new Date(agent.last_seen), { addSuffix: true })}
                  </p>
                </div>
              </div>

              <div className="mt-3">
                <p className="text-gray-500 text-sm mb-1">Executors</p>
                <div className="flex gap-1 flex-wrap">
                  {agent.executors.map((exec) => (
                    <span key={exec} className="badge bg-gray-100 text-gray-700">
                      {exec}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {agents?.length === 0 && (
        <EmptyState
          icon={ComputerDesktopIcon}
          title="No agents connected"
          description="Deploy an agent to get started"
        />
      )}
    </div>
  );
}
