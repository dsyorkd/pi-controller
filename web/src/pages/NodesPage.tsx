import React from 'react';
import NodeManagement from '../components/dashboard/NodeManagement';

const NodesPage: React.FC = () => {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Node Management</h1>
        <p className="text-gray-600 mt-2">Manage and monitor all nodes in your infrastructure</p>
      </div>

      <NodeManagement />
    </div>
  );
};

export default NodesPage;