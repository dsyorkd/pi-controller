import React from 'react';
import { motion } from 'framer-motion';
import { LucideIcon } from 'lucide-react';
import { cn } from '../../utils/cn';
import { Card, CardContent } from './Card';

interface StatCardProps {
  title: string;
  value: string | number;
  icon: LucideIcon;
  description?: string;
  trend?: {
    value: number;
    label: string;
    isPositive: boolean;
  };
  className?: string;
  animate?: boolean;
  delay?: number;
}

export const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  icon: Icon,
  description,
  trend,
  className,
  animate = true,
  delay = 0,
}) => {
  return (
    <Card
      className={cn('relative overflow-hidden', className)}
      animate={animate}
      delay={delay}
    >
      <CardContent className="p-6">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <p className="text-sm font-medium text-gray-600 mb-1">{title}</p>
            <div className="flex items-baseline gap-2">
              <h3 className="text-2xl font-bold text-gray-900">{value}</h3>
              {trend && (
                <motion.span
                  initial={{ opacity: 0, x: -10 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: delay + 0.2 }}
                  className={cn(
                    'text-xs font-medium px-2 py-1 rounded-full',
                    trend.isPositive
                      ? 'text-green-700 bg-green-100'
                      : 'text-red-700 bg-red-100'
                  )}
                >
                  {trend.isPositive ? '+' : ''}{trend.value}% {trend.label}
                </motion.span>
              )}
            </div>
            {description && (
              <p className="text-sm text-gray-500 mt-1">{description}</p>
            )}
          </div>

          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: delay + 0.1, type: 'spring', stiffness: 200 }}
            className="flex-shrink-0"
          >
            <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-lg flex items-center justify-center">
              <Icon className="w-6 h-6 text-white" />
            </div>
          </motion.div>
        </div>

        {/* Decorative background pattern */}
        <div className="absolute -top-2 -right-2 w-20 h-20 bg-gradient-to-br from-blue-500/10 to-cyan-600/10 rounded-full blur-xl" />
        <div className="absolute -bottom-4 -left-4 w-16 h-16 bg-gradient-to-br from-purple-500/10 to-pink-600/10 rounded-full blur-xl" />
      </CardContent>
    </Card>
  );
};