import React from 'react';
import { Link } from 'react-router-dom';
import { useAuthContext } from './AuthProvider';

export const UserProfile: React.FC = () => {
  const { user, logout } = useAuthContext();

  if (!user) {
    return (
      <div className="min-h-full flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Not Authenticated</h2>
          <p className="text-gray-600 mb-8">Please log in to view your profile.</p>
          <Link
            to="/login"
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
          >
            Go to Login
          </Link>
        </div>
      </div>
    );
  }

  const handleLogout = async () => {
    if (window.confirm('Are you sure you want to log out?')) {
      await logout();
    }
  };

  const getRoleColor = (role: string) => {
    switch (role) {
      case 'admin':
        return 'bg-red-100 text-red-800';
      case 'user':
        return 'bg-blue-100 text-blue-800';
      case 'readonly':
        return 'bg-gray-100 text-gray-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const getRoleDescription = (role: string) => {
    switch (role) {
      case 'admin':
        return 'Full system administration access including user management, system settings, and all cluster operations.';
      case 'user':
        return 'Standard user access with ability to manage clusters, nodes, and GPIO devices.';
      case 'readonly':
        return 'Read-only access to view system status and monitoring data.';
      default:
        return 'Standard user access.';
    }
  };

  return (
    <div className="max-w-4xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
      <div className="bg-white shadow rounded-lg">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200">
          <h1 className="text-2xl font-bold text-gray-900">User Profile</h1>
          <p className="text-gray-600">Manage your account information and settings</p>
        </div>

        {/* Profile Content */}
        <div className="px-6 py-6">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* User Info Card */}
            <div className="lg:col-span-2">
              <div className="bg-gray-50 rounded-lg p-6">
                <div className="flex items-start space-x-4">
                  <div className="w-20 h-20 bg-blue-600 rounded-full flex items-center justify-center text-white text-3xl font-bold">
                    {user.username.charAt(0).toUpperCase()}
                  </div>
                  <div className="flex-1">
                    <h3 className="text-xl font-semibold text-gray-900 mb-1">{user.username}</h3>
                    <p className="text-gray-600 mb-3">{user.email}</p>
                    <div className="flex items-center space-x-2">
                      <span className={`inline-flex px-3 py-1 text-sm font-medium rounded-full ${getRoleColor(user.role)}`}>
                        {user.role.charAt(0).toUpperCase() + user.role.slice(1)}
                      </span>
                      <span className="text-xs text-gray-500">
                        ID: {user.id}
                      </span>
                    </div>
                  </div>
                </div>

                <div className="mt-6 pt-6 border-t border-gray-200">
                  <h4 className="text-sm font-medium text-gray-900 mb-2">Role Description</h4>
                  <p className="text-sm text-gray-600">
                    {getRoleDescription(user.role)}
                  </p>
                </div>
              </div>
            </div>

            {/* Account Details */}
            <div className="space-y-6">
              <div className="bg-white border border-gray-200 rounded-lg p-4">
                <h4 className="text-sm font-medium text-gray-900 mb-3">Account Details</h4>
                <dl className="space-y-3">
                  <div>
                    <dt className="text-xs font-medium text-gray-500 uppercase tracking-wide">Member Since</dt>
                    <dd className="mt-1 text-sm text-gray-900">
                      {new Date(user.createdAt).toLocaleDateString('en-US', {
                        year: 'numeric',
                        month: 'long',
                        day: 'numeric'
                      })}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-xs font-medium text-gray-500 uppercase tracking-wide">Last Updated</dt>
                    <dd className="mt-1 text-sm text-gray-900">
                      {new Date(user.updatedAt).toLocaleDateString('en-US', {
                        year: 'numeric',
                        month: 'long',
                        day: 'numeric'
                      })}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-xs font-medium text-gray-500 uppercase tracking-wide">User ID</dt>
                    <dd className="mt-1 text-sm text-gray-900 font-mono">{user.id}</dd>
                  </div>
                </dl>
              </div>

              {/* Quick Actions */}
              <div className="bg-white border border-gray-200 rounded-lg p-4">
                <h4 className="text-sm font-medium text-gray-900 mb-3">Quick Actions</h4>
                <div className="space-y-2">
                  <Link
                    to="/"
                    className="block w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded-md"
                  >
                    🏠 Go to Dashboard
                  </Link>
                  <Link
                    to="/nodes"
                    className="block w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded-md"
                  >
                    🖥️ Manage Nodes
                  </Link>
                  {user.role === 'admin' && (
                    <Link
                      to="/admin"
                      className="block w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded-md"
                    >
                      ⚙️ Admin Panel
                    </Link>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="mt-8 pt-6 border-t border-gray-200 flex flex-col sm:flex-row gap-4">
            <Link
              to="/"
              className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              ← Back to Dashboard
            </Link>

            <div className="flex-1"></div>

            <button
              onClick={handleLogout}
              className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
            >
              🚪 Sign Out
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};