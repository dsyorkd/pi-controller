import React from 'react';
import { Link } from 'react-router-dom';
import { useAuthContext } from '../components/auth/AuthProvider';

export const Dashboard: React.FC = () => {
  const { user } = useAuthContext();

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <div className="flex-shrink-0 text-2xl">🥧</div>
              <h1 className="ml-3 text-xl font-semibold text-gray-900">Pi Controller</h1>
            </div>
            <div className="flex items-center space-x-4">
              <span className="text-sm text-gray-600">Welcome, {user?.username}</span>
              <Link
                to="/profile"
                className="text-blue-600 hover:text-blue-500 text-sm font-medium"
              >
                Profile
              </Link>
            </div>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {/* Clusters Card */}
            <div className="bg-white overflow-hidden shadow rounded-lg">
              <div className="p-5">
                <div className="flex items-center">
                  <div className="flex-shrink-0">
                    <div className="text-2xl">🌐</div>
                  </div>
                  <div className="ml-5 w-0 flex-1">
                    <dl>
                      <dt className="text-sm font-medium text-gray-500 truncate">
                        Clusters
                      </dt>
                      <dd className="text-lg font-medium text-gray-900">
                        Manage K3s clusters
                      </dd>
                    </dl>
                  </div>
                </div>
              </div>
              <div className="bg-gray-50 px-5 py-3">
                <div className="text-sm">
                  <a href="#" className="font-medium text-blue-600 hover:text-blue-500">
                    View all
                  </a>
                </div>
              </div>
            </div>

            {/* Nodes Card */}
            <div className="bg-white overflow-hidden shadow rounded-lg">
              <div className="p-5">
                <div className="flex items-center">
                  <div className="flex-shrink-0">
                    <div className="text-2xl">🖥️</div>
                  </div>
                  <div className="ml-5 w-0 flex-1">
                    <dl>
                      <dt className="text-sm font-medium text-gray-500 truncate">
                        Nodes
                      </dt>
                      <dd className="text-lg font-medium text-gray-900">
                        Raspberry Pi nodes
                      </dd>
                    </dl>
                  </div>
                </div>
              </div>
              <div className="bg-gray-50 px-5 py-3">
                <div className="text-sm">
                  <a href="#" className="font-medium text-blue-600 hover:text-blue-500">
                    View all
                  </a>
                </div>
              </div>
            </div>

            {/* GPIO Card */}
            <div className="bg-white overflow-hidden shadow rounded-lg">
              <div className="p-5">
                <div className="flex items-center">
                  <div className="flex-shrink-0">
                    <div className="text-2xl">🔌</div>
                  </div>
                  <div className="ml-5 w-0 flex-1">
                    <dl>
                      <dt className="text-sm font-medium text-gray-500 truncate">
                        GPIO Devices
                      </dt>
                      <dd className="text-lg font-medium text-gray-900">
                        Hardware control
                      </dd>
                    </dl>
                  </div>
                </div>
              </div>
              <div className="bg-gray-50 px-5 py-3">
                <div className="text-sm">
                  <a href="#" className="font-medium text-blue-600 hover:text-blue-500">
                    View all
                  </a>
                </div>
              </div>
            </div>
          </div>

          {/* Quick Stats */}
          <div className="mt-8">
            <h2 className="text-lg font-medium text-gray-900 mb-4">System Status</h2>
            <div className="bg-white shadow rounded-lg p-6">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="text-center">
                  <div className="text-3xl font-bold text-green-600">●</div>
                  <div className="text-sm text-gray-500">System Online</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-gray-900">0</div>
                  <div className="text-sm text-gray-500">Active Clusters</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-gray-900">0</div>
                  <div className="text-sm text-gray-500">Connected Nodes</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
};