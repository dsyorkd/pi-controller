import React, { type ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuthState } from '../../hooks/useAuth';

interface ProtectedRouteProps {
  children: ReactNode;
  requiredRole?: 'admin' | 'user' | 'readonly';
  redirectTo?: string;
}

/**
 * ProtectedRoute component that guards routes requiring authentication
 *
 * Features:
 * - Redirects to login if not authenticated
 * - Supports role-based access control
 * - Preserves intended destination for post-login redirect
 * - Shows loading state while checking authentication
 */
export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  children,
  requiredRole,
  redirectTo = '/login'
}) => {
  const { user, isAuthenticated, isLoading } = useAuthState();
  const location = useLocation();

  // Show loading state while authentication is being checked
  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600 mb-4"></div>
          <p className="text-gray-600">Loading...</p>
        </div>
      </div>
    );
  }

  // If not authenticated, redirect to login with current location
  if (!isAuthenticated || !user) {
    return (
      <Navigate
        to={redirectTo}
        state={{ from: location }}
        replace
      />
    );
  }

  // If role is required, check user's role
  if (requiredRole) {
    const hasRequiredRole = checkRole(user.role, requiredRole);

    if (!hasRequiredRole) {
      return (
        <div className="flex items-center justify-center min-h-screen">
          <div className="text-center">
            <div className="text-red-600 text-6xl mb-4">🚫</div>
            <h2 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h2>
            <p className="text-gray-600">
              You don't have permission to access this resource.
            </p>
            <p className="text-sm text-gray-500 mt-2">
              Required role: {requiredRole} | Your role: {user.role}
            </p>
          </div>
        </div>
      );
    }
  }

  // User is authenticated and has required role, render children
  return <>{children}</>;
};

/**
 * Role hierarchy check function
 * Admin can access everything, user can access user and readonly, readonly can only access readonly
 */
function checkRole(userRole: 'admin' | 'user' | 'readonly', requiredRole: 'admin' | 'user' | 'readonly'): boolean {
  const roleHierarchy = {
    readonly: 1,
    user: 2,
    admin: 3
  };

  return roleHierarchy[userRole] >= roleHierarchy[requiredRole];
}