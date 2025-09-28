import React, { useContext } from 'react';
import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { useAuthContext } from '../components/auth/AuthProvider';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card';
import { StatusBadge } from '../components/ui/StatusBadge';
import { Button } from '../components/ui/Button';

export const Dashboard: React.FC = () => {
  const authContext = (() => {
    try {
      return useAuthContext();
    } catch (error) {
      // Fallback for demo mode
      return { user: { username: 'demo-admin' } };
    }
  })();
  const { user } = authContext;

  const clusterStats = [
    { label: 'Total Clusters', value: '2', status: 'healthy' },
    { label: 'Total Nodes', value: '4', status: 'healthy' },
    { label: 'Running Pods', value: '28', status: 'healthy' },
    { label: 'Namespaces', value: '8', status: 'healthy' },
  ];

  const clusters = [
    {
      name: 'production',
      provider: 'K3s',
      version: 'v1.28.2+k3s1',
      nodes: 3,
      status: 'Active',
      cpu: '45%',
      memory: '62%',
    },
    {
      name: 'development',
      provider: 'K3s',
      version: 'v1.28.2+k3s1',
      nodes: 1,
      status: 'Active',
      cpu: '12%',
      memory: '28%',
    },
  ];

  const workloads = [
    { type: 'Deployments', count: 12, healthy: 11, unhealthy: 1 },
    { type: 'DaemonSets', count: 4, healthy: 4, unhealthy: 0 },
    { type: 'StatefulSets', count: 3, healthy: 3, unhealthy: 0 },
    { type: 'Jobs', count: 2, healthy: 2, unhealthy: 0 },
  ];

  const events = [
    { type: 'Normal', message: 'Pod scheduled successfully', time: '2 minutes ago', namespace: 'default' },
    { type: 'Normal', message: 'Deployment scaled to 3 replicas', time: '5 minutes ago', namespace: 'kube-system' },
    { type: 'Warning', message: 'Failed to pull image', time: '10 minutes ago', namespace: 'monitoring' },
  ];

  return (
    <div className="space-y-6">
      {/* Top Navigation Bar - Rancher Style */}
      <div className="flex items-center justify-between bg-white border-b border-gray-200 px-6 py-4 -mx-6 -mt-6">
        <div className="flex items-center space-x-6">
          <h1 className="text-xl font-semibold text-gray-900">Cluster Dashboard</h1>
          <div className="flex items-center space-x-4">
            <select className="text-sm border border-gray-300 rounded px-3 py-1 bg-white">
              <option>All Namespaces</option>
              <option>default</option>
              <option>kube-system</option>
              <option>monitoring</option>
            </select>
          </div>
        </div>
        <div className="flex items-center space-x-3">
          <Button size="sm" variant="secondary">Import YAML</Button>
          <Button size="sm">Deploy</Button>
        </div>
      </div>

      {/* Cluster Overview Stats - Bento Grid Style */}
      <div className="grid grid-cols-4 gap-4">
        {clusterStats.map((stat, index) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            className="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-sm transition-shadow"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">{stat.label}</p>
                <p className="text-2xl font-bold text-gray-900">{stat.value}</p>
              </div>
              <div className={`w-3 h-3 rounded-full ${stat.status === 'healthy' ? 'bg-green-500' : 'bg-red-500'}`} />
            </div>
          </motion.div>
        ))}
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Clusters Table */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Clusters</CardTitle>
                <Button size="sm" variant="secondary">+ Create</Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-200">
                      <th className="text-left py-3 font-medium text-gray-600">Name</th>
                      <th className="text-left py-3 font-medium text-gray-600">Provider</th>
                      <th className="text-left py-3 font-medium text-gray-600">Version</th>
                      <th className="text-left py-3 font-medium text-gray-600">Nodes</th>
                      <th className="text-left py-3 font-medium text-gray-600">CPU</th>
                      <th className="text-left py-3 font-medium text-gray-600">Memory</th>
                      <th className="text-left py-3 font-medium text-gray-600">Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {clusters.map((cluster, index) => (
                      <motion.tr
                        key={cluster.name}
                        initial={{ opacity: 0, x: -20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: index * 0.1 }}
                        className="border-b border-gray-100 hover:bg-gray-50 transition-colors"
                      >
                        <td className="py-3">
                          <Link to={`/clusters/${cluster.name}`} className="text-blue-600 hover:text-blue-800 font-medium">
                            {cluster.name}
                          </Link>
                        </td>
                        <td className="py-3 text-gray-700">{cluster.provider}</td>
                        <td className="py-3 text-gray-700 font-mono text-xs">{cluster.version}</td>
                        <td className="py-3 text-gray-700">{cluster.nodes}</td>
                        <td className="py-3">
                          <div className="flex items-center space-x-2">
                            <div className="w-12 bg-gray-200 rounded-full h-2">
                              <div
                                className="bg-blue-500 h-2 rounded-full"
                                style={{ width: cluster.cpu }}
                              />
                            </div>
                            <span className="text-xs text-gray-600">{cluster.cpu}</span>
                          </div>
                        </td>
                        <td className="py-3">
                          <div className="flex items-center space-x-2">
                            <div className="w-12 bg-gray-200 rounded-full h-2">
                              <div
                                className="bg-purple-500 h-2 rounded-full"
                                style={{ width: cluster.memory }}
                              />
                            </div>
                            <span className="text-xs text-gray-600">{cluster.memory}</span>
                          </div>
                        </td>
                        <td className="py-3">
                          <StatusBadge status="online" showDot={true}>
                            {cluster.status}
                          </StatusBadge>
                        </td>
                      </motion.tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Workloads Summary */}
        <div>
          <Card>
            <CardHeader>
              <CardTitle>Workloads</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {workloads.map((workload, index) => (
                  <motion.div
                    key={workload.type}
                    initial={{ opacity: 0, x: 20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.1 }}
                    className="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
                  >
                    <div>
                      <p className="font-medium text-gray-900">{workload.type}</p>
                      <div className="flex items-center space-x-2 text-sm">
                        <span className="text-green-600">{workload.healthy} healthy</span>
                        {workload.unhealthy > 0 && (
                          <span className="text-red-600">{workload.unhealthy} unhealthy</span>
                        )}
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-lg font-bold text-gray-900">{workload.count}</p>
                    </div>
                  </motion.div>
                ))}
              </div>
              <div className="mt-4">
                <Link to="/workloads">
                  <Button variant="secondary" size="sm" className="w-full">
                    View All Workloads
                  </Button>
                </Link>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Events Log */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Events</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {events.map((event, index) => (
              <motion.div
                key={index}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.1 }}
                className="flex items-start space-x-3 p-3 rounded-lg bg-gray-50 hover:bg-gray-100 transition-colors"
              >
                <div className={`w-2 h-2 rounded-full mt-2 ${
                  event.type === 'Normal' ? 'bg-green-500' :
                  event.type === 'Warning' ? 'bg-yellow-500' : 'bg-red-500'
                }`} />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-900">{event.message}</p>
                  <div className="flex items-center space-x-4 mt-1">
                    <span className="text-xs text-gray-500">{event.time}</span>
                    <span className="text-xs text-blue-600">{event.namespace}</span>
                  </div>
                </div>
              </motion.div>
            ))}
          </div>
          <div className="mt-4 text-center">
            <Button variant="secondary" size="sm">
              View All Events
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};