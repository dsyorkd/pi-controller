import React, { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useClusters } from '../hooks/useClusters';
import { useNodes } from '../hooks/useNodes';
import StatusBadge from '../components/common/StatusBadge';
import LoadingSpinner from '../components/common/LoadingSpinner';
import { apiService } from '../services/api';
import type { Cluster, Node } from '../types';

const ClusterDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { clusters, isLoading, error, deleteCluster } = useClusters();
  const { getNodesByCluster } = useNodes();

  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [clusterNodes, setClusterNodes] = useState<Node[]>([]);
  const [loadingCluster, setLoadingCluster] = useState(false);
  const [clusterError, setClusterError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (!id) return;

    // Try to find cluster in store first
    const foundCluster = clusters.find((c) => c.id === id);
    if (foundCluster) {
      setCluster(foundCluster);
      setClusterNodes(getNodesByCluster(id));
    } else if (!isLoading) {
      // If not in store and not currently loading, fetch from API
      fetchClusterDetails(id);
    }
  }, [id, clusters, isLoading, getNodesByCluster]);

  const fetchClusterDetails = async (clusterId: string) => {
    setLoadingCluster(true);
    setClusterError(null);
    try {
      const clusterData = await apiService.clusters.getById(clusterId);
      const nodesData = await apiService.clusters.getNodes(clusterId);
      setCluster(clusterData);
      setClusterNodes(nodesData);
    } catch (err) {
      setClusterError(err instanceof Error ? err.message : 'Failed to fetch cluster details');
    } finally {
      setLoadingCluster(false);
    }
  };

  const handleDelete = async () => {
    if (!cluster || !confirm(`Are you sure you want to delete the cluster "${cluster.name}"?`)) {
      return;
    }

    setDeleting(true);
    try {
      await deleteCluster(cluster.id);
      navigate('/');
    } catch (err) {
      // Error is handled in the hook
    } finally {
      setDeleting(false);
    }
  };

  const loading = isLoading || loadingCluster;
  const displayError = error || clusterError;

  if (loading && !cluster) {
    return (
      <div className="flex items-center justify-center h-64">
        <LoadingSpinner size="large" message="Loading cluster details..." />
      </div>
    );
  }

  if (displayError) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <div className="flex">
          <div className="ml-3">
            <h3 className="text-sm font-medium text-red-800">Error loading cluster</h3>
            <p className="text-sm text-red-600 mt-1">{displayError}</p>
            <div className="mt-4">
              <Link
                to="/"
                className="text-sm text-red-600 underline hover:text-red-800"
              >
                Back to Dashboard
              </Link>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (!cluster) {
    return (
      <div className="text-center py-12">
        <h3 className="text-lg font-medium text-gray-900 mb-2">Cluster not found</h3>
        <p className="text-gray-600 mb-4">The cluster you're looking for doesn't exist.</p>
        <Link
          to="/"
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
        >
          Back to Dashboard
        </Link>
      </div>
    );
  }

  const onlineNodes = clusterNodes.filter((node) => node.status === 'online').length;
  const masterNodes = clusterNodes.filter((node) => node.role === 'master');
  const workerNodes = clusterNodes.filter((node) => node.role === 'worker');

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-white shadow overflow-hidden sm:rounded-lg">
        <div className="px-4 py-5 sm:px-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{cluster.name}</h1>
              <p className="text-sm text-gray-600">Cluster ID: {cluster.id}</p>
            </div>
            <div className="flex items-center space-x-3">
              <StatusBadge status={cluster.status} />
              <button
                onClick={handleDelete}
                disabled={deleting}
                className="inline-flex items-center px-3 py-2 border border-red-300 shadow-sm text-sm leading-4 font-medium rounded-md text-red-700 bg-white hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
              >
                {deleting ? <LoadingSpinner size="small" /> : 'Delete Cluster'}
              </button>
            </div>
          </div>
        </div>

        <div className="border-t border-gray-200">
          <dl>
            <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Status</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                <StatusBadge status={cluster.status} />
              </dd>
            </div>
            <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Total Nodes</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {clusterNodes.length} ({onlineNodes} online)
              </dd>
            </div>
            <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Master Nodes</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {masterNodes.length}
              </dd>
            </div>
            <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Worker Nodes</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {workerNodes.length}
              </dd>
            </div>
            <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Created</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {new Date(cluster.createdAt).toLocaleString()}
              </dd>
            </div>
            <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Last Updated</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {new Date(cluster.updatedAt).toLocaleString()}
              </dd>
            </div>
          </dl>
        </div>
      </div>

      {/* Nodes */}
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <div className="px-4 py-5 sm:px-6 border-b border-gray-200">
          <h2 className="text-lg font-medium text-gray-900">Cluster Nodes</h2>
          <p className="text-sm text-gray-600">
            Nodes assigned to this cluster
          </p>
        </div>

        {clusterNodes.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-500">No nodes assigned to this cluster</p>
            <Link
              to="/nodes"
              className="mt-2 text-sm text-blue-600 hover:text-blue-900"
            >
              Manage Nodes
            </Link>
          </div>
        ) : (
          <ul role="list" className="divide-y divide-gray-200">
            {clusterNodes.map((node) => (
              <li key={node.id}>
                <Link
                  to={`/nodes/${node.id}`}
                  className="block hover:bg-gray-50 px-4 py-4 sm:px-6"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <div className="flex-shrink-0">
                        <div className="h-10 w-10 rounded-full bg-gray-100 flex items-center justify-center">
                          <svg className="h-6 w-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                          </svg>
                        </div>
                      </div>
                      <div className="ml-4">
                        <div className="flex items-center">
                          <p className="text-sm font-medium text-gray-900">{node.name}</p>
                          <span className="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 capitalize">
                            {node.role}
                          </span>
                        </div>
                        <p className="text-sm text-gray-600">{node.ipAddress}</p>
                        {node.model && (
                          <p className="text-xs text-gray-500">{node.model}</p>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center space-x-4">
                      <StatusBadge status={node.status} size="small" />
                      <p className="text-sm text-gray-500">
                        Last seen: {new Date(node.lastSeen).toLocaleString()}
                      </p>
                    </div>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Navigation */}
      <div className="flex justify-between">
        <Link
          to="/"
          className="inline-flex items-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
        >
          <svg className="-ml-1 mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Back to Dashboard
        </Link>
        <Link
          to="/nodes"
          className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
        >
          Manage Nodes
          <svg className="ml-2 -mr-1 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
          </svg>
        </Link>
      </div>
    </div>
  );
};

export default ClusterDetail;