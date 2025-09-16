import { useEffect, useState, useCallback } from 'react';
import { useAppStore } from '../store/useAppStore';
import { apiService } from '../services/api';
import type { Node } from '../types';

export const useNodes = () => {
  const [refetching, setRefetching] = useState(false);
  const {
    nodes,
    isLoading,
    error,
    setLoading,
    setError,
    setNodes,
    addNode,
    updateNode,
    removeNode,
    selectedNode,
    setSelectedNode,
  } = useAppStore((state) => ({
    nodes: state.nodes,
    isLoading: state.isLoading,
    error: state.error,
    setLoading: state.setLoading,
    setError: state.setError,
    setNodes: state.setNodes,
    addNode: state.addNode,
    updateNode: state.updateNode,
    removeNode: state.removeNode,
    selectedNode: state.selectedNode,
    setSelectedNode: state.setSelectedNode,
  }));

  const fetchNodes = useCallback(async () => {
    try {
      if (!refetching) {
        setLoading(true);
      }
      setError(null);
      const response = await apiService.nodes.getAll();
      setNodes(response.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch nodes');
    } finally {
      setLoading(false);
      setRefetching(false);
    }
  }, [refetching, setLoading, setError, setNodes]);

  const refetch = useCallback(() => {
    setRefetching(true);
    fetchNodes();
  }, [fetchNodes]);

  const createNode = useCallback(
    async (nodeData: Partial<Node>) => {
      try {
        setLoading(true);
        setError(null);
        const newNode = await apiService.nodes.create(nodeData);
        addNode(newNode);
        return newNode;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to create node');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, addNode]
  );

  const deleteNode = useCallback(
    async (id: string) => {
      try {
        setLoading(true);
        setError(null);
        await apiService.nodes.delete(id);
        removeNode(id);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to delete node');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, removeNode]
  );

  const updateNodeData = useCallback(
    async (id: string, updates: Partial<Node>) => {
      try {
        setLoading(true);
        setError(null);
        const updatedNode = await apiService.nodes.update(id, updates);
        updateNode(id, updatedNode);
        return updatedNode;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update node');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, updateNode]
  );

  const provisionNode = useCallback(
    async (nodeId: string, clusterId: string) => {
      try {
        setLoading(true);
        setError(null);
        await apiService.nodes.provision(nodeId, clusterId);
        // Update the node's status and cluster assignment
        updateNode(nodeId, { clusterId, status: 'provisioning' });
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to provision node');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, updateNode]
  );

  const deprovisionNode = useCallback(
    async (nodeId: string) => {
      try {
        setLoading(true);
        setError(null);
        await apiService.nodes.deprovision(nodeId);
        // Update the node's status and remove cluster assignment
        updateNode(nodeId, { clusterId: undefined, status: 'offline' });
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to deprovision node');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, updateNode]
  );

  const selectNode = useCallback(
    (node: Node | null) => {
      setSelectedNode(node);
    },
    [setSelectedNode]
  );

  // Filter helpers
  const getNodesByCluster = useCallback(
    (clusterId: string) => {
      return nodes.filter((node) => node.clusterId === clusterId);
    },
    [nodes]
  );

  const getUnassignedNodes = useCallback(() => {
    return nodes.filter((node) => !node.clusterId);
  }, [nodes]);

  const getNodesByStatus = useCallback(
    (status: Node['status']) => {
      return nodes.filter((node) => node.status === status);
    },
    [nodes]
  );

  useEffect(() => {
    if (nodes.length === 0 && !isLoading && !error) {
      fetchNodes();
    }
  }, [nodes.length, isLoading, error, fetchNodes]);

  return {
    nodes,
    selectedNode,
    isLoading,
    error,
    refetch,
    createNode,
    deleteNode,
    updateNode: updateNodeData,
    provisionNode,
    deprovisionNode,
    selectNode,
    getNodesByCluster,
    getUnassignedNodes,
    getNodesByStatus,
  };
};