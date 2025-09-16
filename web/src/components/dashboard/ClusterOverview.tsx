import React from 'react';
import { Link } from 'react-router-dom';
import { useClusters } from '../../hooks/useClusters';
import { useNodes } from '../../hooks/useNodes';
import StatusBadge from '../common/StatusBadge';
import LoadingSpinner from '../common/LoadingSpinner';
import type { Cluster } from '../../types';

const ClusterCard: React.FC<{
  cluster: Cluster;
  nodeCount: number;
  onlineNodes: number;
}> = ({ cluster, nodeCount, onlineNodes }) => {
  return (
    <Link
      to={`/clusters/${cluster.id}`}
      className="block bg-white rounded-lg shadow-md hover:shadow-lg transition-shadow duration-200 border border-gray-200"
    >
      <div className="p-6">
        <div className="flex items-start justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 mb-1">{cluster.name}</h3>
            <p className="text-sm text-gray-600">ID: {cluster.id}</p>
          </div>
          <StatusBadge status={cluster.status} />
        </div>

        <div className="grid grid-cols-2 gap-4 mb-4">
          <div>
            <p className="text-sm text-gray-500">Total Nodes</p>
            <p className="text-2xl font-bold text-gray-900">{nodeCount}</p>
          </div>
          <div>
            <p className="text-sm text-gray-500">Online Nodes</p>
            <p className="text-2xl font-bold text-green-600">{onlineNodes}</p>
          </div>
        </div>

        <div className="border-t border-gray-200 pt-4">
          <div className="flex justify-between text-sm text-gray-500">
            <span>Created: {new Date(cluster.createdAt).toLocaleDateString()}</span>
            <span>Updated: {new Date(cluster.updatedAt).toLocaleDateString()}</span>
          </div>
        </div>
      </div>
    </Link>
  );
};

const ClusterOverview: React.FC = () => {
  const { clusters, isLoading, error, refetch } = useClusters();
  const { getNodesByCluster } = useNodes();

  if (isLoading && clusters.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <LoadingSpinner size="large" message="Loading clusters..." />
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <div className="flex">
          <div className="ml-3">
            <h3 className="text-sm font-medium text-red-800">Error loading clusters</h3>
            <p className="text-sm text-red-600 mt-1">{error}</p>
            <button
              onClick={refetch}
              className="mt-2 text-sm text-red-600 underline hover:text-red-800"
            >
              Try again
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (clusters.length === 0) {
    return (
      <div className="text-center py-12">
        <div className="mx-auto w-24 h-24 bg-gray-100 rounded-full flex items-center justify-center mb-4">
          <svg className="w-12 h-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
            />
          </svg>
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-2">No clusters found</h3>
        <p className="text-gray-600 mb-4">Create your first cluster to get started.</p>
        <Link
          to="/clusters/new"
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          Create Cluster
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">Clusters</h2>
        <div className="flex space-x-3">
          <button
            onClick={refetch}
            className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            <svg
              className="-ml-0.5 mr-2 h-4 w-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
              />
            </svg>
            Refresh
          </button>
          <Link
            to="/clusters/new"
            className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            <svg className="-ml-1 mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            New Cluster
          </Link>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {clusters.map((cluster) => {
          const clusterNodes = getNodesByCluster(cluster.id);
          const onlineNodes = clusterNodes.filter((node) => node.status === 'online').length;

          return (
            <ClusterCard
              key={cluster.id}
              cluster={cluster}
              nodeCount={clusterNodes.length}
              onlineNodes={onlineNodes}
            />
          );
        })}
      </div>
    </div>
  );
};

export default ClusterOverview;