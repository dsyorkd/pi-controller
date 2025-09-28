import React from 'react';
import ClusterOverview from '../components/dashboard/ClusterOverview';

const ClustersPage: React.FC = () => {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Cluster Management</h1>
        <p className="text-gray-600 mt-2">Manage and monitor all clusters in your infrastructure</p>
      </div>

      <ClusterOverview />
    </div>
  );
};

export default ClustersPage;