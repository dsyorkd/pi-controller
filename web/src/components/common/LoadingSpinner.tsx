import React from 'react';

interface LoadingSpinnerProps {
  size?: 'small' | 'medium' | 'large';
  color?: string;
  message?: string;
}

const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  size = 'medium',
  color = '#3b82f6',
  message,
}) => {
  const sizeClasses = {
    small: 'w-4 h-4',
    medium: 'w-8 h-8',
    large: 'w-12 h-12',
  };

  return (
    <div className="flex flex-col items-center justify-center space-y-2">
      <div
        className={`animate-spin rounded-full border-2 border-gray-200 border-t-current ${sizeClasses[size]}`}
        style={{ color }}
      />
      {message && <p className="text-sm text-gray-600">{message}</p>}
    </div>
  );
};

export default LoadingSpinner;