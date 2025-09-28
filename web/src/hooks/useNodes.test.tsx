import { renderHook, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useNodes } from './useNodes';
import { apiService } from '../services/api';

// Mock the API service
vi.mock('../services/api', () => ({
  apiService: {
    nodes: {
      getAll: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      provision: vi.fn(),
      deprovision: vi.fn(),
    },
  },
}));

// Mock the store
vi.mock('../store/useAppStore', () => ({
  useAppStore: vi.fn((selector) => {
    const mockState = {
      nodes: [],
      isLoading: false,
      error: null,
      setLoading: vi.fn(),
      setError: vi.fn(),
      setNodes: vi.fn(),
      addNode: vi.fn(),
      updateNode: vi.fn(),
      removeNode: vi.fn(),
      selectedNode: null,
      setSelectedNode: vi.fn(),
    };
    return selector(mockState);
  }),
}));

describe('useNodes Hook - Infinite Loop Prevention', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Mock successful API response
    vi.mocked(apiService.nodes.getAll).mockResolvedValue({
      data: [
        {
          id: '1',
          name: 'test-node',
          ipAddress: '192.168.1.100',
          status: 'online',
          role: 'worker',
          lastSeen: new Date().toISOString(),
        },
      ],
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should only call fetchNodes once on initial mount', async () => {
    const { result } = renderHook(() => useNodes());

    // Wait for any effects to complete
    await waitFor(() => {
      // The hook should be available
      expect(result.current).toBeDefined();
    });

    // fetchNodes should only be called once for initial fetch
    expect(apiService.nodes.getAll).toHaveBeenCalledTimes(1);
  });

  it('should not create infinite re-renders when dependencies change', async () => {
    let renderCount = 0;
    const { result, rerender } = renderHook(() => {
      renderCount++;
      return useNodes();
    });

    // Wait for initial render
    await waitFor(() => {
      expect(result.current).toBeDefined();
    });

    const initialRenderCount = renderCount;

    // Force a rerender
    rerender();

    // Wait a bit to see if there are additional renders
    await new Promise(resolve => setTimeout(resolve, 100));

    // Should not have excessive re-renders (allowing for a few due to React's behavior)
    expect(renderCount).toBeLessThan(initialRenderCount + 5);
  });

  it('should provide consistent hook structure', () => {
    const { result } = renderHook(() => useNodes());

    // Verify hook returns expected structure
    expect(result.current).toHaveProperty('nodes');
    expect(result.current).toHaveProperty('isLoading');
    expect(result.current).toHaveProperty('error');
    expect(result.current).toHaveProperty('refetch');
    expect(result.current).toHaveProperty('createNode');
    expect(result.current).toHaveProperty('deleteNode');
    expect(result.current).toHaveProperty('updateNode');
    expect(result.current).toHaveProperty('provisionNode');
    expect(result.current).toHaveProperty('deprovisionNode');
    expect(result.current).toHaveProperty('selectNode');
    expect(result.current).toHaveProperty('getNodesByCluster');
    expect(result.current).toHaveProperty('getUnassignedNodes');
    expect(result.current).toHaveProperty('getNodesByStatus');

    // Verify functions are indeed functions
    expect(typeof result.current.refetch).toBe('function');
    expect(typeof result.current.selectNode).toBe('function');
    expect(typeof result.current.getNodesByCluster).toBe('function');
    expect(typeof result.current.getUnassignedNodes).toBe('function');
    expect(typeof result.current.getNodesByStatus).toBe('function');
  });

  it('should handle refetch without causing infinite loops', async () => {
    const { result } = renderHook(() => useNodes());

    await waitFor(() => {
      expect(result.current).toBeDefined();
    });

    // Clear the mock call count from initial mount
    vi.mocked(apiService.nodes.getAll).mockClear();

    // Call refetch multiple times
    result.current.refetch();
    result.current.refetch();
    result.current.refetch();

    // Wait for all calls to complete
    await waitFor(() => {
      // Should have called the API for each refetch
      expect(apiService.nodes.getAll).toHaveBeenCalledTimes(3);
    });

    // Wait a bit more to ensure no additional calls
    await new Promise(resolve => setTimeout(resolve, 100));
    expect(apiService.nodes.getAll).toHaveBeenCalledTimes(3);
  });

  it('should not fetch if nodes already exist and no error state', async () => {
    // Mock store with existing nodes
    const { useAppStore } = await import('../store/useAppStore');
    vi.mocked(useAppStore).mockImplementation((selector) => {
      const mockState = {
        nodes: [{ id: '1', name: 'existing-node' }], // Existing nodes
        isLoading: false,
        error: null,
        setLoading: vi.fn(),
        setError: vi.fn(),
        setNodes: vi.fn(),
        addNode: vi.fn(),
        updateNode: vi.fn(),
        removeNode: vi.fn(),
        selectedNode: null,
        setSelectedNode: vi.fn(),
      };
      return selector(mockState);
    });

    renderHook(() => useNodes());

    // Wait for potential effects
    await new Promise(resolve => setTimeout(resolve, 100));

    // Should not have called API since nodes already exist
    expect(apiService.nodes.getAll).not.toHaveBeenCalled();
  });
});