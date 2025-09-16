import React from 'react';

interface StatusBadgeProps {
  status: 'active' | 'inactive' | 'error' | 'online' | 'offline' | 'provisioning';
  size?: 'small' | 'medium' | 'large';
}

const StatusBadge: React.FC<StatusBadgeProps> = ({ status, size = 'medium' }) => {
  const getStatusStyles = () => {
    switch (status) {
      case 'active':
      case 'online':
        return 'bg-green-100 text-green-800 border-green-300';
      case 'inactive':
      case 'offline':
        return 'bg-gray-100 text-gray-800 border-gray-300';
      case 'provisioning':
        return 'bg-yellow-100 text-yellow-800 border-yellow-300';
      case 'error':
        return 'bg-red-100 text-red-800 border-red-300';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-300';
    }
  };

  const getSizeStyles = () => {
    switch (size) {
      case 'small':
        return 'px-2 py-1 text-xs';
      case 'medium':
        return 'px-3 py-1 text-sm';
      case 'large':
        return 'px-4 py-2 text-base';
      default:
        return 'px-3 py-1 text-sm';
    }
  };

  const getStatusText = () => {
    return status.charAt(0).toUpperCase() + status.slice(1);
  };

  return (
    <span
      className={`inline-flex items-center border rounded-full font-medium ${getStatusStyles()} ${getSizeStyles()}`}
    >
      <span
        className={`w-2 h-2 rounded-full mr-2 ${
          status === 'active' || status === 'online'
            ? 'bg-green-400'
            : status === 'provisioning'
            ? 'bg-yellow-400'
            : status === 'error'
            ? 'bg-red-400'
            : 'bg-gray-400'
        }`}
      />
      {getStatusText()}
    </span>
  );
};

export default StatusBadge;