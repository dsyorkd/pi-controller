import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useNodes } from '../../hooks/useNodes';
import { useClusters } from '../../hooks/useClusters';
import StatusBadge from '../common/StatusBadge';
import LoadingSpinner from '../common/LoadingSpinner';
import type { Node } from '../../types';

interface NodeTableProps {
  nodes: Node[];
  onProvision: (nodeId: string, clusterId: string) => void;
  onDeprovision: (nodeId: string) => void;
  isLoading?: boolean;
}

const NodeTable: React.FC<NodeTableProps> = ({ nodes, onProvision, onDeprovision, isLoading = false }) => {
  const { clusters } = useClusters();
  const [provisioningNode, setProvisioningNode] = useState<string | null>(null);
  const [selectedClusterForProvision, setSelectedClusterForProvision] = useState<{ [key: string]: string }>({});

  const handleProvision = async (nodeId: string) => {
    const clusterId = selectedClusterForProvision[nodeId];
    if (!clusterId) return;

    setProvisioningNode(nodeId);
    try {
      await onProvision(nodeId, clusterId);
      setSelectedClusterForProvision((prev) => ({ ...prev, [nodeId]: '' }));
    } finally {
      setProvisioningNode(null);
    }
  };

  const handleDeprovision = async (nodeId: string) => {
    setProvisioningNode(nodeId);
    try {
      await onDeprovision(nodeId);
    } finally {
      setProvisioningNode(null);
    }
  };

  if (nodes.length === 0) {
    return (
      <div className="text-center py-8">
        <p className="text-gray-500">No nodes found</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Node
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Role
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Cluster
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Hardware
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Last Seen
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-gray-200">
          {nodes.map((node) => {
            const assignedCluster = clusters.find((c) => c.id === node.clusterId);
            const isNodeLoading = provisioningNode === node.id || isLoading;

            return (
              <tr key={node.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap">
                  <div>
                    <Link
                      to={`/nodes/${node.id}`}
                      className="text-sm font-medium text-blue-600 hover:text-blue-900"
                    >
                      {node.name}
                    </Link>
                    <div className="text-sm text-gray-500">{node.ipAddress}</div>
                    <div className="text-xs text-gray-400">{node.id}</div>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <StatusBadge status={node.status} size="small" />
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 capitalize">
                  {node.role}
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  {assignedCluster ? (
                    <Link
                      to={`/clusters/${assignedCluster.id}`}
                      className="text-sm text-blue-600 hover:text-blue-900"
                    >
                      {assignedCluster.name}
                    </Link>
                  ) : (
                    <span className="text-sm text-gray-500">Unassigned</span>
                  )}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                  <div>
                    {node.model && <div>{node.model}</div>}
                    {node.cpuCores && node.memory && (
                      <div className="text-xs text-gray-500">
                        {node.cpuCores} cores, {Math.round(node.memory / 1024 / 1024)}MB
                      </div>
                    )}
                    {node.architecture && (
                      <div className="text-xs text-gray-500">{node.architecture}</div>
                    )}
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {new Date(node.lastSeen).toLocaleString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                  {isNodeLoading ? (
                    <LoadingSpinner size="small" />
                  ) : !node.clusterId ? (
                    <div className="flex items-center space-x-2">
                      <select
                        value={selectedClusterForProvision[node.id] || ''}
                        onChange={(e) =>
                          setSelectedClusterForProvision((prev) => ({
                            ...prev,
                            [node.id]: e.target.value,
                          }))
                        }
                        className="text-sm border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                      >
                        <option value="">Select cluster...</option>
                        {clusters.map((cluster) => (
                          <option key={cluster.id} value={cluster.id}>
                            {cluster.name}
                          </option>
                        ))}
                      </select>
                      <button
                        onClick={() => handleProvision(node.id)}
                        disabled={!selectedClusterForProvision[node.id]}
                        className="text-blue-600 hover:text-blue-900 disabled:text-gray-400"
                      >
                        Provision
                      </button>
                    </div>
                  ) : (
                    <button
                      onClick={() => handleDeprovision(node.id)}
                      className="text-red-600 hover:text-red-900"
                    >
                      Deprovision
                    </button>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

const NodeManagement: React.FC = () => {
  const {
    nodes,
    isLoading,
    error,
    refetch,
    provisionNode,
    deprovisionNode,
    getUnassignedNodes,
    getNodesByStatus,
  } = useNodes();

  const [activeTab, setActiveTab] = useState<'all' | 'assigned' | 'unassigned' | 'offline'>('all');

  const getFilteredNodes = () => {
    switch (activeTab) {
      case 'assigned':
        return nodes.filter((node) => node.clusterId);
      case 'unassigned':
        return getUnassignedNodes();
      case 'offline':
        return getNodesByStatus('offline');
      default:
        return nodes;
    }
  };

  const filteredNodes = getFilteredNodes();
  const stats = {
    total: nodes.length,
    online: getNodesByStatus('online').length,
    offline: getNodesByStatus('offline').length,
    unassigned: getUnassignedNodes().length,
  };

  if (isLoading && nodes.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <LoadingSpinner size="large" message="Loading nodes..." />
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <div className="flex">
          <div className="ml-3">
            <h3 className="text-sm font-medium text-red-800">Error loading nodes</h3>
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">Node Management</h2>
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
            to="/nodes/new"
            className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            <svg className="-ml-1 mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            Add Node
          </Link>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="p-5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <div className="w-8 h-8 bg-blue-100 rounded-md flex items-center justify-center">
                  <svg className="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                  </svg>
                </div>
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-gray-500 truncate">Total Nodes</dt>
                  <dd className="text-lg font-medium text-gray-900">{stats.total}</dd>
                </dl>
              </div>
            </div>
          </div>
        </div>
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="p-5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <div className="w-8 h-8 bg-green-100 rounded-md flex items-center justify-center">
                  <div className="w-3 h-3 bg-green-400 rounded-full"></div>
                </div>
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-gray-500 truncate">Online</dt>
                  <dd className="text-lg font-medium text-gray-900">{stats.online}</dd>
                </dl>
              </div>
            </div>
          </div>
        </div>
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="p-5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <div className="w-8 h-8 bg-red-100 rounded-md flex items-center justify-center">
                  <div className="w-3 h-3 bg-red-400 rounded-full"></div>
                </div>
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-gray-500 truncate">Offline</dt>
                  <dd className="text-lg font-medium text-gray-900">{stats.offline}</dd>
                </dl>
              </div>
            </div>
          </div>
        </div>
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="p-5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <div className="w-8 h-8 bg-yellow-100 rounded-md flex items-center justify-center">
                  <svg className="w-5 h-5 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-gray-500 truncate">Unassigned</dt>
                  <dd className="text-lg font-medium text-gray-900">{stats.unassigned}</dd>
                </dl>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Filter tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8" aria-label="Tabs">
          {[
            { key: 'all', label: `All (${stats.total})` },
            { key: 'assigned', label: `Assigned (${stats.total - stats.unassigned})` },
            { key: 'unassigned', label: `Unassigned (${stats.unassigned})` },
            { key: 'offline', label: `Offline (${stats.offline})` },
          ].map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key as typeof activeTab)}
              className={`whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.key
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Nodes table */}
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <NodeTable
          nodes={filteredNodes}
          onProvision={provisionNode}
          onDeprovision={deprovisionNode}
          isLoading={isLoading}
        />
      </div>
    </div>
  );
};

export default NodeManagement;