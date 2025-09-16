import { useEffect, useState, useCallback } from 'react';
import { useAppStore } from '../store/useAppStore';
import { apiService } from '../services/api';
import type { Cluster } from '../types';

export const useClusters = () => {
  const [refetching, setRefetching] = useState(false);
  const {
    clusters,
    isLoading,
    error,
    setLoading,
    setError,
    setClusters,
    addCluster,
    updateCluster,
    removeCluster,
    selectedCluster,
    setSelectedCluster,
  } = useAppStore((state) => ({
    clusters: state.clusters,
    isLoading: state.isLoading,
    error: state.error,
    setLoading: state.setLoading,
    setError: state.setError,
    setClusters: state.setClusters,
    addCluster: state.addCluster,
    updateCluster: state.updateCluster,
    removeCluster: state.removeCluster,
    selectedCluster: state.selectedCluster,
    setSelectedCluster: state.setSelectedCluster,
  }));

  const fetchClusters = useCallback(async () => {
    try {
      if (!refetching) {
        setLoading(true);
      }
      setError(null);
      const response = await apiService.clusters.getAll();
      setClusters(response.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch clusters');
    } finally {
      setLoading(false);
      setRefetching(false);
    }
  }, [refetching, setLoading, setError, setClusters]);

  const refetch = useCallback(() => {
    setRefetching(true);
    fetchClusters();
  }, [fetchClusters]);

  const createCluster = useCallback(
    async (clusterData: Partial<Cluster>) => {
      try {
        setLoading(true);
        setError(null);
        const newCluster = await apiService.clusters.create(clusterData);
        addCluster(newCluster);
        return newCluster;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to create cluster');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, addCluster]
  );

  const deleteCluster = useCallback(
    async (id: string) => {
      try {
        setLoading(true);
        setError(null);
        await apiService.clusters.delete(id);
        removeCluster(id);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to delete cluster');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, removeCluster]
  );

  const updateClusterData = useCallback(
    async (id: string, updates: Partial<Cluster>) => {
      try {
        setLoading(true);
        setError(null);
        const updatedCluster = await apiService.clusters.update(id, updates);
        updateCluster(id, updatedCluster);
        return updatedCluster;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update cluster');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, updateCluster]
  );

  const selectCluster = useCallback(
    (cluster: Cluster | null) => {
      setSelectedCluster(cluster);
    },
    [setSelectedCluster]
  );

  useEffect(() => {
    if (clusters.length === 0 && !isLoading && !error) {
      fetchClusters();
    }
  }, [clusters.length, isLoading, error, fetchClusters]);

  return {
    clusters,
    selectedCluster,
    isLoading,
    error,
    refetch,
    createCluster,
    deleteCluster,
    updateCluster: updateClusterData,
    selectCluster,
  };
};