import { renderHook, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useClusters } from './useClusters';
import { apiService } from '../services/api';

// Mock the API service
vi.mock('../services/api', () => ({
  apiService: {
    clusters: {
      getAll: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
    },
  },
}));

// Mock the store
vi.mock('../store/useAppStore', () => ({
  useAppStore: vi.fn((selector) => {
    const mockState = {
      clusters: [],
      isLoading: false,
      error: null,
      setLoading: vi.fn(),
      setError: vi.fn(),
      setClusters: vi.fn(),
      addCluster: vi.fn(),
      updateCluster: vi.fn(),
      removeCluster: vi.fn(),
      selectedCluster: null,
      setSelectedCluster: vi.fn(),
    };
    return selector(mockState);
  }),
}));

describe('useClusters Hook - Infinite Loop Prevention', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Mock successful API response
    vi.mocked(apiService.clusters.getAll).mockResolvedValue({
      data: [
        {
          id: '1',
          name: 'test-cluster',
          status: 'running',
          nodeCount: 3,
          createdAt: new Date().toISOString(),
        },
      ],
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should only call fetchClusters once on initial mount', async () => {
    const { result } = renderHook(() => useClusters());

    // Wait for any effects to complete
    await waitFor(() => {
      // The hook should be available
      expect(result.current).toBeDefined();
    });

    // fetchClusters should only be called once for initial fetch
    expect(apiService.clusters.getAll).toHaveBeenCalledTimes(1);
  });

  it('should not create infinite re-renders when dependencies change', async () => {
    let renderCount = 0;
    const { result, rerender } = renderHook(() => {
      renderCount++;
      return useClusters();
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
    const { result } = renderHook(() => useClusters());

    // Verify hook returns expected structure
    expect(result.current).toHaveProperty('clusters');
    expect(result.current).toHaveProperty('isLoading');
    expect(result.current).toHaveProperty('error');
    expect(result.current).toHaveProperty('refetch');
    expect(result.current).toHaveProperty('createCluster');
    expect(result.current).toHaveProperty('deleteCluster');
    expect(result.current).toHaveProperty('updateCluster');
    expect(result.current).toHaveProperty('selectCluster');

    // Verify functions are indeed functions
    expect(typeof result.current.refetch).toBe('function');
    expect(typeof result.current.createCluster).toBe('function');
    expect(typeof result.current.deleteCluster).toBe('function');
    expect(typeof result.current.updateCluster).toBe('function');
    expect(typeof result.current.selectCluster).toBe('function');
  });

  it('should handle refetch without causing infinite loops', async () => {
    const { result } = renderHook(() => useClusters());

    await waitFor(() => {
      expect(result.current).toBeDefined();
    });

    // Clear the mock call count from initial mount
    vi.mocked(apiService.clusters.getAll).mockClear();

    // Call refetch multiple times
    result.current.refetch();
    result.current.refetch();
    result.current.refetch();

    // Wait for all calls to complete
    await waitFor(() => {
      // Should have called the API for each refetch
      expect(apiService.clusters.getAll).toHaveBeenCalledTimes(3);
    });

    // Wait a bit more to ensure no additional calls
    await new Promise(resolve => setTimeout(resolve, 100));
    expect(apiService.clusters.getAll).toHaveBeenCalledTimes(3);
  });

  it('should not fetch if clusters already exist and no error state', async () => {
    // Mock store with existing clusters
    const { useAppStore } = await import('../store/useAppStore');
    vi.mocked(useAppStore).mockImplementation((selector) => {
      const mockState = {
        clusters: [{ id: '1', name: 'existing-cluster' }], // Existing clusters
        isLoading: false,
        error: null,
        setLoading: vi.fn(),
        setError: vi.fn(),
        setClusters: vi.fn(),
        addCluster: vi.fn(),
        updateCluster: vi.fn(),
        removeCluster: vi.fn(),
        selectedCluster: null,
        setSelectedCluster: vi.fn(),
      };
      return selector(mockState);
    });

    renderHook(() => useClusters());

    // Wait for potential effects
    await new Promise(resolve => setTimeout(resolve, 100));

    // Should not have called API since clusters already exist
    expect(apiService.clusters.getAll).not.toHaveBeenCalled();
  });
});