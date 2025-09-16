import React, { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useNodes } from '../hooks/useNodes';
import { useClusters } from '../hooks/useClusters';
import StatusBadge from '../components/common/StatusBadge';
import LoadingSpinner from '../components/common/LoadingSpinner';
import { apiService } from '../services/api';
import type { Node, GPIODevice } from '../types';

const NodeDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { nodes, isLoading, error, deleteNode, provisionNode, deprovisionNode } = useNodes();
  const { clusters } = useClusters();

  const [node, setNode] = useState<Node | null>(null);
  const [gpioDevices, setGpioDevices] = useState<GPIODevice[]>([]);
  const [loadingNode, setLoadingNode] = useState(false);
  const [loadingGpio, setLoadingGpio] = useState(false);
  const [nodeError, setNodeError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [provisioning, setProvisioning] = useState(false);

  useEffect(() => {
    if (!id) return;

    // Try to find node in store first
    const foundNode = nodes.find((n) => n.id === id);
    if (foundNode) {
      setNode(foundNode);
      fetchGpioDevices(id);
    } else if (!isLoading) {
      // If not in store and not currently loading, fetch from API
      fetchNodeDetails(id);
    }
  }, [id, nodes, isLoading]);

  const fetchNodeDetails = async (nodeId: string) => {
    setLoadingNode(true);
    setNodeError(null);
    try {
      const nodeData = await apiService.nodes.getById(nodeId);
      setNode(nodeData);
      fetchGpioDevices(nodeId);
    } catch (err) {
      setNodeError(err instanceof Error ? err.message : 'Failed to fetch node details');
    } finally {
      setLoadingNode(false);
    }
  };

  const fetchGpioDevices = async (nodeId: string) => {
    setLoadingGpio(true);
    try {
      const devices = await apiService.nodes.getGPIO(nodeId);
      setGpioDevices(devices);
    } catch (err) {
      console.error('Failed to fetch GPIO devices:', err);
      // Don't show error for GPIO as it might not be available
    } finally {
      setLoadingGpio(false);
    }
  };

  const handleDelete = async () => {
    if (!node || !confirm(`Are you sure you want to delete the node "${node.name}"?`)) {
      return;
    }

    setDeleting(true);
    try {
      await deleteNode(node.id);
      navigate('/nodes');
    } catch (err) {
      // Error is handled in the hook
    } finally {
      setDeleting(false);
    }
  };

  const handleProvision = async (clusterId: string) => {
    if (!node) return;

    setProvisioning(true);
    try {
      await provisionNode(node.id, clusterId);
      // Update local state
      setNode({ ...node, clusterId, status: 'provisioning' });
    } catch (err) {
      // Error is handled in the hook
    } finally {
      setProvisioning(false);
    }
  };

  const handleDeprovision = async () => {
    if (!node) return;

    setProvisioning(true);
    try {
      await deprovisionNode(node.id);
      // Update local state
      setNode({ ...node, clusterId: undefined, status: 'offline' });
    } catch (err) {
      // Error is handled in the hook
    } finally {
      setProvisioning(false);
    }
  };

  const loading = isLoading || loadingNode;
  const displayError = error || nodeError;

  if (loading && !node) {
    return (
      <div className="flex items-center justify-center h-64">
        <LoadingSpinner size="large" message="Loading node details..." />
      </div>
    );
  }

  if (displayError) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <div className="flex">
          <div className="ml-3">
            <h3 className="text-sm font-medium text-red-800">Error loading node</h3>
            <p className="text-sm text-red-600 mt-1">{displayError}</p>
            <div className="mt-4">
              <Link
                to="/nodes"
                className="text-sm text-red-600 underline hover:text-red-800"
              >
                Back to Node Management
              </Link>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (!node) {
    return (
      <div className="text-center py-12">
        <h3 className="text-lg font-medium text-gray-900 mb-2">Node not found</h3>
        <p className="text-gray-600 mb-4">The node you're looking for doesn't exist.</p>
        <Link
          to="/nodes"
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
        >
          Back to Node Management
        </Link>
      </div>
    );
  }

  const assignedCluster = clusters.find((c) => c.id === node.clusterId);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-white shadow overflow-hidden sm:rounded-lg">
        <div className="px-4 py-5 sm:px-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{node.name}</h1>
              <p className="text-sm text-gray-600">Node ID: {node.id}</p>
            </div>
            <div className="flex items-center space-x-3">
              <StatusBadge status={node.status} />
              <button
                onClick={handleDelete}
                disabled={deleting}
                className="inline-flex items-center px-3 py-2 border border-red-300 shadow-sm text-sm leading-4 font-medium rounded-md text-red-700 bg-white hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
              >
                {deleting ? <LoadingSpinner size="small" /> : 'Delete Node'}
              </button>
            </div>
          </div>
        </div>

        <div className="border-t border-gray-200">
          <dl>
            <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Status</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                <StatusBadge status={node.status} />
              </dd>
            </div>
            <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Role</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2 capitalize">
                {node.role}
              </dd>
            </div>
            <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">IP Address</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {node.ipAddress}
              </dd>
            </div>
            <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">MAC Address</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {node.macAddress}
              </dd>
            </div>
            <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Cluster Assignment</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {assignedCluster ? (
                  <div className="flex items-center space-x-2">
                    <Link
                      to={`/clusters/${assignedCluster.id}`}
                      className="text-blue-600 hover:text-blue-900"
                    >
                      {assignedCluster.name}
                    </Link>
                    <button
                      onClick={handleDeprovision}
                      disabled={provisioning}
                      className="text-sm text-red-600 hover:text-red-900 disabled:opacity-50"
                    >
                      {provisioning ? <LoadingSpinner size="small" /> : 'Deprovision'}
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center space-x-2">
                    <span className="text-gray-500">Unassigned</span>
                    <select
                      onChange={(e) => e.target.value && handleProvision(e.target.value)}
                      disabled={provisioning}
                      className="text-sm border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                    >
                      <option value="">Assign to cluster...</option>
                      {clusters.map((cluster) => (
                        <option key={cluster.id} value={cluster.id}>
                          {cluster.name}
                        </option>
                      ))}
                    </select>
                  </div>
                )}
              </dd>
            </div>
            {node.architecture && (
              <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500">Architecture</dt>
                <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                  {node.architecture}
                </dd>
              </div>
            )}
            {node.model && (
              <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500">Model</dt>
                <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                  {node.model}
                </dd>
              </div>
            )}
            {node.cpuCores && (
              <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500">CPU Cores</dt>
                <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                  {node.cpuCores}
                </dd>
              </div>
            )}
            {node.memory && (
              <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500">Memory</dt>
                <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                  {Math.round(node.memory / 1024 / 1024)} MB
                </dd>
              </div>
            )}
            <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Last Seen</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {new Date(node.lastSeen).toLocaleString()}
              </dd>
            </div>
          </dl>
        </div>
      </div>

      {/* GPIO Devices */}
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <div className="px-4 py-5 sm:px-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-medium text-gray-900">GPIO Devices</h2>
              <p className="text-sm text-gray-600">
                GPIO pins and devices configured on this node
              </p>
            </div>
            {loadingGpio && <LoadingSpinner size="small" />}
          </div>
        </div>

        {gpioDevices.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-500">No GPIO devices configured</p>
            <p className="text-sm text-gray-400 mt-1">
              GPIO devices will appear here when configured through Kubernetes CRDs
            </p>
          </div>
        ) : (
          <ul role="list" className="divide-y divide-gray-200">
            {gpioDevices.map((device) => (
              <li key={device.id} className="px-4 py-4 sm:px-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-900">
                      {device.name || `GPIO Pin ${device.pin}`}
                    </p>
                    {device.description && (
                      <p className="text-sm text-gray-600">{device.description}</p>
                    )}
                    <p className="text-xs text-gray-500">
                      Pin {device.pin} • {device.direction} • ID: {device.id}
                    </p>
                  </div>
                  <div className="flex items-center space-x-4">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        device.direction === 'input'
                          ? 'bg-blue-100 text-blue-800'
                          : 'bg-green-100 text-green-800'
                      }`}
                    >
                      {device.direction}
                    </span>
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        device.value
                          ? 'bg-yellow-100 text-yellow-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}
                    >
                      {device.value ? 'HIGH' : 'LOW'}
                    </span>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Navigation */}
      <div className="flex justify-between">
        <Link
          to="/nodes"
          className="inline-flex items-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
        >
          <svg className="-ml-1 mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Back to Node Management
        </Link>
        {assignedCluster && (
          <Link
            to={`/clusters/${assignedCluster.id}`}
            className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
          >
            View Cluster
            <svg className="ml-2 -mr-1 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
            </svg>
          </Link>
        )}
      </div>
    </div>
  );
};

export default NodeDetail;