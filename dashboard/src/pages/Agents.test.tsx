import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Agents from './Agents';
import { api } from '../lib/api';

// Mock navigator.clipboard
const mockWriteText = vi.fn().mockResolvedValue(undefined);
Object.defineProperty(navigator, 'clipboard', {
  value: { writeText: mockWriteText },
  writable: true,
  configurable: true,
});

// Mock the API
vi.mock('../lib/api', () => ({
  api: {
    get: vi.fn(),
  },
}));

// Mock date-fns
vi.mock('date-fns', () => ({
  formatDistanceToNow: vi.fn(() => '5 minutes ago'),
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
    <QueryClientProvider client={testQueryClient}>{ui}</QueryClientProvider>
  );
}

describe('Agents Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', () => {
    vi.mocked(api.get).mockReturnValue(new Promise(() => {}) as never);

    renderWithClient(<Agents />);
    expect(screen.getByText('Loading agents...')).toBeInTheDocument();
  });

  it('renders agents list', async () => {
    const mockAgents = [
      {
        paw: 'agent-123-456-789',
        hostname: 'DESKTOP-TEST',
        username: 'testuser',
        platform: 'windows',
        executors: ['psh', 'cmd'],
        status: 'online',
        last_seen: '2024-01-15T12:00:00Z',
      },
    ];
    vi.mocked(api.get).mockResolvedValue({ data: mockAgents } as never);

    renderWithClient(<Agents />);

    expect(await screen.findByText('DESKTOP-TEST')).toBeInTheDocument();
    expect(screen.getByText('testuser')).toBeInTheDocument();
    expect(screen.getByText('online')).toBeInTheDocument();
    expect(screen.getByText('windows')).toBeInTheDocument();
    expect(screen.getByText('psh')).toBeInTheDocument();
    expect(screen.getByText('cmd')).toBeInTheDocument();
  });

  it('renders offline agent with correct badge', async () => {
    const mockAgents = [
      {
        paw: 'agent-offline',
        hostname: 'OFFLINE-PC',
        username: 'user',
        platform: 'linux',
        executors: ['bash'],
        status: 'offline',
        last_seen: '2024-01-10T12:00:00Z',
      },
    ];
    vi.mocked(api.get).mockResolvedValue({ data: mockAgents } as never);

    renderWithClient(<Agents />);

    expect(await screen.findByText('offline')).toBeInTheDocument();
  });

  it('renders empty state when no agents', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] } as never);

    renderWithClient(<Agents />);

    expect(await screen.findByText('No agents connected')).toBeInTheDocument();
    expect(screen.getByText('Deploy an agent to get started')).toBeInTheDocument();
  });

  it('renders page title and add button', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] } as never);

    renderWithClient(<Agents />);

    expect(await screen.findByText('Agents')).toBeInTheDocument();
    expect(screen.getByText('Add Agent')).toBeInTheDocument();
  });

  it('displays truncated PAW', async () => {
    const mockAgents = [
      {
        paw: 'abcdefgh-1234-5678-9012-ijklmnopqrst',
        hostname: 'TEST-PC',
        username: 'user',
        platform: 'windows',
        executors: ['cmd'],
        status: 'online',
        last_seen: '2024-01-15T12:00:00Z',
      },
    ];
    vi.mocked(api.get).mockResolvedValue({ data: mockAgents } as never);

    renderWithClient(<Agents />);

    expect(await screen.findByText('PAW: abcdefgh...')).toBeInTheDocument();
  });

  describe('Deploy Agent Modal', () => {
    beforeEach(() => {
      vi.mocked(api.get).mockResolvedValue({ data: [] } as never);
      mockWriteText.mockClear();
    });

    it('opens modal when Add Agent button is clicked', async () => {
      renderWithClient(<Agents />);

      await screen.findByText('Add Agent');
      fireEvent.click(screen.getByText('Add Agent'));

      expect(screen.getByText('Deploy Agent')).toBeInTheDocument();
      expect(screen.getByText('Linux / macOS')).toBeInTheDocument();
      expect(screen.getByText('Windows')).toBeInTheDocument();
      expect(screen.getByText('Docker')).toBeInTheDocument();
    });

    it('displays correct deployment commands', async () => {
      renderWithClient(<Agents />);

      await screen.findByText('Add Agent');
      fireEvent.click(screen.getByText('Add Agent'));

      // Check that commands contain expected patterns
      const codeBlocks = screen.getAllByRole('code');
      expect(codeBlocks).toHaveLength(3);

      // Linux command should contain dist/autostrike-agent
      expect(codeBlocks[0].textContent).toContain('./dist/autostrike-agent');
      // Windows command should contain dist/autostrike-agent.exe
      expect(codeBlocks[1].textContent).toContain('autostrike-agent.exe');
      // Docker command
      expect(codeBlocks[2].textContent).toContain('docker run');
    });

    it('closes modal when Close button is clicked', async () => {
      renderWithClient(<Agents />);

      await screen.findByText('Add Agent');
      fireEvent.click(screen.getByText('Add Agent'));

      expect(screen.getByText('Deploy Agent')).toBeInTheDocument();

      fireEvent.click(screen.getByText('Close'));

      await waitFor(() => {
        expect(screen.queryByText('Deploy Agent')).not.toBeInTheDocument();
      });
    });

    it('closes modal when X button is clicked', async () => {
      renderWithClient(<Agents />);

      await screen.findByText('Add Agent');
      fireEvent.click(screen.getByText('Add Agent'));

      // Find the X button by its parent structure
      const closeButtons = screen.getAllByRole('button');
      const xButton = closeButtons.find(btn => btn.querySelector('svg.h-5.w-5'));
      expect(xButton).toBeDefined();
      if (xButton) {
        fireEvent.click(xButton);
      }

      await waitFor(() => {
        expect(screen.queryByText('Deploy Agent')).not.toBeInTheDocument();
      });
    });

    it('copies Linux command to clipboard', async () => {
      renderWithClient(<Agents />);

      await screen.findByText('Add Agent');
      fireEvent.click(screen.getByText('Add Agent'));

      // Find copy buttons by title
      const copyButtons = screen.getAllByTitle('Copy');
      expect(copyButtons).toHaveLength(3);

      fireEvent.click(copyButtons[0]); // Linux copy button

      await waitFor(() => {
        expect(mockWriteText).toHaveBeenCalledWith(
          expect.stringContaining('./dist/autostrike-agent')
        );
      });
    });

    it('copies Windows command to clipboard', async () => {
      renderWithClient(<Agents />);

      await screen.findByText('Add Agent');
      fireEvent.click(screen.getByText('Add Agent'));

      const copyButtons = screen.getAllByTitle('Copy');
      fireEvent.click(copyButtons[1]); // Windows copy button

      await waitFor(() => {
        expect(mockWriteText).toHaveBeenCalledWith(
          expect.stringContaining('autostrike-agent.exe')
        );
      });
    });

    it('copies Docker command to clipboard', async () => {
      renderWithClient(<Agents />);

      await screen.findByText('Add Agent');
      fireEvent.click(screen.getByText('Add Agent'));

      const copyButtons = screen.getAllByTitle('Copy');
      fireEvent.click(copyButtons[2]); // Docker copy button

      await waitFor(() => {
        expect(mockWriteText).toHaveBeenCalledWith(
          expect.stringContaining('docker run')
        );
      });
    });

    it('shows check icon after copying', async () => {
      renderWithClient(<Agents />);

      await screen.findByText('Add Agent');
      fireEvent.click(screen.getByText('Add Agent'));

      const copyButtons = screen.getAllByTitle('Copy');
      fireEvent.click(copyButtons[0]);

      // After clicking, the clipboard icon should change to check icon
      await waitFor(() => {
        const checkIcons = document.querySelectorAll('.text-green-500');
        expect(checkIcons.length).toBeGreaterThan(0);
      });
    });

    it('displays helpful text in modal', async () => {
      renderWithClient(<Agents />);

      await screen.findByText('Add Agent');
      fireEvent.click(screen.getByText('Add Agent'));

      await waitFor(() => {
        expect(
          screen.getByText('Download the agent binary for your platform and run:')
        ).toBeInTheDocument();
      });
      expect(
        screen.getByText(
          'The agent will automatically register with the server once started.'
        )
      ).toBeInTheDocument();
    });
  });
});
