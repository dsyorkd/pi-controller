import React from 'react';
import { Link } from 'react-router-dom';
import { UserProfile } from '../components/auth/UserProfile';
import { useAuthContext } from '../components/auth/AuthProvider';

export const ProfilePage: React.FC = () => {
  const { user } = useAuthContext();

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <Link to="/dashboard" className="flex items-center">
                <div className="flex-shrink-0 text-2xl">🥧</div>
                <h1 className="ml-3 text-xl font-semibold text-gray-900">Pi Controller</h1>
              </Link>
            </div>
            <div className="flex items-center space-x-4">
              <span className="text-sm text-gray-600">Welcome, {user?.username}</span>
              <Link
                to="/dashboard"
                className="text-blue-600 hover:text-blue-500 text-sm font-medium"
              >
                Dashboard
              </Link>
            </div>
          </div>
        </div>
      </nav>

      <main className="max-w-3xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="mb-6">
            <h1 className="text-2xl font-bold text-gray-900">User Profile</h1>
            <p className="text-gray-600">Manage your account settings and preferences</p>
          </div>

          <UserProfile />
        </div>
      </main>
    </div>
  );
};