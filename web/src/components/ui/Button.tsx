import React from 'react';
import { motion } from 'framer-motion';
import { cn } from '../../utils/cn';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  children: React.ReactNode;
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  animate?: boolean;
}

export const Button: React.FC<ButtonProps> = ({
  children,
  variant = 'primary',
  size = 'md',
  loading = false,
  animate = true,
  className,
  disabled,
  ...props
}) => {
  const baseClasses = 'inline-flex items-center justify-center gap-2 font-medium transition-all focus-ring';

  const variantClasses = {
    primary: 'btn-primary',
    secondary: 'btn-secondary',
    ghost: 'hover:bg-gray-100 text-gray-700',
    danger: 'bg-red-500 text-white hover:bg-red-600',
  };

  const sizeClasses = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-sm',
    lg: 'px-6 py-3 text-base',
  };

  const buttonClasses = cn(
    baseClasses,
    variantClasses[variant],
    sizeClasses[size],
    (disabled || loading) && 'opacity-50 cursor-not-allowed',
    className
  );

  const ButtonComponent = animate ? motion.button : 'button';

  const animationProps = animate
    ? {
        whileHover: !disabled && !loading ? { scale: 1.02 } : {},
        whileTap: !disabled && !loading ? { scale: 0.98 } : {},
        transition: { duration: 0.1 },
      }
    : {};

  return (
    <ButtonComponent
      className={buttonClasses}
      disabled={disabled || loading}
      {...animationProps}
      {...props}
    >
      {loading && (
        <div className="w-4 h-4 spinner"></div>
      )}
      {children}
    </ButtonComponent>
  );
};