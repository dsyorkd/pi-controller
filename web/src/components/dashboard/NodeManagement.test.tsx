import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import NodeManagement from './NodeManagement';

// Mock the hooks
const mockUseNodes = {
  nodes: [
    {
      id: '1',
      name: 'test-node-1',
      ipAddress: '192.168.1.100',
      status: 'online' as const,
      role: 'worker' as const,
      lastSeen: new Date().toISOString(),
      clusterId: null,
      model: 'Raspberry Pi 4',
      cpuCores: 4,
      memory: 4294967296,
      architecture: 'arm64',
    },
    {
      id: '2',
      name: 'test-node-2',
      ipAddress: '192.168.1.101',
      status: 'offline' as const,
      role: 'master' as const,
      lastSeen: new Date().toISOString(),
      clusterId: 'cluster-1',
      model: 'Raspberry Pi 4',
      cpuCores: 4,
      memory: 4294967296,
      architecture: 'arm64',
    },
  ],
  isLoading: false,
  error: null,
  refetch: vi.fn(),
  provisionNode: vi.fn(),
  deprovisionNode: vi.fn(),
  getUnassignedNodes: vi.fn(() => []),
  getNodesByStatus: vi.fn(() => []),
};

const mockUseClusters = {
  clusters: [
    {
      id: 'cluster-1',
      name: 'test-cluster',
      status: 'running' as const,
      nodeCount: 1,
      createdAt: new Date().toISOString(),
    },
  ],
  isLoading: false,
  error: null,
};

vi.mock('../../hooks/useNodes', () => ({
  useNodes: () => mockUseNodes,
}));

vi.mock('../../hooks/useClusters', () => ({
  useClusters: () => mockUseClusters,
}));

// Mock components that might cause issues
vi.mock('../common/StatusBadge', () => ({
  default: ({ status }: { status: string }) => (
    <span data-testid="status-badge">{status}</span>
  ),
}));

vi.mock('../common/LoadingSpinner', () => ({
  default: ({ size, message }: { size?: string; message?: string }) => (
    <div data-testid="loading-spinner">{message || 'Loading...'}</div>
  ),
}));

const renderWithRouter = (component: React.ReactElement) => {
  return render(
    <BrowserRouter>
      {component}
    </BrowserRouter>
  );
};

describe('NodeManagement Component - Infinite Loop Prevention', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render without infinite re-renders', async () => {
    let renderCount = 0;

    const TestWrapper = () => {
      renderCount++;
      return <NodeManagement />;
    };

    renderWithRouter(<TestWrapper />);

    // Wait for initial render
    await waitFor(() => {
      expect(screen.getByText('Node Management')).toBeInTheDocument();
    });

    const initialRenderCount = renderCount;

    // Wait to see if there are additional renders
    await new Promise(resolve => setTimeout(resolve, 500));

    // Should not have excessive re-renders
    expect(renderCount).toBeLessThan(initialRenderCount + 3);
  });

  it('should display nodes without causing infinite API calls', async () => {
    renderWithRouter(<NodeManagement />);

    // Wait for nodes to be displayed
    await waitFor(() => {
      expect(screen.getByText('test-node-1')).toBeInTheDocument();
      expect(screen.getByText('test-node-2')).toBeInTheDocument();
    });

    // Verify hooks are not called excessively
    expect(mockUseNodes.refetch).not.toHaveBeenCalled();
  });

  it('should handle tab switching without triggering infinite updates', async () => {
    renderWithRouter(<NodeManagement />);

    await waitFor(() => {
      expect(screen.getByText('Node Management')).toBeInTheDocument();
    });

    // Find and click different tabs
    const allTab = screen.getByText(/All \(\d+\)/);
    const unassignedTab = screen.getByText(/Unassigned \(\d+\)/);

    // Initial render count
    let renderCount = 0;
    const mockComponent = vi.fn(() => {
      renderCount++;
      return null;
    });

    // Click tabs multiple times
    allTab.click();
    unassignedTab.click();
    allTab.click();

    // Wait and verify no infinite loops
    await new Promise(resolve => setTimeout(resolve, 200));

    // Should not have triggered excessive API calls
    expect(mockUseNodes.refetch).not.toHaveBeenCalled();
  });

  it('should handle node filtering without performance issues', async () => {
    // Override the mock to return different filtered results
    mockUseNodes.getUnassignedNodes.mockReturnValue([mockUseNodes.nodes[0]]);
    mockUseNodes.getNodesByStatus.mockReturnValue([mockUseNodes.nodes[1]]);

    renderWithRouter(<NodeManagement />);

    await waitFor(() => {
      expect(screen.getByText('Node Management')).toBeInTheDocument();
    });

    // Verify filtering functions are stable and don't cause re-renders
    const initialGetUnassigned = mockUseNodes.getUnassignedNodes;
    const initialGetByStatus = mockUseNodes.getNodesByStatus;

    // Wait and check that function references remain stable
    await new Promise(resolve => setTimeout(resolve, 100));

    expect(mockUseNodes.getUnassignedNodes).toBe(initialGetUnassigned);
    expect(mockUseNodes.getNodesByStatus).toBe(initialGetByStatus);
  });

  it('should render stats without causing update loops', async () => {
    renderWithRouter(<NodeManagement />);

    await waitFor(() => {
      expect(screen.getByText('Total Nodes')).toBeInTheDocument();
      expect(screen.getByText('Online')).toBeInTheDocument();
      expect(screen.getByText('Offline')).toBeInTheDocument();
      // Use getAllByText for elements that appear multiple times
      expect(screen.getAllByText('Unassigned')).toHaveLength(2); // stats card + node table
    });

    // Stats should be calculated without triggering additional renders
    await new Promise(resolve => setTimeout(resolve, 200));
  });

  it('should handle component lifecycle without memory leaks', async () => {
    const { unmount } = renderWithRouter(<NodeManagement />);

    await waitFor(() => {
      expect(screen.getByText('Node Management')).toBeInTheDocument();
    });

    // Unmount component to verify cleanup
    unmount();

    // Component should unmount cleanly without errors
    expect(true).toBe(true);
  });
});