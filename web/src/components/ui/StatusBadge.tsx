import React from 'react';
import { motion } from 'framer-motion';
import { cn } from '../../utils/cn';

interface StatusBadgeProps {
  status: 'online' | 'offline' | 'warning' | 'error';
  children: React.ReactNode;
  showDot?: boolean;
  animate?: boolean;
  className?: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({
  status,
  children,
  showDot = true,
  animate = true,
  className,
}) => {
  const statusClasses = {
    online: 'status-online',
    offline: 'status-offline',
    warning: 'status-warning',
    error: 'status-error',
  };

  const StatusComponent = animate ? motion.div : 'div';

  const animationProps = animate
    ? {
        initial: { scale: 0.8, opacity: 0 },
        animate: { scale: 1, opacity: 1 },
        transition: { duration: 0.3 },
      }
    : {};

  return (
    <StatusComponent
      className={cn('flex items-center gap-2', className)}
      {...animationProps}
    >
      {showDot && (
        <div
          className={cn(
            'w-2 h-2 rounded-full',
            statusClasses[status]
          )}
        />
      )}
      <span className={cn('text-sm font-medium', statusClasses[status])}>
        {children}
      </span>
    </StatusComponent>
  );
};